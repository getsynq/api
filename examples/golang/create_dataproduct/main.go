package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/google/uuid"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	dataproductsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/dataproducts/v1/dataproductsv1grpc"
	dataproductsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/dataproducts/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	queriesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/queries/v1"
)

func main() {
    ctx := context.Background()

    host := "developer.synq.io"
    port := "443"
    apiUrl := fmt.Sprintf("%s:%s", host, port)

    clientID := os.Getenv("SYNQ_CLIENT_ID")
	clientSecret := os.Getenv("SYNQ_CLIENT_SECRET")
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

    dataproductsapi := dataproductsv1grpc.NewDataproductsServiceClient(conn)

	resp, err := dataproductsapi.Upsert(ctx, &dataproductsv1.UpsertRequest{
		Title: "Dim Payment Types",
		Description: "A data product that contains the payment types used in the system.",
	})
	if err != nil {
		panic(err)
	}
	dataproductIdentifier := resp.Identifier.Id
	fmt.Printf("✅ Data product created: Dim Payment Types (%s)\n", dataproductIdentifier)

	_, err = dataproductsapi.AddDefinitionPart(ctx, &dataproductsv1.AddDefinitionPartRequest{
		ProductIdentifier: &entitiesv1.DataproductIdentifier{
			Id: dataproductIdentifier,
		},
		Part: &dataproductsv1.DataproductDefinition_Part{
			Id: uuid.NewString(),
			Part: &dataproductsv1.DataproductDefinition_Part_Query{
				Query: &dataproductsv1.AssetSelectionQuery{
					Parts: []*dataproductsv1.AssetSelectionQuery_QueryPart{
						{
							Part: &dataproductsv1.AssetSelectionQuery_QueryPart_IdentifierList{
								IdentifierList: &queriesv1.IdentifierList{
									Identifiers: []*entitiesv1.Identifier{
										{
											Id: &entitiesv1.Identifier_SynqPath{
												SynqPath: &entitiesv1.SynqPathIdentifier{
													Path: "dbt-cloud-74734::model.synq_demo.dim_payment_types",
												},
											},
										},
									},
								},
							},

						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("✅ Data product definition created")

}
