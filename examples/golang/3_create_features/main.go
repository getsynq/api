package main

import (
	"buf.build/gen/go/getsynq/api/grpc/go/synq/entities/coordinates/v1/coordinatesv1grpc"
	entitiescustomv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/custom/v1/customv1grpc"
	coordinatesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/coordinates/v1"
	entitiescustomfeaturesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/features/v1"
	entitiescustomv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/timestamppb"
	"os"
)

func main() {
	ctx := context.Background()

	// EU is developer.synq.io (the default). Set QUALITY_API_ENDPOINT to
	// api.us.synq.io for US or api.au.synq.io for AU.
	host := getenv("QUALITY_API_ENDPOINT", "developer.synq.io")
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	clientID := os.Getenv("QUALITY_CLIENT_ID")
	clientSecret := os.Getenv("QUALITY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		panic("QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET must be set")
	}
	tokenURL := fmt.Sprintf("https://%s/oauth2/token", host)

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}
	oauthTokenSource := oauth.TokenSource{TokenSource: config.TokenSource(ctx)}
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(oauthTokenSource),
		grpc.WithAuthority(host),
	}

	conn, err := grpc.DialContext(ctx, apiUrl, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	featuresapi := entitiescustomv1grpc.NewFeaturesServiceClient(conn)

	// Define relationships of custom entity via SQL code
	_, err = featuresapi.UpsertEntityFeature(ctx, &entitiescustomv1.UpsertEntityFeatureRequest{
		Feature: &entitiescustomv1.Feature{
			EntityId: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-accounts",
					},
				},
			},
			FeatureId: "sql",
			Feature: &entitiescustomv1.Feature_SqlDefinition{
				SqlDefinition: &entitiescustomfeaturesv1.SqlDefinition{
					StateAt: timestamppb.Now(),
					Dialect: entitiesv1.SqlDialect_SQL_DIALECT_CLICKHOUSE,
					Sql:     "SELECT * FROM default.runs",
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Create Code&Changes code
	_, err = featuresapi.UpsertEntityFeature(ctx, &entitiescustomv1.UpsertEntityFeatureRequest{
		Feature: &entitiescustomv1.Feature{
			EntityId: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-accounts",
					},
				},
			},
			FeatureId: "main.py",
			Feature: &entitiescustomv1.Feature_Code{
				Code: &entitiescustomfeaturesv1.Code{
					Name:     "launcher",
					CodeType: entitiesv1.CodeType_CODE_TYPE_PYTHON,
					Content:  "from airflow.operators.python_operator import PythonOperator\n\nprint('Hello, world!')",
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Link custom entity to history of a git versioned file
	_, err = featuresapi.UpsertEntityFeature(ctx, &entitiescustomv1.UpsertEntityFeatureRequest{
		Feature: &entitiescustomv1.Feature{
			EntityId: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-accounts",
					},
				},
			},
			FeatureId: "main.libsonnet",
			Feature: &entitiescustomv1.Feature_GitFileReference{
				GitFileReference: &entitiescustomfeaturesv1.GitFileReference{
					RepositoryUrl: "git@github.com:getsynq/cloud.git",
					BranchName:    "main",
					FilePath:      "kernel-accounts/main.libsonnet",
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Define what columns this custom entity has
	_, err = featuresapi.UpsertEntityFeature(ctx, &entitiescustomv1.UpsertEntityFeatureRequest{
		Feature: &entitiescustomv1.Feature{
			EntityId: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-accounts",
					},
				},
			},
			FeatureId: "schema",
			Feature: &entitiescustomv1.Feature_Schema{
				Schema: &entitiescustomfeaturesv1.Schema{
					Columns: []*entitiesv1.SchemaColumn{
						{
							Name:        "id",
							NativeType:  "TEXT",
							Description: "Unique identifier",
						},
						{
							Name:        "roles",
							NativeType:  "TEXT",
							Description: "Roles associated to the user",
						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	// Declare column-level lineage that SQL cannot express.
	//
	// A SQL definition is parsed and its column lineage derived from the tables the SQL names, so it
	// can only reach objects that exist in a database. When the upstream is another custom entity
	// (for example, a dashboard that reads from a semantic-layer model), there is no SQL to give.
	// Use this feature to state the result directly.
	//
	// The declaration is complete: every write replaces the previous one, so an edge left out is
	// withdrawn and re-sending the same set changes nothing. That makes it safe to regenerate the
	// whole feature from your own metadata on every run. Only one column-lineage feature is allowed
	// per entity, so keep the feature_id stable and re-write it to edit.
	//
	// Declaring a column edge also puts the entity downstream of each upstream named here, so the
	// table-level edge does not have to be declared separately.
	edges := []*entitiescustomfeaturesv1.ColumnEdge{
		// From another custom entity. The two column names are stated
		// independently, so they do not have to match: here the dashboard
		// prefixes its own columns with the model they came from.
		{
			Upstream:       customID("service::kernel-accounts"),
			UpstreamColumn: "id",
			Column:         "kernel_accounts.id",
		},
		{
			Upstream:       customID("service::kernel-accounts"),
			UpstreamColumn: "roles",
			Column:         "kernel_accounts.roles",
		},
	}

	// An upstream on a connected platform works the same way. Name it by its entity_id,
	// looked up by the SQL name your tool knows. Set WAREHOUSE_TABLE to a table of your
	// warehouse that has a `run_id` column.
	coordinatesapi := coordinatesv1grpc.NewDatabaseCoordinatesServiceClient(conn)
	tableName := getenv("WAREHOUSE_TABLE", "default.runs")
	table, found, err := tableEntityID(ctx, coordinatesapi, tableName)
	if err != nil {
		panic(err)
	}
	if found {
		edges = append(edges, &entitiescustomfeaturesv1.ColumnEdge{
			Upstream:       table,
			UpstreamColumn: "run_id",
			Column:         "runs.run_id",
		})
	} else {
		fmt.Printf("%s is not in the catalog, so the dashboard gets no column from it. Set WAREHOUSE_TABLE to a table of your warehouse.\n", tableName)
	}

	_, err = featuresapi.UpsertEntityFeature(ctx, &entitiescustomv1.UpsertEntityFeatureRequest{
		Feature: &entitiescustomv1.Feature{
			EntityId:  customID("dashboard::accounts-overview"),
			FeatureId: "column-lineage",
			Feature: &entitiescustomv1.Feature_ColumnLineage{
				ColumnLineage: &entitiescustomfeaturesv1.ColumnLineage{
					StateAt: timestamppb.Now(),
					Edges:   edges,
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

func customID(id string) *entitiesv1.Identifier {
	return &entitiesv1.Identifier{Id: &entitiesv1.Identifier_Custom{Custom: &entitiesv1.CustomIdentifier{Id: id}}}
}

// tableEntityID looks up the entity_id of a warehouse table by its SQL name, as the
// warehouse accepts it (`schema.table` or `database.schema.table`), and returns it as an
// identifier. found is false if the catalog has no such table.
func tableEntityID(
	ctx context.Context,
	api coordinatesv1grpc.DatabaseCoordinatesServiceClient,
	sqlName string,
) (*entitiesv1.Identifier, bool, error) {
	resp, err := api.BatchIdsByCoordinates(ctx, &coordinatesv1.BatchIdsByCoordinatesRequest{SqlFqn: []string{sqlName}})
	if err != nil {
		return nil, false, err
	}
	for _, match := range resp.GetMatchedCoordinates() {
		candidates := match.GetCandidates()
		if len(candidates) > 1 {
			return nil, false, fmt.Errorf("%s matches %d tables: give the full name, with the database", sqlName, len(candidates))
		}
		for _, candidate := range candidates {
			for _, entityID := range candidate.GetSynqPaths() {
				return &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_SynqPath{SynqPath: &entitiesv1.SynqPathIdentifier{Path: entityID}},
				}, true, nil
			}
		}
	}
	return nil, false, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
