"""
Modelling a BI tool Coalesce Quality has no integration for, end to end.

The estate this builds is four layers deep, and each hop uses a DIFFERENT
lineage mechanism, on purpose — picking the right one per hop is the whole skill
of modelling a new tool:

    warehouse tables                 (already in Coalesce Quality)
      |  SQL, resolved by warehouse address
      v
    BI models                        custom entities, SqlDefinition
      |  SQL, resolved by a BINDING — a model has no warehouse address
      v
    BI questions                     custom entities, SqlDefinition + references
      |  no SQL at all: declared column lineage
      v
    BI dashboard                     custom entity, ColumnLineage feature
      |  no columns: a plain relationship
      v
    BI subscription                  custom entity

    and off to one side, a BI alert   custom entity, CheckCategory feature
                                      attached with a check relationship

Read metabase.py first — it is the tool's own metadata, and the mapping below is
easier to follow once you know what is being mapped.

Everything written here is idempotent: re-running the script produces the same
estate, which is what lets it be a cron job rather than a migration.
"""

import os
import sys
import time

import grpc
from google.protobuf.timestamp_pb2 import Timestamp

from auth import TokenAuth, TokenSource
from metabase import ALERTS, DASHBOARDS, MODELS, QUESTIONS, SUBSCRIPTIONS, Column
from synq.entities.coordinates.v1 import database_context_pb2
from synq.entities.custom.features.v1 import checks_pb2, code_pb2, column_lineage_pb2
from synq.entities.custom.features.v1 import git_file_reference_pb2
from synq.entities.custom.features.v1 import schema_pb2 as feature_schema_pb2
from synq.entities.custom.features.v1 import sql_definition_pb2
from synq.entities.custom.v1 import (
    checks_relationships_service_pb2,
    checks_relationships_service_pb2_grpc,
    entities_service_pb2,
    entities_service_pb2_grpc,
    entity_executions_service_pb2,
    entity_executions_service_pb2_grpc,
    features_service_pb2,
    features_service_pb2_grpc,
    groups_service_pb2,
    groups_service_pb2_grpc,
    relationships_service_pb2,
    relationships_service_pb2_grpc,
    types_service_pb2,
    types_service_pb2_grpc,
)
from synq.entities.lineage.v1 import (
    lineage_direction_pb2,
    lineage_pb2,
    lineage_service_pb2,
    lineage_service_pb2_grpc,
)
from synq.entities.v1 import (
    annotation_pb2,
    code_type_pb2,
    entity_pb2,
    identifier_pb2,
    schema_pb2,
    sql_dialect_pb2,
    type_pb2,
    type_traits_pb2,
)

# Custom type ids are workspace-wide and yours to allocate: 1..1000, one number
# per kind of thing your tool has. Pick a free band and keep it — an entity's
# type is its id, so reusing a number silently reclassifies everything that had
# it. check_type_ids_are_free below refuses to start if one of these is already
# in use under another name.
TYPE_MODEL = 40
TYPE_QUESTION = 41
TYPE_DASHBOARD = 42
TYPE_SUBSCRIPTION = 43
TYPE_ALERT = 44

# The entity group every entity below is a member of. The group is what makes a
# re-sync self-cleaning: content deleted in the BI tool simply stops being sent,
# and the server deletes it. Nothing has to be remembered on this side between
# runs.
GROUP_ID = "metabase"

# Any SVG works. These are plain geometry so the example carries no licence
# question; swap in the BI tool's own mark when you adapt it.
ICON_LAYERS = (
    b'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">'
    b'<path d="M256 48 32 160l224 112 224-112L256 48z"/>'
    b'<path d="M32 240l224 112 224-112v64L256 416 32 304v-64z"/></svg>'
)
ICON_CHART = (
    b'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">'
    b'<path d="M96 320h48v96H96zM200 240h48v176h-48zM304 160h48v256h-48z'
    b'M408 272h48v144h-48zM48 448h416v32H48z"/></svg>'
)
ICON_TILES = (
    b'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">'
    b'<path d="M64 64h176v176H64zM272 64h176v96H272zM272 192h176v256H272z'
    b'M64 272h176v176H64z"/></svg>'
)
ICON_ENVELOPE = (
    b'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">'
    b'<path d="M32 96h448L256 256 32 96z"/>'
    b'<path d="M32 136l224 160 224-160v280H32V136z"/></svg>'
)
ICON_WARNING = (
    b'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">'
    b'<path fill-rule="evenodd" d="M256 48l224 400H32L256 48zm-24 128h48v128h-48V176z'
    b'm0 168h48v48h-48v-48z"/></svg>'
)


