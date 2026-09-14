package main

import (
	entitiescustomv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/custom/v1/customv1grpc"
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
)

func main() {
	ctx := context.Background()

	host := "developer.synq.io"
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	clientID := "foo"
	clientSecret := "bar"
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
	// can only reach objects that exist in a database. When the upstream is another custom entity —
	// a dashboard reading from a semantic-layer model, say — there is no SQL to give, and this is
	// how the result is stated directly.
	//
	// The declaration is complete: every write replaces the previous one, so an edge left out is
	// withdrawn and re-sending the same set changes nothing. That makes it safe to regenerate the
	// whole feature from your own metadata on every run. Only one column-lineage feature is allowed
	// per entity, so keep the feature_id stable and re-write it to edit.
	//
	// Declaring a column edge also puts the entity downstream of each upstream named here, so the
	// table-level edge does not have to be declared separately.
	_, err = featuresapi.UpsertEntityFeature(ctx, &entitiescustomv1.UpsertEntityFeatureRequest{
		Feature: &entitiescustomv1.Feature{
			EntityId: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "dashboard::accounts-overview",
					},
				},
			},
			FeatureId: "column-lineage",
			Feature: &entitiescustomv1.Feature_ColumnLineage{
				ColumnLineage: &entitiescustomfeaturesv1.ColumnLineage{
					StateAt: timestamppb.Now(),
					Edges: []*entitiescustomfeaturesv1.ColumnEdge{
						// From another custom entity. The two column names are stated
						// independently, so they do not have to match: here the dashboard
						// prefixes its own columns with the model they came from.
						{
							Upstream: &entitiesv1.Identifier{
								Id: &entitiesv1.Identifier_Custom{
									Custom: &entitiesv1.CustomIdentifier{
										Id: "service::kernel-accounts",
									},
								},
							},
							UpstreamColumn: "id",
							Column:         "kernel_accounts.id",
						},
						{
							Upstream: &entitiesv1.Identifier{
								Id: &entitiesv1.Identifier_Custom{
									Custom: &entitiesv1.CustomIdentifier{
										Id: "service::kernel-accounts",
									},
								},
							},
							UpstreamColumn: "roles",
							Column:         "kernel_accounts.roles",
						},
						// An upstream on a connected platform works the same way. It does not
						// have to exist yet — the edge appears once it is next ingested.
						{
							Upstream: &entitiesv1.Identifier{
								Id: &entitiesv1.Identifier_ClickhouseTable{
									ClickhouseTable: &entitiesv1.ClickhouseTableIdentifier{
										Host:   "clickhouse.example.com",
										Schema: "default",
										Table:  "runs",
									},
								},
							},
							UpstreamColumn: "run_id",
							Column:         "runs.run_id",
						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

}
