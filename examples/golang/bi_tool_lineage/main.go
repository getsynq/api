// Modelling a BI tool Coalesce Quality has no integration for, end to end.
//
// The estate this builds is four layers deep, and each hop uses a DIFFERENT
// lineage mechanism, on purpose — picking the right one per hop is the whole
// skill of modelling a new tool:
//
//	warehouse tables                 (already in Coalesce Quality)
//	  |  SQL, resolved by warehouse address
//	  v
//	BI models                        custom entities, SqlDefinition
//	  |  SQL, resolved by a BINDING — a model has no warehouse address
//	  v
//	BI questions                     custom entities, SqlDefinition + references
//	  |  no SQL at all: declared column lineage
//	  v
//	BI dashboard                     custom entity, ColumnLineage feature
//	  |  no columns: a plain relationship
//	  v
//	BI subscription                  custom entity
//
//	and off to one side, a BI alert   custom entity, CheckCategory feature
//	                                  attached with a check relationship
//
// Read metabase.go first — it is the tool's own metadata, and the mapping in
// sync.go is easier to follow once you know what is being mapped.
//
// Everything written here is idempotent: re-running the program produces the
// same estate, which is what lets it be a cron job rather than a migration.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	entitiescustomv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/custom/v1/customv1grpc"
	lineagev1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/lineage/v1/lineagev1grpc"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// Custom type ids are workspace-wide and yours to allocate: 1..1000, one number
// per kind of thing your tool has. Pick a free band and keep it — an entity's
// type is its id, so reusing a number silently reclassifies everything that had
// it. checkTypeIdsAreFree below refuses to start if one of these is already in
// use under another name.
const (
	typeModel        = 40
	typeQuestion     = 41
	typeDashboard    = 42
	typeSubscription = 43
	typeAlert        = 44
)

// groupID is the entity group every entity below is a member of. The group is
// what makes a re-sync self-cleaning: content deleted in the BI tool simply
// stops being sent, and the server deletes it. Nothing has to be remembered on
// this side between runs.
const groupID = "metabase"

type config struct {
	endpoint string

	// The warehouse the BI tool queries. These name the DEFAULT execution
	// context of every query the tool runs, which is what lets an unqualified
	// table name in the SQL resolve to a real warehouse object.
	project string
	dataset string

	// Optional: the repository a serialized export of the BI content is
	// committed to, so Coalesce Quality can show its history. Leave unset to
	// skip that step.
	gitRepo   string
	gitBranch string
}

func main() {
	ctx := context.Background()

	cfg := config{
		// developer.synq.io (EU) is the default. The other deployments are
		// api.us.synq.io (US) and api.au.synq.io (AU); set QUALITY_API_ENDPOINT
		// to the one your workspace lives in.
		endpoint:  env("QUALITY_API_ENDPOINT", "developer.synq.io"),
		project:   env("BIGQUERY_PROJECT", "my-gcp-project"),
		dataset:   env("BIGQUERY_DATASET", "analytics"),
		gitRepo:   os.Getenv("BI_EXPORT_GIT_REPO"),
		gitBranch: env("BI_EXPORT_GIT_BRANCH", "main"),
	}

	clientID := env("QUALITY_CLIENT_ID", os.Getenv("SYNQ_CLIENT_ID"))
	clientSecret := env("QUALITY_CLIENT_SECRET", os.Getenv("SYNQ_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		fmt.Println("set QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET (see README.md)")
		os.Exit(1)
	}

	oauthConfig := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("https://%s/oauth2/token", cfg.endpoint),
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:443", cfg.endpoint),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: oauthConfig.TokenSource(ctx)}),
		grpc.WithAuthority(cfg.endpoint),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	api := &clients{
		types:              entitiescustomv1grpc.NewTypesServiceClient(conn),
		entities:           entitiescustomv1grpc.NewEntitiesServiceClient(conn),
		features:           entitiescustomv1grpc.NewFeaturesServiceClient(conn),
		relationships:      entitiescustomv1grpc.NewRelationshipsServiceClient(conn),
		checkRelationships: entitiescustomv1grpc.NewChecksRelationshipsServiceClient(conn),
		groups:             entitiescustomv1grpc.NewGroupsServiceClient(conn),
		executions:         entitiescustomv1grpc.NewEntityExecutionsServiceClient(conn),
		lineage:            lineagev1grpc.NewLineageServiceClient(conn),
	}

	// The order below is not cosmetic. Three of these steps depend on an earlier
	// one having landed:
	//
	//  - An entity carries a type id, so the types go first.
	//  - A binding to a custom entity that does not exist is REFUSED, so every
	//    entity exists before any SQL definition binds to one.
	//  - A `SELECT *` over a bound model expands only if that model has declared
	//    its columns, so schemas are written before the SQL that reads them.
	//
	// The last one is invisible when it is wrong: the table-level edge still
	// appears, only the column-level edges are missing.
	steps := []struct {
		name string
		run  func(context.Context, *clients, config) error
	}{
		{"check type ids are free", checkTypeIdsAreFree},
		{"declare entity types", syncTypes},
		{"declare entities", syncEntities},
		{"declare schemas", syncSchemas},
		{"declare model SQL (resolved by warehouse address)", syncModelSql},
		{"declare question SQL (resolved by binding)", syncQuestionSql},
		{"declare dashboard column lineage", syncDashboardColumnLineage},
		{"attach the tool's own definitions as code", syncCode},
		{"link the serialized export to git", syncGitFileReferences},
		{"join the subscription to its dashboard", syncRelationships},
		{"declare the alert as a check", syncAlerts},
		{"reconcile the entity group", syncGroup},
		{"report the last refresh of each dashboard", syncExecutions},
	}
	for _, step := range steps {
		fmt.Printf("\n== %s\n", step.name)
		if err := step.run(ctx, api, cfg); err != nil {
			panic(fmt.Errorf("%s: %w", step.name, err))
		}
	}

	// Lineage is computed asynchronously from what was just written: the SQL is
	// parsed, the bindings are resolved, and the graph is rebuilt. Reading it
	// back immediately usually finds it half-built, so this polls.
	fmt.Printf("\n== verify\n")
	verify(ctx, api, cfg, 2*time.Minute)
}

type clients struct {
	types              entitiescustomv1grpc.TypesServiceClient
	entities           entitiescustomv1grpc.EntitiesServiceClient
	features           entitiescustomv1grpc.FeaturesServiceClient
	relationships      entitiescustomv1grpc.RelationshipsServiceClient
	checkRelationships entitiescustomv1grpc.ChecksRelationshipsServiceClient
	groups             entitiescustomv1grpc.GroupsServiceClient
	executions         entitiescustomv1grpc.EntityExecutionsServiceClient
	lineage            lineagev1grpc.LineageServiceClient
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