class Config:
    def __init__(self):
        # developer.synq.io (EU) is the default. The other deployments are
        # api.us.synq.io (US) and api.au.synq.io (AU); set QUALITY_API_ENDPOINT
        # to the one your workspace lives in.
        self.endpoint = os.environ.get("QUALITY_API_ENDPOINT") or "developer.synq.io"

        # The warehouse the BI tool queries. These name the DEFAULT execution
        # context of every query the tool runs, which is what lets an unqualified
        # table name in the SQL resolve to a real warehouse object.
        self.project = os.environ.get("BIGQUERY_PROJECT") or "my-gcp-project"
        self.dataset = os.environ.get("BIGQUERY_DATASET") or "analytics"

        # Optional: the repository a serialized export of the BI content is
        # committed to, so Coalesce Quality can show its history. Leave unset to
        # skip that step.
        self.git_repo = os.environ.get("BI_EXPORT_GIT_REPO") or ""
        self.git_branch = os.environ.get("BI_EXPORT_GIT_BRANCH") or "main"


class Clients:
    def __init__(self, channel):
        self.types = types_service_pb2_grpc.TypesServiceStub(channel)
        self.entities = entities_service_pb2_grpc.EntitiesServiceStub(channel)
        self.features = features_service_pb2_grpc.FeaturesServiceStub(channel)
        self.relationships = relationships_service_pb2_grpc.RelationshipsServiceStub(channel)
        self.check_relationships = (
            checks_relationships_service_pb2_grpc.ChecksRelationshipsServiceStub(channel)
        )
        self.groups = groups_service_pb2_grpc.GroupsServiceStub(channel)
        self.executions = (
            entity_executions_service_pb2_grpc.EntityExecutionsServiceStub(channel)
        )
        self.lineage = lineage_service_pb2_grpc.LineageServiceStub(channel)


# -----------------------------------------------------------------------------
# Identifiers
# -----------------------------------------------------------------------------
#
# A custom entity's id is yours, and it is the one thing that must never change:
# every feature, relationship and declared edge hangs off it, and changing it
# creates a second entity rather than renaming the first. So build it from what
# is STABLE in the source system — the object's id — and never from its title.
# The kind is in the id purely so a human reading a lineage graph can tell a
# model from a question.


def custom_id(id_: str) -> identifier_pb2.Identifier:
    return identifier_pb2.Identifier(custom=identifier_pb2.CustomIdentifier(id=id_))


def model_id(id_: int) -> identifier_pb2.Identifier:
    return custom_id(f"metabase::model::{id_}")


def question_id(id_: int) -> identifier_pb2.Identifier:
    return custom_id(f"metabase::question::{id_}")


def dashboard_id(id_: int) -> identifier_pb2.Identifier:
    return custom_id(f"metabase::dashboard::{id_}")


def subscription_id(id_: int) -> identifier_pb2.Identifier:
    return custom_id(f"metabase::subscription::{id_}")


def alert_id(id_: int) -> identifier_pb2.Identifier:
    return custom_id(f"metabase::alert::{id_}")


def warehouse_table(cfg: Config, table: str) -> identifier_pb2.Identifier:
    """
    Name an entity Coalesce Quality already has, from an integration you did not
    write. Custom entities and native ones are the same kind of thing to every
    API here: an edge, a binding or a declared column works across the boundary
    without ceremony.
    """
    return identifier_pb2.Identifier(
        bigquery_table=identifier_pb2.BigqueryTableIdentifier(
            project=cfg.project, dataset=cfg.dataset, table=table
        )
    )


def every_entity():
    """
    The full set this sync owns, in a stable order. The group step sends exactly
    this, which is what makes deletion in the BI tool propagate.
    """
    ids = [model_id(m.id) for m in MODELS]
    ids += [question_id(q.id) for q in QUESTIONS]
    ids += [dashboard_id(d.id) for d in DASHBOARDS]
    ids += [subscription_id(s.id) for s in SUBSCRIPTIONS]
    ids += [alert_id(a.id) for a in ALERTS]
    return ids


def now() -> Timestamp:
    ts = Timestamp()
    ts.GetCurrentTime()
    return ts


# -----------------------------------------------------------------------------
# Step 1 — types
# -----------------------------------------------------------------------------


