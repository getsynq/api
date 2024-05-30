package main

import (
	entitiesservicev1 "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/v1/entitiesv1grpc"
	lineageservicev1 "buf.build/gen/go/getsynq/api/grpc/go/synq/lineage/v1/lineagev1grpc"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	lineagev1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/lineage/v1"
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

	host := "developer.synq.dev"
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

	entitiesapi := entitiesservicev1.NewEntitiesServiceClient(conn)
	relationshipsapi := lineageservicev1.NewRelationshipsServiceClient(conn)

	// Begin: Create Entities

	// Create datadog monitor for kernel-accounts entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiesv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "datadog::monitor::kernel-accounts-api_error_too_high",
					},
				},
			},
			Name:        "kernel-accounts API error rate is too high",
			Description: "Monitoring API errors for service kernel-accounts",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	// Create datadog monitor for kernel-auth entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiesv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "datadog::monitor::kernel-auth-api_error_too_high",
					},
				},
			},
			Name:        "kernel-auth API error rate is too high",
			Description: "Monitoring API errors for service kernel-auth",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	// Create kernel-accounts service entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiesv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-accounts",
					},
				},
			},
			Name:        "Kernel accounts service",
			Description: "Service responsible for storing users",
			CreatedAt:   timestamppb.Now(),
		},
	})

	if err != nil {
		panic(err)
	}

	// Create kernel-auth service entity
	_, err = entitiesapi.UpsertEntity(ctx, &entitiesv1.UpsertEntityRequest{
		Entity: &entitiesv1.Entity{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Custom{
					Custom: &entitiesv1.CustomIdentifier{
						Id: "service::kernel-auth",
					},
				},
			},
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
	_, err = relationshipsapi.UpsertRelationships(ctx, &lineagev1.UpsertRelationshipsRequest{
		Relationships: []*lineagev1.Relationship{
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
