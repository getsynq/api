package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	coordinatesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/coordinates/v1"
	customfeaturesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/features/v1"
	customv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// -----------------------------------------------------------------------------
// Identifiers
// -----------------------------------------------------------------------------
//
// You choose a custom entity's id, and it must never change. Every feature,
// relationship and declared edge is attached to it. If you change it, you create
// a second entity, and the first one is not renamed. So build it from what is
// STABLE in the source system (the object's id), never from its title. The kind
// is in the id only so that a person who reads a lineage graph can tell a model
// from a question.

func modelID(id int) *entitiesv1.Identifier { return customID(fmt.Sprintf("metabase::model::%d", id)) }
func questionID(id int) *entitiesv1.Identifier {
	return customID(fmt.Sprintf("metabase::question::%d", id))
}
func dashboardID(id int) *entitiesv1.Identifier {
	return customID(fmt.Sprintf("metabase::dashboard::%d", id))
}
func subscriptionID(id int) *entitiesv1.Identifier {
	return customID(fmt.Sprintf("metabase::subscription::%d", id))
}
func alertID(id int) *entitiesv1.Identifier { return customID(fmt.Sprintf("metabase::alert::%d", id)) }

func customID(id string) *entitiesv1.Identifier {
	return &entitiesv1.Identifier{
		Id: &entitiesv1.Identifier_Custom{Custom: &entitiesv1.CustomIdentifier{Id: id}},
	}
}

// warehouseTable names an entity that Coalesce Quality already has, from an
// integration you did not write. Every API here treats custom entities and native
// entities the same way: an edge, a binding or a declared column works between
// them with no extra steps.
func warehouseTable(cfg config, table string) *entitiesv1.Identifier {
	return &entitiesv1.Identifier{
		Id: &entitiesv1.Identifier_BigqueryTable{
			BigqueryTable: &entitiesv1.BigqueryTableIdentifier{
				Project: cfg.project,
				Dataset: cfg.dataset,
				Table:   table,
			},
		},
	}
}

// everyEntity is the full set this sync owns, in a stable order. The group step
// sends exactly this, which is what makes deletion in the BI tool propagate.
func everyEntity() []*entitiesv1.Identifier {
	var ids []*entitiesv1.Identifier
	for _, m := range models {
		ids = append(ids, modelID(m.id))
	}
	for _, q := range questions {
		ids = append(ids, questionID(q.id))
	}
	for _, d := range dashboards {
		ids = append(ids, dashboardID(d.id))
	}
	for _, s := range subscriptions {
		ids = append(ids, subscriptionID(s.id))
	}
	for _, a := range alerts {
		ids = append(ids, alertID(a.id))
	}
	return ids
}

// -----------------------------------------------------------------------------
// Step 1: types
// -----------------------------------------------------------------------------

// checkTypeIdsAreFree refuses to start if one of this example's type ids is
// already in use in the workspace under a different name. Type ids are
// workspace-wide and there is no allocator, so the only protection against two
// integrations picking the same number is to look before writing.
func checkTypeIdsAreFree(ctx context.Context, api *clients, _ config) error {
	resp, err := api.types.ListTypes(ctx, &customv1.ListTypesRequest{})
	if err != nil {
		return err
	}
	existing := map[int32]string{}
	for _, t := range resp.GetTypes() {
		existing[t.GetTypeId()] = t.GetName()
	}
	for _, want := range biTypes() {
		if name, taken := existing[want.GetTypeId()]; taken && name != want.GetName() {
			return fmt.Errorf(
				"type id %d is already used by %q: pick a different band in main.go",
				want.GetTypeId(), name,
			)
		}
	}
	fmt.Printf("   %d type ids free or already ours\n", len(biTypes()))
	return nil
}