def bi_types():
    """
    The tool's object model, one Type per kind of thing.

    Traits are the part worth getting right. A type declares how its entities
    BEHAVE, and every entity of the type inherits it: is_model makes the BI
    models rank and behave as transformation models do (the same way a dbt model
    does), and is_bi_like marks the questions and dashboards as the leaves of the
    graph a business reads. Get these wrong and the entities still appear, but
    they sort, filter and rank as if they were something else.
    """
    return [
        type_pb2.Type(
            type_id=TYPE_MODEL,
            name="Metabase Model",
            svg_icon=ICON_LAYERS,
            traits=type_traits_pb2.TypeTraits(is_model=True),
        ),
        type_pb2.Type(
            type_id=TYPE_QUESTION,
            name="Metabase Question",
            svg_icon=ICON_CHART,
            # A question transforms data AND is read by a person, so it is both.
            traits=type_traits_pb2.TypeTraits(is_model=True, is_bi_like=True),
        ),
        type_pb2.Type(
            type_id=TYPE_DASHBOARD,
            name="Metabase Dashboard",
            svg_icon=ICON_TILES,
            traits=type_traits_pb2.TypeTraits(is_bi_like=True),
        ),
        type_pb2.Type(
            type_id=TYPE_SUBSCRIPTION,
            name="Metabase Subscription",
            svg_icon=ICON_ENVELOPE,
            traits=type_traits_pb2.TypeTraits(is_bi_like=True),
        ),
        type_pb2.Type(
            type_id=TYPE_ALERT,
            name="Metabase Alert",
            svg_icon=ICON_WARNING,
            # A check, not a stage data flows through.
            traits=type_traits_pb2.TypeTraits(is_test_type=True),
        ),
    ]


def check_type_ids_are_free(api: Clients, cfg: Config):
    """
    Refuse to start if one of this example's type ids is already in use in the
    workspace under a different name. Type ids are workspace-wide and there is no
    allocator, so the only protection against two integrations picking the same
    number is to look before writing.
    """
    resp = api.types.ListTypes(types_service_pb2.ListTypesRequest())
    existing = {t.type_id: t.name for t in resp.types}
    for want in bi_types():
        taken = existing.get(want.type_id)
        if taken is not None and taken != want.name:
            raise RuntimeError(
                f"type id {want.type_id} is already used by {taken!r}: "
                "pick a different band at the top of sync.py"
            )
    print(f"   {len(bi_types())} type ids free or already ours")


def sync_types(api: Clients, cfg: Config):
    for t in bi_types():
        api.types.UpsertType(types_service_pb2.UpsertTypeRequest(type=t))
        print(f"   type {t.type_id} {t.name}")


# -----------------------------------------------------------------------------
# Step 2 — entities
# -----------------------------------------------------------------------------


def sync_entities(api: Clients, cfg: Config):
    """
    Create every entity before anything points at one. Two later steps depend on
    that: a SQL binding to a custom entity that does not exist is rejected at the
    write, and an entity group whose members do not all exist is not a set anyone
    can reconcile against.

    Annotations are the tool's own filing system carried across, so the catalog
    can be filtered the way the BI tool is browsed. Keep them to labels a person
    would pick out of a list; prose belongs in the description.
    """
    entities = []

    for m in MODELS:
        entities.append(
            entity_pb2.Entity(
                id=model_id(m.id),
                type_id=TYPE_MODEL,
                name=m.name,
                description=m.description,
                annotations=[
                    annotation_pb2.Annotation(name="metabase.collection", values=[m.collection]),
                    annotation_pb2.Annotation(name="metabase.kind", values=["model"]),
                ],
            )
        )
    for q in QUESTIONS:
        entities.append(
            entity_pb2.Entity(
                id=question_id(q.id),
                type_id=TYPE_QUESTION,
                name=q.name,
                description=q.description,
                annotations=[
                    annotation_pb2.Annotation(name="metabase.collection", values=[q.collection]),
                    annotation_pb2.Annotation(name="metabase.kind", values=["question"]),
                ],
            )
        )
    for d in DASHBOARDS:
        entities.append(
            entity_pb2.Entity(
                id=dashboard_id(d.id),
                type_id=TYPE_DASHBOARD,
                name=d.name,
                description=d.description,
                annotations=[
                    annotation_pb2.Annotation(name="metabase.collection", values=[d.collection]),
                    annotation_pb2.Annotation(name="metabase.kind", values=["dashboard"]),
                ],
            )
        )
    for s in SUBSCRIPTIONS:
        entities.append(
            entity_pb2.Entity(
                id=subscription_id(s.id),
                type_id=TYPE_SUBSCRIPTION,
                name=s.name,
                description=f"{s.description}\n\nSchedule: {s.schedule}.",
                annotations=[
                    annotation_pb2.Annotation(name="metabase.kind", values=["subscription"]),
                ],
            )
        )
    for a in ALERTS:
        entities.append(
            entity_pb2.Entity(
                id=alert_id(a.id),
                type_id=TYPE_ALERT,
                name=a.name,
                description=a.description,
                annotations=[annotation_pb2.Annotation(name="metabase.kind", values=["alert"])],
            )
        )

    for e in entities:
        api.entities.UpsertEntity(entities_service_pb2.UpsertEntityRequest(entity=e))
        print(f"   {e.id.custom.id}  {e.name}")


