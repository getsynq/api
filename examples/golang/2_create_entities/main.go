package main

import (
	entitiescustomv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/custom/v1/customv1grpc"
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

	entitiesapi := entitiescustomv1grpc.NewEntitiesServiceClient(conn)
	relationshipsapi := entitiescustomv1grpc.NewRelationshipsServiceClient(conn)

	// Begin: Create Entities

	// Create datadog monitor for kernel-accounts entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiescustomv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "datadog::monitor::kernel-accounts-api_error_too_high",
					},
				},
			},
			TypeId:      11,
			Name:        "kernel-accounts API error rate is too high",
			Description: "Monitoring API errors for service kernel-accounts",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	// Create datadog monitor for kernel-auth entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiescustomv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "datadog::monitor::kernel-auth-api_error_too_high",
					},
				},
			},
			TypeId:      11,
			Name:        "kernel-auth API error rate is too high",
			Description: "Monitoring API errors for service kernel-auth",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	// Create kernel-accounts service entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiescustomv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-accounts",
					},
				},
			},
			TypeId:      10,
			Name:        "Kernel accounts service",
			Description: "Service responsible for storing users",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	// Create kernel-auth service entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiescustomv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-auth",
					},
				},
			},
			TypeId:      10,
			Name:        "Kernel auth service",
			Description: "Service responsible for user authentication",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	//
	// Service kernel-accounts is downstream of its datadog's monitor
	// Service kernel-auth is downstream of its datadog's monitor
	// Both services are upstream of clickhouse table `users`
	_, err = relationshipsapi.UpsertRelationships(ctx, &entitiescustomv1.UpsertRelationshipsRequest{
		Relationships: []*entitiescustomv1.Relationship{
			{
				Upstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Custom{
						Custom: &entitiesv1.CustomIdentifier{
							Id: "datadog::monitor::kernel-accounts-api_error_too_high",
						},
					},
				},
				Downstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Custom{
						Custom: &entitiesv1.CustomIdentifier{
							Id: "service::kernel-accounts",
						},
					},
				},
			},
			{
				Upstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Custom{
						Custom: &entitiesv1.CustomIdentifier{
							Id: "datadog::monitor::kernel-auth-api_error_too_high",
						},
					},
				},
				Downstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Custom{
						Custom: &entitiesv1.CustomIdentifier{
							Id: "service::kernel-auth",
						},
					},
				},
			},
			{
				Upstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Custom{
						Custom: &entitiesv1.CustomIdentifier{
							Id: "service::kernel-auth",
						},
					},
				},
				Downstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_ClickhouseTable{
						ClickhouseTable: &entitiesv1.ClickhouseTableIdentifier{
							Host:   "prod",
							Schema: "system",
							Table:  "users",
						},
					},
				},
			},
			{
				Upstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Custom{
						Custom: &entitiesv1.CustomIdentifier{
							Id: "service::kernel-accounts",
						},
					},
				},
				Downstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_ClickhouseTable{
						ClickhouseTable: &entitiesv1.ClickhouseTableIdentifier{
							Host:   "prod",
							Schema: "system",
							Table:  "users",
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