// biTypes is the tool's object model, one Type per kind of thing.
//
// Traits are the part worth getting right. A type declares how its entities
// BEHAVE, and every entity of the type inherits it: is_model makes the BI
// models rank and behave as transformation models do (the same way a dbt model
// does), and is_bi_like marks the questions and dashboards as the leaves of the
// graph a business reads. Get these wrong and the entities still appear, but
// they sort, filter and rank as if they were something else.
func biTypes() []*entitiesv1.Type {
	return []*entitiesv1.Type{
		{
			TypeId:  typeModel,
			Name:    "Metabase Model",
			SvgIcon: iconLayers,
			Traits:  &entitiesv1.TypeTraits{IsModel: proto.Bool(true)},
		},
		{
			TypeId:  typeQuestion,
			Name:    "Metabase Question",
			SvgIcon: iconChart,
			// A question transforms data AND is read by a person, so it is both.
			Traits: &entitiesv1.TypeTraits{IsModel: proto.Bool(true), IsBiLike: proto.Bool(true)},
		},
		{
			TypeId:  typeDashboard,
			Name:    "Metabase Dashboard",
			SvgIcon: iconTiles,
			Traits:  &entitiesv1.TypeTraits{IsBiLike: proto.Bool(true)},
		},
		{
			TypeId:  typeSubscription,
			Name:    "Metabase Subscription",
			SvgIcon: iconEnvelope,
			Traits:  &entitiesv1.TypeTraits{IsBiLike: proto.Bool(true)},
		},
		{
			TypeId:  typeAlert,
			Name:    "Metabase Alert",
			SvgIcon: iconWarning,
			// A check, not a stage data flows through.
			Traits: &entitiesv1.TypeTraits{IsTestType: proto.Bool(true)},
		},
	}
}

