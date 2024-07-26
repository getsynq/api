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
					Dialect: entitiescustomfeaturesv1.SqlDialect_SQL_DIALECT_CLICKHOUSE,
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
					CodeType: entitiescustomfeaturesv1.CodeType_CODE_TYPE_PYTHON,
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
					Columns: []*entitiescustomfeaturesv1.SchemaColumn{
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

}
