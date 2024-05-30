package main

import (
	statusservicev1 "buf.build/gen/go/getsynq/api/grpc/go/synq/status/v1/statusv1grpc"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	statusv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/status/v1"
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

	statusServiceClient := statusservicev1.NewEntityIssuesServiceClient(conn)

	issuesStatusResponse, err := statusServiceClient.BatchGetIssuesStatus(ctx, &statusv1.BatchGetIssuesStatusRequest{
		Requests: []*statusv1.GetIssuesStatusRequest{
			{
				Id: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_Dataproduct{
						Dataproduct: &entitiesv1.DataproductIdentifier{
							Id: "e6232018-39f9-4b4a-a5bb-824dfb9a220d",
						},
					},
				},
				FetchUpstreamStatus: true,
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
				FetchUpstreamStatus: true,
			},
			{
				Id: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_SynqPath{
						SynqPath: &entitiesv1.SynqPathIdentifier{
							Path: "ch-prod::default::runs",
						},
					},
				},
				FetchUpstreamStatus: true,
			},
			{
				Id: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_SynqPath{
						SynqPath: &entitiesv1.SynqPathIdentifier{
							Path: "ch-prod::anomalies::predictions_v2",
						},
					},
				},
				FetchUpstreamStatus: true,
			},
			{
				Id: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_SynqPath{
						SynqPath: &entitiesv1.SynqPathIdentifier{
							Path: "ch-prod::analytics::s_kernel_anomalies_predictions",
						},
					},
				},
				FetchUpstreamStatus: true,
			},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("issuesStatusResponse:")

	for _, resp := range issuesStatusResponse.Responses {
		fmt.Printf("Entity %s\n", resp.Id.String())
		fmt.Printf("entity status: %s upstream status: %s\n", resp.EntityIssuesStatus.String(), resp.UpstreamIssuesStatus.String())
		if resp.EntityIssuesSummary != nil && resp.EntityIssuesSummary.String() != "" {
			fmt.Printf("summary: %s\n", resp.EntityIssuesSummary.String())
		}
		if resp.UpstreamIssuesSummary != nil && resp.UpstreamIssuesSummary.String() != "" {
			fmt.Printf("upstream summary: %s\n", resp.UpstreamIssuesSummary.String())
		}
	}
}
