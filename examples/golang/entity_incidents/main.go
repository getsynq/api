package main

import (
	entitiesstatusv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/status/v1/statusv1grpc"
	entitiesstatusv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/status/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
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

	statusServiceClient := entitiesstatusv1grpc.NewEntityIncidentsServiceClient(conn)

	requests := []*entitiesstatusv1.GetIncidentsRequest{
		{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_Dataproduct{
					Dataproduct: &entitiesv1.DataproductIdentifier{
						Id: "e6232018-39f9-4b4a-a5bb-824dfb9a220d",
					},
				},
			},
			FetchUpstreamIncidents: true,
		},
		{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_DbtCoreNode{
					DbtCoreNode: &entitiesv1.DbtCoreNodeIdentifier{
						IntegrationId: "d577b364-a867-11ed-b4b2-fe8020e7ba25",
						NodeId:        "model.ops.stg_runs",
					},
				},
			},
			FetchUpstreamIncidents: true,
		},
		{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_SynqPath{
					SynqPath: &entitiesv1.SynqPathIdentifier{
						Path: "ch-prod::default::runs",
					},
				},
			},
			FetchUpstreamIncidents: true,
		},
		{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_SynqPath{
					SynqPath: &entitiesv1.SynqPathIdentifier{
						Path: "ch-prod::anomalies::predictions_v2",
					},
				},
			},
			FetchUpstreamIncidents: true,
		},
		{
			Id: &entitiesv1.Identifier{
				Id: &entitiesv1.Identifier_SynqPath{
					SynqPath: &entitiesv1.SynqPathIdentifier{
						Path: "ch-prod::analytics::s_kernel_anomalies_predictions",
					},
				},
			},
			FetchUpstreamIncidents: true,
		},
	}

	issuesStatusResponse, err := statusServiceClient.BatchGetIncidents(ctx, &entitiesstatusv1.BatchGetIncidentsRequest{
		Requests: requests,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Found open incidents:")

	for i, resp := range issuesStatusResponse.Responses {
		fmt.Printf("Entity %s\n", resp.Id.String())
		if len(resp.EntityOpenIncidents) == 0 {
			fmt.Println("No open incidents")
		} else {
			for _, incident := range resp.EntityOpenIncidents {
				fmt.Printf("Incident %s: %s %s \n", incident.Id, incident.Name, incident.Url)
			}
		}
		if requests[i].FetchUpstreamIncidents {
			if len(resp.UpstreamOpenIncidents) == 0 {
				fmt.Println("No open upstream incidents")
			} else {
				for _, incident := range resp.UpstreamOpenIncidents {
					fmt.Printf("Upstream Incident %s: %s %s \n", incident.Id, incident.Name, incident.Url)
				}
			}
		}
	}
}