# -----------------------------------------------------------------------------
# Step 3 — schemas
# -----------------------------------------------------------------------------


def sync_schemas(api: Clients, cfg: Config):
    """
    Declare what columns each entity has.

    This is not decoration, and it is the step most easily skipped. A declared
    schema is what lets the next entity down read the columns of this one: a
    `SELECT *` over a bound model expands only over a model whose columns are
    known, and a declared column edge only shows against a column the entity
    actually has. A dashboard with no schema still appears in table-level
    lineage, and silently carries no column lineage at all.
    """

    def put(entity_id, columns):
        schema = feature_schema_pb2.Schema(
            state_at=now(),
            columns=[
                schema_pb2.SchemaColumn(
                    name=c.name,
                    native_type=c.native_type,
                    description=c.desc,
                    ordinal_position=i + 1,
                )
                for i, c in enumerate(columns)
            ],
        )
        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=entity_id,
                    # One schema feature per entity, so the id is a constant.
                    # Never generate it — a fresh id on every run creates a new
                    # feature each time instead of replacing the previous one.
                    feature_id="schema",
                    schema=schema,
                )
            )
        )
        print(f"   {entity_id.custom.id}  {len(columns)} columns")

    for m in MODELS:
        put(model_id(m.id), m.columns)
    for q in QUESTIONS:
        put(question_id(q.id), q.columns)
    for d in DASHBOARDS:
        put(dashboard_id(d.id), [Column(name=c.field) for c in d.cards])


# -----------------------------------------------------------------------------
# Step 4 — model SQL, resolved by warehouse address
# -----------------------------------------------------------------------------


def warehouse_context(cfg: Config) -> database_context_pb2.DatabaseContext:
    """
    The default execution context of every query the BI tool runs: the database
    and schema an unqualified table name in its SQL resolves against. It is the
    tool's connection settings, expressed the way Coalesce Quality addresses a
    warehouse object.

    object_name is deliberately left empty. It is for a definition that
    MATERIALIZES something — `CREATE TABLE x AS SELECT ...` — and nothing in a BI
    tool does; a model is a saved query, not a table.
    """
    return database_context_pb2.DatabaseContext(
        # BigQuery has no instance above the project, so instance_name stays
        # empty. On Snowflake this would be the account, on Databricks the
        # workspace URL.
        database_name=cfg.project,
        schema_name=cfg.dataset,
    )


def sync_model_sql(api: Clients, cfg: Config):
    """
    Declare the models' SQL. Every table name in it is a real warehouse object,
    so it resolves by address and no binding is needed — this is the plain case,
    and the one to reach for whenever it fits.

    Coalesce Quality parses the SQL and derives BOTH grains from it: the model
    appears downstream of `order_items` and `products`, and `net_revenue` traces
    back to `order_items.quantity`, `order_items.unit_price` and
    `order_items.discount` without any of that being stated. That is the whole
    argument for giving SQL rather than declaring edges: the lineage follows the
    SQL when the SQL changes.
    """
    for m in MODELS:
        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=model_id(m.id),
                    feature_id="sql",
                    sql_definition=sql_definition_pb2.SqlDefinition(
                        state_at=now(),
                        dialect=sql_dialect_pb2.SQL_DIALECT_BIGQUERY,
                        sql=m.sql,
                        database_context=warehouse_context(cfg),
                    ),
                )
            )
        )
        print(f"   {model_id(m.id).custom.id}  {len(m.sql)} chars of SQL, no bindings")


# -----------------------------------------------------------------------------
# Step 5 — question SQL, resolved by a binding
# -----------------------------------------------------------------------------


