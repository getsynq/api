// Lineage for a replication tool: a table copied from one database into a
// table in a different warehouse, with column lineage across the hop.
//
//	Postgres table                   (already in Coalesce Quality)
//	  |  SQL: INSERT INTO <target> (...) SELECT ... FROM <source>
//	  v
//	replication stream               custom entity, SqlDefinition + references
//	  |  the same SQL's write target
//	  v
//	Snowflake table                  (already in Coalesce Quality)
//
// Two tables cannot be joined by a relationship directly: a relationship needs a
// custom entity on at least one side. That is not a limitation to work around,
// it is the model — the rows do not move by themselves, something copies them,
// and that something is what fails, lags or drops a column. So it gets an
// entity, one per replicated table (a "stream"), and the two tables hang off it.
//
// One entity per stream, never one for the whole connection: with a single
// entity in the middle, every source table would read as upstream of every
// target table the connection writes.
//
// The hop is described as SQL rather than as a list of edges, because SQL says
// which column lands in which, including renames, and the parser classifies each
// one (passthrough, renamed, transformed). The two tables live in different
// warehouses, so neither name in the SQL can be looked up by warehouse address
// from the other side. Both are BOUND instead: a reference pins the name as the
// SQL writes it to the entity it stands for, whatever platform that is on.
//
// Everything written here is idempotent: re-running the program produces the
// same estate, and a stream that disappears from the list is deleted.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	entitiescustomv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/custom/v1/customv1grpc"
	lineagev1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/lineage/v1/lineagev1grpc"
	customfeaturesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/features/v1"
	customv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/v1"
	lineagev1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/lineage/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Custom type ids are workspace-wide and yours to allocate: 1..1000. Pick a free
// one and keep it — an entity's type is its id, so reusing a number silently
// reclassifies everything that had it.
const typeStream = 60

// groupID makes a re-sync self-cleaning: a stream removed from the replication
// tool simply stops being sent, and the server deletes its entity.
const groupID = "replication"

// stream is one replicated table, as the replication tool describes it.
type stream struct {
	name string

	source *entitiesv1.Identifier
	target *entitiesv1.Identifier

	// Source column -> target column. Most tools copy names through unchanged;
	// list the columns anyway, because the list is what says which of the
	// target's columns came from the source and which the tool added itself.
	columns [][2]string
}

// A real integration reads these from the replication tool's own catalog.
var streams = []stream{
	{
		name:   "app_db.public.customers",
		source: postgresTable("app-db.internal.example.com", "app_db", "public", "customers"),
		target: snowflakeTable("xy12345.eu-west-1", "RAW", "APP_DB", "CUSTOMERS"),
		columns: [][2]string{
			{"id", "id"},
			{"email", "email"},
			{"created_at", "created_at"},
		},
	},
	{
		name:   "app_db.public.orders",
		source: postgresTable("app-db.internal.example.com", "app_db", "public", "orders"),
		target: snowflakeTable("xy12345.eu-west-1", "RAW", "APP_DB", "ORDERS"),
		columns: [][2]string{
			{"id", "id"},
			{"customer_id", "customer_id"},
			{"total", "total_amount"},
			{"created_at", "created_at"},
		},
	},
}

func main() {
	ctx := context.Background()

	// developer.synq.io (EU) is the default. The other deployments are
	// api.us.synq.io (US) and api.au.synq.io (AU).
	endpoint := env("QUALITY_API_ENDPOINT", "developer.synq.io")
	clientID := os.Getenv("QUALITY_CLIENT_ID")
	clientSecret := os.Getenv("QUALITY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		fmt.Println("set QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET (see README.md)")
		os.Exit(1)
	}

	oauthConfig := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("https://%s/oauth2/token", endpoint),
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:443", endpoint),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: oauthConfig.TokenSource(ctx)}),
		grpc.WithAuthority(endpoint),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	types := entitiescustomv1grpc.NewTypesServiceClient(conn)
	entities := entitiescustomv1grpc.NewEntitiesServiceClient(conn)
	features := entitiescustomv1grpc.NewFeaturesServiceClient(conn)
	groups := entitiescustomv1grpc.NewGroupsServiceClient(conn)
	lineage := lineagev1grpc.NewLineageServiceClient(conn)

	fmt.Println("== declare the stream type")
	_, err = types.UpsertType(ctx, &customv1.UpsertTypeRequest{Type: &entitiesv1.Type{
		TypeId:  typeStream,
		Name:    "Replication stream",
		SvgIcon: iconArrows,
	}})
	must(err)

	fmt.Println("== declare one entity per stream")
	for _, s := range streams {
		_, err := entities.UpsertEntity(ctx, &customv1.UpsertEntityRequest{Entity: &entitiesv1.Entity{
			Id:          streamID(s),
			TypeId:      typeStream,
			Name:        s.name,
			Description: "Replicated by the replication tool.",
		}})
		must(err)
		fmt.Printf("   %s\n", s.name)
	}

	fmt.Println("== describe each hop as SQL")
	for _, s := range streams {
		_, err := features.UpsertEntityFeature(ctx, &customv1.UpsertEntityFeatureRequest{
			Feature: &customv1.Feature{
				EntityId:  streamID(s),
				FeatureId: "sql",
				Feature:   &customv1.Feature_SqlDefinition{SqlDefinition: streamSql(s)},
			},
		})
		must(err)
		fmt.Printf("   %s\n", streamSql(s).GetSql())
	}

	fmt.Println("== reconcile the group")
	ids := make([]*entitiesv1.Identifier, 0, len(streams))
	for _, s := range streams {
		ids = append(ids, streamID(s))
	}
	resp, err := groups.UpsertEntitiesGroup(ctx, &customv1.UpsertEntitiesGroupRequest{
		Group: &customv1.Group{GroupId: groupID, EntityIds: ids},
	})
	must(err)
	for _, deleted := range resp.GetDeletedIds() {
		fmt.Printf("   deleted (no longer replicated): %s\n", deleted.GetCustom().GetId())
	}

	fmt.Println("== verify")
	verify(ctx, lineage, streams[1], 2*time.Minute)
}