func syncTypes(ctx context.Context, api *clients, _ config) error {
	for _, t := range biTypes() {
		if _, err := api.types.UpsertType(ctx, &customv1.UpsertTypeRequest{Type: t}); err != nil {
			return err
		}
		fmt.Printf("   type %d %s\n", t.GetTypeId(), t.GetName())
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 2: entities
// -----------------------------------------------------------------------------

// syncEntities creates every entity before anything points at one. Two later
// steps depend on that: a SQL binding to a custom entity that does not exist is
// rejected at the write, and an entity group whose members do not all exist is
// not a set anyone can reconcile against.
//
// Annotations are the tool's own filing system carried across, so the catalog
// can be filtered the way the BI tool is browsed. Keep them to labels a person
// would pick out of a list; prose belongs in the description.
func syncEntities(ctx context.Context, api *clients, cfg config) error {
	var entities []*entitiesv1.Entity

	for _, m := range models {
		entities = append(entities, &entitiesv1.Entity{
			Id:          modelID(m.id),
			TypeId:      typeModel,
			Name:        m.name,
			Description: m.description,
			Annotations: []*entitiesv1.Annotation{
				{Name: "metabase.collection", Values: []string{m.collection}},
				{Name: "metabase.kind", Values: []string{"model"}},
			},
		})
	}
	for _, q := range questions {
		entities = append(entities, &entitiesv1.Entity{
			Id:          questionID(q.id),
			TypeId:      typeQuestion,
			Name:        q.name,
			Description: q.description,
			Annotations: []*entitiesv1.Annotation{
				{Name: "metabase.collection", Values: []string{q.collection}},
				{Name: "metabase.kind", Values: []string{"question"}},
			},
		})
	}
	for _, d := range dashboards {
		entities = append(entities, &entitiesv1.Entity{
			Id:          dashboardID(d.id),
			TypeId:      typeDashboard,
			Name:        d.name,
			Description: d.description,
			Annotations: []*entitiesv1.Annotation{
				{Name: "metabase.collection", Values: []string{d.collection}},
				{Name: "metabase.kind", Values: []string{"dashboard"}},
			},
		})
	}
	for _, s := range subscriptions {
		entities = append(entities, &entitiesv1.Entity{
			Id:          subscriptionID(s.id),
			TypeId:      typeSubscription,
			Name:        s.name,
			Description: fmt.Sprintf("%s\n\nSchedule: %s.", s.description, s.schedule),
			Annotations: []*entitiesv1.Annotation{
				{Name: "metabase.kind", Values: []string{"subscription"}},
			},
		})
	}
	for _, a := range alerts {
		entities = append(entities, &entitiesv1.Entity{
			Id:          alertID(a.id),
			TypeId:      typeAlert,
			Name:        a.name,
			Description: a.description,
			Annotations: []*entitiesv1.Annotation{
				{Name: "metabase.kind", Values: []string{"alert"}},
			},
		})
	}

	for _, e := range entities {
		if _, err := api.entities.UpsertEntity(ctx, &customv1.UpsertEntityRequest{Entity: e}); err != nil {
			return err
		}
		fmt.Printf("   %s  %s\n", e.GetId().GetCustom().GetId(), e.GetName())
	}
	_ = cfg
	return nil
}

// -----------------------------------------------------------------------------
// Step 3: schemas
// -----------------------------------------------------------------------------

// syncSchemas declares what columns each entity has.
//
// This step is required, and it is the easiest one to skip. The declared schema
// lets the next entity downstream read the columns of this one: a `SELECT *`
// over a bound model expands only when the model's columns are known, and a
// declared column edge shows only for a column that the entity has. A dashboard
// with no schema still appears in table-level lineage, but it silently has no
// column lineage at all.
func syncSchemas(ctx context.Context, api *clients, _ config) error {
	put := func(id *entitiesv1.Identifier, cols []column) error {
		schema := &customfeaturesv1.Schema{
			StateAt: timestamppb.Now(),
			Columns: make([]*entitiesv1.SchemaColumn, 0, len(cols)),
		}
		for i, c := range cols {
			schema.Columns = append(schema.Columns, &entitiesv1.SchemaColumn{
				Name:            c.name,
				NativeType:      c.nativeType,
				Description:     c.desc,
				OrdinalPosition: int32(i + 1),
			})
		}
		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId: id,
				// One schema feature per entity, so the id is a constant. Never
				// generate it: a new id on every run creates a new feature each
				// time and does not replace the previous one.
				FeatureId: "schema",
				Feature:   &customv1.Feature_Schema{Schema: schema},
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("   %s  %d columns\n", id.GetCustom().GetId(), len(cols))
		return nil
	}

	for _, m := range models {
		if err := put(modelID(m.id), m.columns); err != nil {
			return err
		}
	}
	for _, q := range questions {
		if err := put(questionID(q.id), q.columns); err != nil {
			return err
		}
	}
	for _, d := range dashboards {
		cols := make([]column, 0, len(d.cards))
		for _, c := range d.cards {
			cols = append(cols, column{name: c.field, nativeType: "", desc: ""})
		}
		if err := put(dashboardID(d.id), cols); err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 4: model SQL, resolved by warehouse address
// -----------------------------------------------------------------------------

// warehouseContext is the default execution context of every query the BI tool
// runs: the database and schema an unqualified table name in its SQL resolves
// against. It is the tool's connection settings, expressed the way Coalesce
// Quality addresses a warehouse object.
//
// object_name is empty on purpose. It is for a definition that MATERIALIZES
// something (`CREATE TABLE x AS SELECT ...`), and nothing in a BI tool does. A
// model is a saved query, not a table.
func warehouseContext(cfg config) *coordinatesv1.DatabaseContext {
	return &coordinatesv1.DatabaseContext{
		// BigQuery has no instance above the project, so instance_name stays
		// empty. On Snowflake this would be the account, on Databricks the
		// workspace URL.
		DatabaseName: cfg.project,
		SchemaName:   cfg.dataset,
	}
}

// syncModelSql declares the models' SQL. Every table name in it is a real
// warehouse object, so it resolves by address and needs no binding. This is the
// simple case. Use it whenever it fits.
//
// Coalesce Quality parses the SQL and gets BOTH table and column lineage from
// it: the model appears downstream of `order_items` and `products`, and
// `net_revenue` traces back to `order_items.quantity`, `order_items.unit_price`
// and `order_items.discount`, and you state none of this. This is why you give
// SQL and do not declare edges: the lineage follows the SQL when the SQL changes.
func syncModelSql(ctx context.Context, api *clients, cfg config) error {
	for _, m := range models {
		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId:  modelID(m.id),
				FeatureId: "sql",
				Feature: &customv1.Feature_SqlDefinition{
					SqlDefinition: &customfeaturesv1.SqlDefinition{
						StateAt:         timestamppb.Now(),
						Dialect:         entitiesv1.SqlDialect_SQL_DIALECT_BIGQUERY,
						Sql:             m.sql,
						DatabaseContext: warehouseContext(cfg),
					},
				},
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("   %s  %d chars of SQL, no bindings\n", modelID(m.id).GetCustom().GetId(), len(m.sql))
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 5: question SQL, resolved by a binding
// -----------------------------------------------------------------------------

// syncQuestionSql declares the questions' SQL. This is the main step.
//
// A question reads a MODEL, and a model is not something the warehouse can
// address: it has no database, schema and table for a lookup to find. Without a
// binding, `FROM order_items_enriched` resolves to nothing and is dropped, and
// the question gets no upstream at all. There is no error, and the empty result
// looks the same as a query that reads nothing.
//
// `references` tells the definition what such a name means. Each binding pairs a
// name AS THE SQL WRITES IT with the entity that it stands for. The parse then
// gets table and column lineage from the SQL as usual: the table edge, and
// `revenue` traced through `order_items_enriched.net_revenue` down to the
// warehouse columns that the model computed it from.
//
// Three things to know:
//
//   - A binding has priority over the warehouse. If a real table has the same
//     name, the binding still resolves. So you can also use a binding as a
//     manual override.
//   - Empty parts of a binding default from the definition's own
//     database_context, exactly as an unqualified name in the SQL does. So a
//     bare object_name matches both `FROM order_items_enriched` and
//     `FROM <project>.<dataset>.order_items_enriched`.
//   - Names that ARE warehouse objects need no binding. Question 88 below reads
//     a model and a raw table in one query, and only the model is listed.
func syncQuestionSql(ctx context.Context, api *clients, cfg config) error {
	byID := map[int]model{}
	for _, m := range models {
		byID[m.id] = m
	}

	for _, q := range questions {
		// Sorted so two runs send byte-identical requests, which is what keeps a
		// re-sync from being mistaken for a change.
		names := make([]string, 0, len(q.sourceModels))
		for name := range q.sourceModels {
			names = append(names, name)
		}
		sort.Strings(names)

		refs := make([]*customfeaturesv1.SqlTableReference, 0, len(names))
		for _, name := range names {
			m, ok := byID[q.sourceModels[name]]
			if !ok {
				return fmt.Errorf("question %d binds %q to unknown model %d", q.id, name, q.sourceModels[name])
			}
			refs = append(refs, &customfeaturesv1.SqlTableReference{
				// The name exactly as the SQL writes it. Matched
				// case-insensitively; leave database_name and schema_name empty
				// to default them from database_context.
				ObjectName: name,
				Entity:     modelID(m.id),
			})
		}

		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId:  questionID(q.id),
				FeatureId: "sql",
				Feature: &customv1.Feature_SqlDefinition{
					SqlDefinition: &customfeaturesv1.SqlDefinition{
						StateAt:         timestamppb.Now(),
						Dialect:         entitiesv1.SqlDialect_SQL_DIALECT_BIGQUERY,
						Sql:             q.sql,
						DatabaseContext: warehouseContext(cfg),
						// The bindings are part of the definition and replace with
						// it: one dropped from a later write is gone, and the
						// lineage it produced is withdrawn.
						References: refs,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		for _, r := range refs {
			fmt.Printf("   %s  %q -> %s\n",
				questionID(q.id).GetCustom().GetId(), r.GetObjectName(), r.GetEntity().GetCustom().GetId())
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 6: dashboard column lineage, declared
// -----------------------------------------------------------------------------

// syncDashboardColumnLineage states the dashboard's column lineage outright.
//
// A dashboard runs no SQL: a tile takes a question's result and shows it,
// maybe under a different label. There is nothing for a parser to read, so you
// declare the result. The general rule: use SqlDefinition when the component has
// SQL, and ColumnLineage when it does not.
//
// The declaration is COMPLETE and replaces the previous one: an edge left out of
// a write is removed, and sending the same set again changes nothing. So you
// can safely build it again from the tool's metadata on every run, and you do
// not need to compare it with what you sent last time.
//
// Declaring a column edge also declares the table edge it implies, so the
// dashboard appears downstream of each question here without a relationship
// being written too.
func syncDashboardColumnLineage(ctx context.Context, api *clients, cfg config) error {
	for _, d := range dashboards {
		edges := make([]*customfeaturesv1.ColumnEdge, 0, len(d.cards))
		for _, c := range d.cards {
			upstream := questionID(c.fromQuestion)
			if c.fromWarehouseTable != "" {
				// An upstream on a connected platform works exactly the same way,
				// and it does not have to exist yet: the edge is remembered and
				// appears once the entity is next ingested.
				upstream = warehouseTable(cfg, c.fromWarehouseTable)
			}
			edges = append(edges, &customfeaturesv1.ColumnEdge{
				Upstream:       upstream,
				UpstreamColumn: c.fromColumn,
				// The two sides are named separately, so the names do not have to
				// match. Here the dashboard prefixes each field with its tile.
				Column: c.field,
			})
		}

		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId: dashboardID(d.id),
				// Only one column-lineage feature is allowed per entity, so a
				// stable id is how it is edited.
				FeatureId: "column-lineage",
				Feature: &customv1.Feature_ColumnLineage{
					ColumnLineage: &customfeaturesv1.ColumnLineage{
						StateAt: timestamppb.Now(),
						Edges:   edges,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("   %s  %d column edges declared\n", dashboardID(d.id).GetCustom().GetId(), len(edges))
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 7: the tool's own definitions, as code
// -----------------------------------------------------------------------------

// syncCode attaches each object's definition as it comes out of the BI tool.
// Coalesce Quality shows it beside the entity and tracks what changed between
// versions, which is how a question that quietly started filtering differently
// gets noticed.
//
// You can use Code and SqlDefinition together: SqlDefinition is parsed and gives
// lineage, and Code is displayed and gives history. The models below have both:
// the SQL for the graph, and the card definition for the diff.
//
// Unlike schema and SQL, an entity may carry several code features, so the
// feature id is the file name it stands for.
func syncCode(ctx context.Context, api *clients, _ config) error {
	for _, m := range models {
		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId:  modelID(m.id),
				FeatureId: fmt.Sprintf("card_%d.json", m.id),
				Feature: &customv1.Feature_Code{
					Code: &customfeaturesv1.Code{
						Name:     fmt.Sprintf("card_%d.json", m.id),
						CodeType: entitiesv1.CodeType_CODE_TYPE_JSON,
						Content:  m.definition,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("   %s  card_%d.json\n", modelID(m.id).GetCustom().GetId(), m.id)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 8: the serialized export in git
// -----------------------------------------------------------------------------

// syncGitFileReferences points each dashboard at the file its serialized export
// lives in, when the BI content is version-controlled (Metabase calls this
// serialization; most tools have an equivalent). Coalesce Quality then reads the
// file's history from the repository, so "what changed, and who changed it" is
// answered from the commit rather than from the tool's audit log.
//
// Skipped unless BI_EXPORT_GIT_REPO is set, because a reference to a repository
// that does not exist is worse than no reference.
func syncGitFileReferences(ctx context.Context, api *clients, cfg config) error {
	if cfg.gitRepo == "" {
		fmt.Println("   skipped: BI_EXPORT_GIT_REPO is not set")
		return nil
	}
	for _, d := range dashboards {
		path := fmt.Sprintf("collections/%s/dashboards/%d.yaml", d.collection, d.id)
		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId:  dashboardID(d.id),
				FeatureId: path,
				Feature: &customv1.Feature_GitFileReference{
					GitFileReference: &customfeaturesv1.GitFileReference{
						RepositoryUrl: cfg.gitRepo,
						BranchName:    cfg.gitBranch,
						FilePath:      path,
					},
				},
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("   %s  %s\n", dashboardID(d.id).GetCustom().GetId(), path)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 9: relationships with no columns
// -----------------------------------------------------------------------------

// syncRelationships joins each subscription to the dashboard it sends.
//
// A subscription has no columns of its own, because it delivers data and does
// not transform it. So there is nothing to state about columns, and a plain
// relationship is correct. Use one whenever the edge is real but there is no
// column mapping: a reverse-ETL job, an export, a downstream service.
//
// The response reports what each write DID. Read it, and do not assume: an edge
// is stored between the two entities that the endpoints RESOLVE to, and one
// entity accepts several spellings. So two relationships that look different can
// be the same edge.
func syncRelationships(ctx context.Context, api *clients, _ config) error {
	var rels []*customv1.Relationship
	for _, s := range subscriptions {
		rels = append(rels, &customv1.Relationship{
			Upstream:   dashboardID(s.dashboard),
			Downstream: subscriptionID(s.id),
		})
	}

	resp, err := api.relationships.UpsertRelationships(ctx, &customv1.UpsertRelationshipsRequest{
		Relationships: rels,
	})
	if err != nil {
		return err
	}
	for _, r := range resp.GetResults() {
		fmt.Printf("   %s -> %s  %s\n",
			r.GetRelationship().GetUpstream().GetEntityId(),
			r.GetRelationship().GetDownstream().GetEntityId(),
			r.GetOutcome(),
		)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 10: the alert as a check
// -----------------------------------------------------------------------------

// syncAlerts models the BI tool's alert as a CHECK on the question it watches,
// not as another node that data flows through. A check relationship is a
// different edge from a lineage edge: it says "this validates that". So the
// alert's state adds to the question's health, and the graph does not get
// longer.
//
// The categories are unset on purpose. The workspace's own categorisation rules
// set `category` (the technical kind of check) and `governance_category` (what
// the check is for), and a value sent here OVERRIDES those rules. It is tempting
// to derive one from `kind`, but then every rule that the workspace wrote
// silently stops applying, and nothing in the product explains why. Send what
// the tool reports (`package` and `kind`) and let the rules decide the rest.
func syncAlerts(ctx context.Context, api *clients, _ config) error {
	for _, a := range alerts {
		_, err := api.features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId:  alertID(a.id),
				FeatureId: "check",
				Feature: &customv1.Feature_CheckCategory{
					CheckCategory: &customfeaturesv1.CheckCategory{
						Package: "metabase",
						Kind:    a.kind,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		resp, err := api.checkRelationships.UpsertCheckRelationships(ctx, &customv1.UpsertCheckRelationshipsRequest{
			CheckRelationships: []*customv1.CheckRelationship{{
				Check:   alertID(a.id),
				Checked: questionID(a.question),
			}},
		})
		if err != nil {
			return err
		}
		for _, r := range resp.GetResults() {
			fmt.Printf("   %s checks %s  %s\n",
				r.GetCheckRelationship().GetCheck().GetEntityId(),
				r.GetCheckRelationship().GetChecked().GetEntityId(),
				r.GetOutcome(),
			)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 11: the group, which is what makes deletion work
// -----------------------------------------------------------------------------

// syncGroup sends the complete set of entities this integration owns.
//
// The server keeps the previous set, compares the two, and deletes what is not
// in the new set. This is why you use a group. Without one, an integration must
// remember on its own side what it created last time, so it can clean up after
// someone deletes a dashboard in the BI tool. Any record it keeps for that will
// drift from the real state.
//
// Send the FULL set on every run. A partial send deletes everything you leave
// out.
func syncGroup(ctx context.Context, api *clients, _ config) error {
	ids := everyEntity()
	resp, err := api.groups.UpsertEntitiesGroup(ctx, &customv1.UpsertEntitiesGroupRequest{
		Group: &customv1.Group{GroupId: groupID, EntityIds: ids},
	})
	if err != nil {
		return err
	}
	fmt.Printf("   group %q now holds %d entities\n", groupID, len(ids))
	for _, deleted := range resp.GetDeletedIds() {
		fmt.Printf("   deleted (gone from the BI tool): %s\n", deleted.GetCustom().GetId())
	}
	return nil
}

// -----------------------------------------------------------------------------
// Step 12: executions
// -----------------------------------------------------------------------------

// syncExecutions reports the last refresh of each dashboard. This gives a custom
// entity a status: fresh, stale or failing. Without it, the entity is a node in
// a graph with no known health.
//
// Report only what the tool tells you. `created_at` must be in the past, so do
// not report a refresh that has not happened yet.
func syncExecutions(ctx context.Context, api *clients, _ config) error {
	// A real integration takes this from the tool's own run history.
	finished := time.Now().Add(-15 * time.Minute)
	started := finished.Add(-42 * time.Second)

	for _, d := range dashboards {
		_, err := api.executions.UpsertExecution(ctx, &customv1.UpsertExecutionRequest{
			Execution: &customv1.Execution{
				Id:         dashboardID(d.id),
				Status:     customv1.ExecutionStatus_EXECUTION_STATUS_OK,
				Message:    "All cards refreshed.",
				CreatedAt:  timestamppb.New(finished),
				StartedAt:  timestamppb.New(started),
				FinishedAt: timestamppb.New(finished),
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("   %s  refreshed %s\n", dashboardID(d.id).GetCustom().GetId(), finished.Format(time.RFC3339))
	}
	return nil
}

// -----------------------------------------------------------------------------
// Icons
// -----------------------------------------------------------------------------
//
// Any SVG works. These are plain geometry so the example carries no licence
// question; swap in the BI tool's own mark when you adapt it.

var (
	iconLayers   = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M256 48 32 160l224 112 224-112L256 48z"/><path d="M32 240l224 112 224-112v64L256 416 32 304v-64z"/></svg>`)
	iconChart    = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M96 320h48v96H96zM200 240h48v176h-48zM304 160h48v256h-48zM408 272h48v144h-48zM48 448h416v32H48z"/></svg>`)
	iconTiles    = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M64 64h176v176H64zM272 64h176v96H272zM272 192h176v256H272zM64 272h176v176H64z"/></svg>`)
	iconEnvelope = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M32 96h448L256 256 32 96z"/><path d="M32 136l224 160 224-160v280H32V136z"/></svg>`)
	iconWarning  = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path fill-rule="evenodd" d="M256 48l224 400H32L256 48zm-24 128h48v128h-48V176zm0 168h48v48h-48v-48z"/></svg>`)
)