def sync_question_sql(api: Clients, cfg: Config):
    """
    Declare the questions' SQL, and this is the headline.

    A question reads a MODEL, and a model is not something the warehouse can
    address: it has no database, schema and table for a lookup to land on. Left
    alone, `FROM order_items_enriched` resolves to nothing, is dropped, and the
    question ends up with no upstream at all — with no error anywhere, and an
    empty result indistinguishable from a query that genuinely reads nothing.

    `references` is how the definition says what such a name means. Each binding
    pairs a name AS THE SQL WRITES IT with the entity it stands for, and the
    parse then derives both grains from the SQL as usual: the table edge, and
    `revenue` tracing through `order_items_enriched.net_revenue` all the way down
    to the warehouse columns the model computed it from.

    Three things worth knowing:

      - A binding wins over the warehouse. If a real table happened to share the
        name, the binding is still what resolves — which is what makes it usable
        as a manual override, not just a fallback.
      - Empty parts of a binding default from the definition's own
        database_context, exactly as an unqualified name in the SQL does. So a
        bare object_name matches both `FROM order_items_enriched` and
        `FROM <project>.<dataset>.order_items_enriched`.
      - Names that ARE warehouse objects need no binding. Question 88 reads a
        model and a raw table in one query, and only the model is listed.
    """
    by_id = {m.id: m for m in MODELS}

    for q in QUESTIONS:
        refs = []
        # Sorted so two runs send byte-identical requests, which is what keeps a
        # re-sync from being mistaken for a change.
        for name in sorted(q.source_models):
            model = by_id.get(q.source_models[name])
            if model is None:
                raise RuntimeError(
                    f"question {q.id} binds {name!r} to unknown model {q.source_models[name]}"
                )
            refs.append(
                sql_definition_pb2.SqlTableReference(
                    # The name exactly as the SQL writes it. Matched
                    # case-insensitively; leave database_name and schema_name
                    # empty to default them from database_context.
                    object_name=name,
                    entity=model_id(model.id),
                )
            )

        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=question_id(q.id),
                    feature_id="sql",
                    sql_definition=sql_definition_pb2.SqlDefinition(
                        state_at=now(),
                        dialect=sql_dialect_pb2.SQL_DIALECT_BIGQUERY,
                        sql=q.sql,
                        database_context=warehouse_context(cfg),
                        # The bindings are part of the definition and replace
                        # with it: one dropped from a later write is gone, and
                        # the lineage it produced is withdrawn.
                        references=refs,
                    ),
                )
            )
        )
        for r in refs:
            print(f"   {question_id(q.id).custom.id}  {r.object_name!r} -> {r.entity.custom.id}")


# -----------------------------------------------------------------------------
# Step 6 — dashboard column lineage, declared
# -----------------------------------------------------------------------------


def sync_dashboard_column_lineage(api: Clients, cfg: Config):
    """
    State the dashboard's column lineage outright.

    A dashboard runs no SQL: a tile takes a question's result and shows it,
    possibly under a different label. There is nothing for a parser to read, so
    the answer is declared instead of derived. That is the rule of thumb for the
    whole exercise — SqlDefinition when the component has SQL, ColumnLineage when
    it does not.

    The declaration is COMPLETE and replaces the previous one whole: an edge left
    out of a write is withdrawn, and re-sending the same set changes nothing.
    That is what makes it safe to regenerate from the tool's metadata on every
    run rather than diffing against what was sent last time.

    Declaring a column edge also declares the table edge it implies, so the
    dashboard appears downstream of each question here without a relationship
    being written too.
    """
    for d in DASHBOARDS:
        edges = []
        for c in d.cards:
            if c.from_warehouse_table:
                # An upstream on a connected platform works exactly the same way,
                # and it does not have to exist yet: the edge is remembered and
                # appears once the entity is next ingested.
                upstream = warehouse_table(cfg, c.from_warehouse_table)
            else:
                upstream = question_id(c.from_question)
            edges.append(
                column_lineage_pb2.ColumnEdge(
                    upstream=upstream,
                    upstream_column=c.from_column,
                    # The two sides are named independently, so they need not
                    # match — here the dashboard prefixes each field with the
                    # tile it sits on.
                    column=c.field,
                )
            )

        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=dashboard_id(d.id),
                    # Only one column-lineage feature is allowed per entity, so a
                    # stable id is how it is edited.
                    feature_id="column-lineage",
                    column_lineage=column_lineage_pb2.ColumnLineage(state_at=now(), edges=edges),
                )
            )
        )
        print(f"   {dashboard_id(d.id).custom.id}  {len(edges)} column edges declared")


# -----------------------------------------------------------------------------
# Step 7 — the tool's own definitions, as code
# -----------------------------------------------------------------------------