// streamSql is the hop, written as the SQL a replication would run if source and
// target were in one database.
//
// The dialect is the SOURCE side's, and nothing in the SQL is ever executed: it
// is parsed for lineage only. The two names it uses are placeholders — they only
// have to match the references' object_name — which is what lets the target sit
// in a different warehouse from the source.
func streamSql(s stream) *customfeaturesv1.SqlDefinition {
	targetColumns := make([]string, 0, len(s.columns))
	selectList := make([]string, 0, len(s.columns))
	for _, c := range s.columns {
		selectList = append(selectList, c[0])
		targetColumns = append(targetColumns, c[1])
	}

	return &customfeaturesv1.SqlDefinition{
		StateAt: timestamppb.Now(),
		Dialect: entitiesv1.SqlDialect_SQL_DIALECT_POSTGRESQL,
		// An explicit column list on the INSERT is what pairs each selected
		// column with its target column. Without one the pairing is positional
		// and needs the target's shape to resolve.
		Sql: fmt.Sprintf("INSERT INTO target_table (%s) SELECT %s FROM source_table",
			strings.Join(targetColumns, ", "), strings.Join(selectList, ", ")),
		// Bound, not looked up: the source and the target are on different
		// platforms, and a reference resolves to its entity on whichever one it
		// is. A binding to a table Coalesce Quality has not ingested yet is kept,
		// and the edge appears once the table does.
		References: []*customfeaturesv1.SqlTableReference{
			{ObjectName: "source_table", Entity: s.source},
			{ObjectName: "target_table", Entity: s.target},
		},
	}
}

// verify reads the column lineage back from the target side.
//
// The way this goes wrong is silent: a reference to a table that is not in
// Coalesce Quality under that identifier leaves a valid write and an empty
// graph. Lineage is computed asynchronously from what was written, so this polls.
func verify(ctx context.Context, api lineagev1grpc.LineageServiceClient, s stream, budget time.Duration) {
	columns := make([]string, 0, len(s.columns))
	for _, c := range s.columns {
		columns = append(columns, c[1])
	}

	deadline := time.Now().Add(budget)
	for {
		resp, err := api.GetLineage(ctx, &lineagev1.GetLineageRequest{
			LineageDirection: lineagev1.LineageDirection_LINEAGE_DIRECTION_UPSTREAM,
			StartPoint: &lineagev1.GetLineageStartPoint{
				From: &lineagev1.GetLineageStartPoint_EntityColumns{
					EntityColumns: &lineagev1.EntityColumnsStartPoint{Id: s.target, ColumnNames: columns},
				},
			},
		})
		if err != nil {
			fmt.Printf("   (lineage read failed: %v)\n", err)
			return
		}
		lin := resp.GetLineage()
		// Two hops per column: source -> stream -> target.
		if len(lin.GetColumnDependencies()) >= 2*len(columns) {
			for _, d := range lin.GetColumnDependencies() {
				fmt.Printf("   %s.%s -> %s.%s\n",
					lin.GetNodes()[d.GetSourceNodeIdx()].GetIds()[0].GetEntityId(), d.GetSourceNodeColumnId(),
					lin.GetNodes()[d.GetTargetNodeIdx()].GetIds()[0].GetEntityId(), d.GetTargetNodeColumnId(),
				)
			}
			return
		}
		if time.Now().After(deadline) {
			fmt.Printf("   (gave up after %s: check both tables exist in Coalesce Quality under these identifiers)\n", budget)
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func streamID(s stream) *entitiesv1.Identifier {
	return &entitiesv1.Identifier{Id: &entitiesv1.Identifier_Custom{
		Custom: &entitiesv1.CustomIdentifier{Id: "replication::stream::" + s.name},
	}}
}

func postgresTable(host, database, schema, table string) *entitiesv1.Identifier {
	return &entitiesv1.Identifier{Id: &entitiesv1.Identifier_PostgresTable{
		PostgresTable: &entitiesv1.PostgresTableIdentifier{Host: host, Database: database, Schema: schema, Table: table},
	}}
}

func snowflakeTable(account, database, schema, table string) *entitiesv1.Identifier {
	return &entitiesv1.Identifier{Id: &entitiesv1.Identifier_SnowflakeTable{
		SnowflakeTable: &entitiesv1.SnowflakeTableIdentifier{Account: account, Database: database, Schema: schema, Table: table},
	}}
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var iconArrows = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M32 160h352l-64-64 32-32 128 128-128 128-32-32 64-64H32z"/><path d="M480 352H128l64 64-32 32L32 320l128-128 32 32-64 64h352z"/></svg>`)
