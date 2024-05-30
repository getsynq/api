package main

import (
	"context"
	"crypto/tls"
	"fmt"

	statusservicev1 "buf.build/gen/go/getsynq/api/grpc/go/synq/status/v1/statusv1grpc"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	statusv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/status/v1"
	"github.com/getsynq/api/examples/go/token_auth/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx := context.Background()

	host := "developer.synq.dev"
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	longLivedToken := "long-lived-token"

	oauthTokenSource, err := auth.LongLivedTokenSource(longLivedToken, host)
	if err != nil {
		panic(err)
	}
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

	issuesStatus, err := statusServiceClient.BatchGetIssuesStatus(ctx, &statusv1.BatchGetIssuesStatusRequest{
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

	fmt.Println("API worked:")

	fmt.Println("issuesStatus:", issuesStatus)

}