def sync_code(api: Clients, cfg: Config):
    """
    Attach each object's definition as it comes out of the BI tool. Coalesce
    Quality shows it beside the entity and tracks what changed between versions,
    which is how a question that quietly started filtering differently gets
    noticed.

    Code and SqlDefinition are not alternatives: SqlDefinition is parsed and
    produces lineage, Code is displayed and produces history. The models below
    have both — the SQL for the graph, the card definition for the diff.

    Unlike schema and SQL, an entity may carry several code features, so the
    feature id is the file name it stands for.
    """
    for m in MODELS:
        name = f"card_{m.id}.json"
        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=model_id(m.id),
                    feature_id=name,
                    code=code_pb2.Code(
                        name=name,
                        code_type=code_type_pb2.CODE_TYPE_JSON,
                        content=m.definition,
                    ),
                )
            )
        )
        print(f"   {model_id(m.id).custom.id}  {name}")


# -----------------------------------------------------------------------------
# Step 8 — the serialized export in git
# -----------------------------------------------------------------------------


def sync_git_file_references(api: Clients, cfg: Config):
    """
    Point each dashboard at the file its serialized export lives in, when the BI
    content is version-controlled (Metabase calls this serialization; most tools
    have an equivalent). Coalesce Quality then reads the file's history from the
    repository, so "what changed, and who changed it" is answered from the commit
    rather than from the tool's audit log.

    Skipped unless BI_EXPORT_GIT_REPO is set, because a reference to a repository
    that does not exist is worse than no reference.
    """
    if not cfg.git_repo:
        print("   skipped: BI_EXPORT_GIT_REPO is not set")
        return
    for d in DASHBOARDS:
        path = f"collections/{d.collection}/dashboards/{d.id}.yaml"
        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=dashboard_id(d.id),
                    feature_id=path,
                    git_file_reference=git_file_reference_pb2.GitFileReference(
                        repository_url=cfg.git_repo,
                        branch_name=cfg.git_branch,
                        file_path=path,
                    ),
                )
            )
        )
        print(f"   {dashboard_id(d.id).custom.id}  {path}")


# -----------------------------------------------------------------------------
# Step 9 — relationships with no columns
# -----------------------------------------------------------------------------


def sync_relationships(api: Clients, cfg: Config):
    """
    Join each subscription to the dashboard it sends.

    A subscription has no columns of its own — it is a delivery, not a
    transformation — so there is nothing to state at column grain and a plain
    relationship is the right shape. Reach for one whenever the edge is real but
    the column mapping is not: a reverse-ETL job, an export, a downstream
    service.

    The response reports what each write DID. Read it rather than assuming: an
    edge is held between the two entities the endpoints RESOLVE to, and one
    entity accepts several spellings, so two relationships that look different
    can be the same edge.
    """
    rels = [
        relationships_service_pb2.Relationship(
            upstream=dashboard_id(s.dashboard), downstream=subscription_id(s.id)
        )
        for s in SUBSCRIPTIONS
    ]
    resp = api.relationships.UpsertRelationships(
        relationships_service_pb2.UpsertRelationshipsRequest(relationships=rels)
    )
    for r in resp.results:
        outcome = relationships_service_pb2.RelationshipWriteOutcome.Name(r.outcome)
        print(
            f"   {r.relationship.upstream.entity_id} -> "
            f"{r.relationship.downstream.entity_id}  {outcome}"
        )


# -----------------------------------------------------------------------------
# Step 10 — the alert as a check
# -----------------------------------------------------------------------------


def sync_alerts(api: Clients, cfg: Config):
    """
    Model the BI tool's alert as a CHECK on the question it watches, rather than
    as another node data flows through. A check relationship is a different edge
    from a lineage one: it says "this validates that", so the alert's state rolls
    up to the question's health instead of extending the graph.

    The categories are deliberately left unset. `category` (what kind of check
    this is, mechanically) and `governance_category` (what the check is for) are
    resolved by the workspace's own categorisation rules, and a value sent here
    OUTRANKS those rules. Deriving one from `kind` — which is the obvious thing
    to do, and wrong — would silently switch off every rule the workspace wrote,
    with nothing in the product explaining why. Send what the tool actually says
    (`package` and `kind`) and let the rules decide the rest.
    """
    for a in ALERTS:
        api.features.UpsertEntityFeature(
            features_service_pb2.UpsertEntityFeatureRequest(
                feature=features_service_pb2.Feature(
                    entity_id=alert_id(a.id),
                    feature_id="check",
                    check_category=checks_pb2.CheckCategory(package="metabase", kind=a.kind),
                )
            )
        )

        resp = api.check_relationships.UpsertCheckRelationships(
            checks_relationships_service_pb2.UpsertCheckRelationshipsRequest(
                check_relationships=[
                    checks_relationships_service_pb2.CheckRelationship(
                        check=alert_id(a.id), checked=question_id(a.question)
                    )
                ]
            )
        )
        for r in resp.results:
            outcome = relationships_service_pb2.RelationshipWriteOutcome.Name(r.outcome)
            print(
                f"   {r.check_relationship.check.entity_id} checks "
                f"{r.check_relationship.checked.entity_id}  {outcome}"
            )


# -----------------------------------------------------------------------------
# Step 11 — the group, which is what makes deletion work
# -----------------------------------------------------------------------------


def sync_group(api: Clients, cfg: Config):
    """
    Send the complete set of entities this integration owns.

    The server holds the previous set, diffs it, and deletes what is no longer
    there. That is the whole reason to use a group: without one, an integration
    has to remember on its own side what it created last time in order to clean
    up after a dashboard someone deleted in the BI tool — and any state it keeps
    for that will eventually disagree with reality.

    Send the FULL set every run. A partial send is not an update, it is a
    deletion of everything omitted.
    """
    ids = every_entity()
    resp = api.groups.UpsertEntitiesGroup(
        groups_service_pb2.UpsertEntitiesGroupRequest(
            group=groups_service_pb2.Group(group_id=GROUP_ID, entity_ids=ids)
        )
    )
    print(f"   group {GROUP_ID!r} now holds {len(ids)} entities")
    for deleted in resp.deleted_ids:
        print(f"   deleted (gone from the BI tool): {deleted.custom.id}")


# -----------------------------------------------------------------------------
# Step 12 — executions
# -----------------------------------------------------------------------------


def sync_executions(api: Clients, cfg: Config):
    """
    Report each dashboard's last refresh, which is what gives a custom entity a
    status: fresh, stale, failing. Without one it is a node in a graph with
    nothing known about its health.

    Report what the tool tells you and nothing more. `created_at` must be in the
    past, so a refresh that has not happened yet is simply not reported.
    """
    # A real integration takes this from the tool's own run history.
    finished = Timestamp()
    finished.FromSeconds(int(time.time()) - 15 * 60)
    started = Timestamp()
    started.FromSeconds(finished.seconds - 42)

    for d in DASHBOARDS:
        api.executions.UpsertExecution(
            entity_executions_service_pb2.UpsertExecutionRequest(
                execution=entity_executions_service_pb2.Execution(
                    id=dashboard_id(d.id),
                    status=entity_executions_service_pb2.EXECUTION_STATUS_OK,
                    message="All cards refreshed.",
                    created_at=finished,
                    started_at=started,
                    finished_at=finished,
                )
            )
        )
        print(f"   {dashboard_id(d.id).custom.id}  refreshed {finished.ToJsonString()}")


# -----------------------------------------------------------------------------
# Verify
# -----------------------------------------------------------------------------


def node_name(node) -> str:
    """
    Prefer the identifier shape that says what the node IS. A node carries every
    identifier that resolves to it, so a warehouse table reached through a custom
    entity's binding may list several.
    """
    for i in node.ids:
        if i.HasField("custom"):
            return i.custom.id
        if i.HasField("bigquery_table"):
            b = i.bigquery_table
            return f"{b.project}.{b.dataset}.{b.table}"
    return node.ids[0].entity_id if node.ids else "?"


def get_lineage(api: Clients, start_point):
    try:
        resp = api.lineage.GetLineage(
            lineage_service_pb2.GetLineageRequest(
                lineage_direction=lineage_direction_pb2.LINEAGE_DIRECTION_UPSTREAM,
                start_point=start_point,
                max_depth=10,
            )
        )
        return resp.lineage
    except grpc.RpcError as e:
        print(f"   (lineage read failed: {e.details()})")
        return None


def poll(budget_seconds, once):
    deadline = time.time() + budget_seconds
    while True:
        lineage, done = once()
        if done:
            return lineage
        if time.time() > deadline:
            print(f"   (gave up after {budget_seconds}s — the graph may still be building)")
            return lineage
        time.sleep(10)


def verify(api: Clients, cfg: Config, budget_seconds=120):
    """
    Read the graph back.

    None of this is needed to write the estate — it is here because the way this
    goes wrong is silent. A missing binding, a schema that was never declared, a
    column edge whose table edge is absent: each of them leaves a valid-looking
    write and an empty result. Reading the lineage back after a sync is the only
    thing that distinguishes "no upstreams" from "upstreams that did not
    resolve".

    Lineage is computed asynchronously from what was written: the SQL is parsed,
    bindings are resolved, the graph is rebuilt. So this polls rather than asking
    once.
    """
    dash = dashboard_id(DASHBOARDS[0].id)

    # The table-level chain. Expect the dashboard, two questions, two models and
    # the three warehouse tables under them.
    want_nodes = 7
    print(f"\n-- table lineage upstream of {dash.custom.id}")

    def table_once():
        lineage = get_lineage(
            api,
            lineage_service_pb2.GetLineageStartPoint(
                entities=lineage_service_pb2.EntitiesStartPoint(entities=[dash])
            ),
        )
        return lineage, lineage is not None and len(lineage.nodes) >= want_nodes

    table = poll(budget_seconds, table_once)
    if table is not None:
        for i, n in enumerate(table.nodes):
            position = lineage_pb2.NodePosition.Name(n.position).removeprefix("NODE_POSITION_")
            print(f"   [{i}] {position:<14} {node_name(n)}")
        for d in table.node_dependencies:
            print(
                f"   {node_name(table.nodes[d.source_node_idx])} -> "
                f"{node_name(table.nodes[d.target_node_idx])}"
            )

    # The column-level chain, which is the claim the whole example is making: a
    # field on a dashboard that runs no SQL traces through a question and a
    # model, neither of which the warehouse can address, to the physical column
    # the number came from.
    col = "revenue_by_category.revenue"
    print(f"\n-- column lineage upstream of {dash.custom.id}.{col}")

    def cll_once():
        lineage = get_lineage(
            api,
            lineage_service_pb2.GetLineageStartPoint(
                entity_columns=lineage_service_pb2.EntityColumnsStartPoint(
                    id=dash, column_names=[col]
                )
            ),
        )
        return lineage, lineage is not None and len(lineage.column_dependencies) > 0

    cll = poll(budget_seconds, cll_once)
    if cll is not None:
        for d in cll.column_dependencies:
            print(
                f"   {node_name(cll.nodes[d.source_node_idx])}.{d.source_node_column_id} -> "
                f"{node_name(cll.nodes[d.target_node_idx])}.{d.target_node_column_id}"
            )


# -----------------------------------------------------------------------------
# Entry point
# -----------------------------------------------------------------------------

# The order below is not cosmetic. Three of these steps depend on an earlier one
# having landed:
#
#  - An entity carries a type id, so the types go first.
#  - A binding to a custom entity that does not exist is REFUSED, so every entity
#    exists before any SQL definition binds to one.
#  - A `SELECT *` over a bound model expands only if that model has declared its
#    columns, so schemas are written before the SQL that reads them.
#
# The last one is invisible when it is wrong: the table-level edge still appears,
# only the column-level edges are missing.
STEPS = [
    ("check type ids are free", check_type_ids_are_free),
    ("declare entity types", sync_types),
    ("declare entities", sync_entities),
    ("declare schemas", sync_schemas),
    ("declare model SQL (resolved by warehouse address)", sync_model_sql),
    ("declare question SQL (resolved by binding)", sync_question_sql),
    ("declare dashboard column lineage", sync_dashboard_column_lineage),
    ("attach the tool's own definitions as code", sync_code),
    ("link the serialized export to git", sync_git_file_references),
    ("join the subscription to its dashboard", sync_relationships),
    ("declare the alert as a check", sync_alerts),
    ("reconcile the entity group", sync_group),
    ("report the last refresh of each dashboard", sync_executions),
]


def main():
    cfg = Config()

    client_id = os.environ.get("QUALITY_CLIENT_ID") or os.environ.get("SYNQ_CLIENT_ID")
    client_secret = os.environ.get("QUALITY_CLIENT_SECRET") or os.environ.get(
        "SYNQ_CLIENT_SECRET"
    )
    if not client_id or not client_secret:
        print("set QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET (see README.md)")
        sys.exit(1)

    token_source = TokenSource(client_id, client_secret, cfg.endpoint)
    channel = grpc.secure_channel(
        f"{cfg.endpoint}:443",
        grpc.composite_channel_credentials(
            grpc.ssl_channel_credentials(),
            grpc.metadata_call_credentials(TokenAuth(token_source)),
        ),
        # Without an explicit authority the port travels in the Host header and
        # every RPC comes back UNIMPLEMENTED with no message — which reads as a
        # missing service rather than a routing miss.
        options=(("grpc.default_authority", cfg.endpoint),),
    )

    with channel:
        api = Clients(channel)
        for name, run in STEPS:
            print(f"\n== {name}")
            run(api, cfg)

        print("\n== verify")
        verify(api, cfg)


if __name__ == "__main__":
    main()
