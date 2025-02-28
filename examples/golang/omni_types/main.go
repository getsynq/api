package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	entitiescustomv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/custom/v1/customv1grpc"
	entitiescustomv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/custom/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
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

	clientID := os.Getenv("SYNQ_CLIENT_ID")
	clientSecret := os.Getenv("SYNQ_CLIENT_SECRET")
	tokenURL := fmt.Sprintf("https://%s/oauth2/token", host)
	clickhouseHost := "prod"

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

	typesServiceClient := entitiescustomv1grpc.NewTypesServiceClient(conn)
	entitiesapi := entitiescustomv1grpc.NewEntitiesServiceClient(conn)
	relationshipsapi := entitiescustomv1grpc.NewRelationshipsServiceClient(conn)

	// Define custom entity types
	omniCustomType := 30
	{
		_, err := typesServiceClient.UpsertType(ctx, &entitiescustomv1.UpsertTypeRequest{
			Type: &entitiesv1.Type{
				TypeId:  int32(omniCustomType),
				Name:    "Omni Dashboard",
				SvgIcon: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512"><!--!Font Awesome Free 6.7.2 by @fontawesome - https://fontawesome.com License - https://fontawesome.com/license/free Copyright 2025 Fonticons, Inc.--><path d="M224 96a160 160 0 1 0 0 320 160 160 0 1 0 0-320zM448 256A224 224 0 1 1 0 256a224 224 0 1 1 448 0z"/></svg>`),
			},
		})
		if err != nil {
			panic(err)
		}
		fmt.Println("Created Omni Dashboard entity type...\n")
	}

	// Fetch documents from Omni
	omniUrl := os.Getenv("OMNI_BASE_URL")
	omniApiKey := os.Getenv("OMNI_API_KEY")

	records := []*Document{}
	{
		client := &http.Client{}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/unstable/documents", omniUrl), nil)
		if err != nil {
			panic(err)
		}

		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", omniApiKey))
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)

		// Process response
		documentsResponse := DocumentsResponse{}
		json.Unmarshal(b, &documentsResponse)
		records = documentsResponse.Records
	}

	for _, record := range records {
		fmt.Printf("Processing Omni Dashboard %s\n", record.Identifier)

		client := &http.Client{}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/unstable/documents/%s/export", omniUrl, record.Identifier), nil)
		if err != nil {
			panic(err)
		}

		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", omniApiKey))
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)

		// Process response
		exportResp := DocumentExportResponse{}
		json.Unmarshal(b, &exportResp)

		fmt.Printf("Omni Dashboard Connection Dialect - %s\n", exportResp.Document.Connection.Dialect)
		if exportResp.Document.Connection.Dialect != "clickhouse" {
			fmt.Printf("Skipping dashboard on non-clickhouse connection - %s\n", record.Identifier)
			continue
		}

		identifier := &entitiesv1.Identifier_Custom{
			Custom: &entitiesv1.CustomIdentifier{
				Id: fmt.Sprintf("omni::dashboard::%s", record.Identifier),
			},
		}

		if _, err := entitiesapi.UpsertEntity(ctx, &entitiescustomv1.UpsertEntityRequest{
			Entity: &entitiesv1.Entity{
				Id: &entitiesv1.Identifier{
					Id: identifier,
				},
				TypeId:    int32(omniCustomType),
				Name:      exportResp.Dashboard.Name,
				CreatedAt: timestamppb.Now(),
			},
		}); err != nil {
			panic(err)
		}
		fmt.Printf("Upserted custom Omni Dashboard entity.\n")

		relationships := []*entitiescustomv1.Relationship{}
		for _, member := range exportResp.Dashboard.QueryPresentationCollection.QueryPresentationCollectionMemberships {
			relationships = append(relationships, &entitiescustomv1.Relationship{
				Upstream: &entitiesv1.Identifier{
					Id: &entitiesv1.Identifier_ClickhouseTable{
						ClickhouseTable: &entitiesv1.ClickhouseTableIdentifier{
							Host:   clickhouseHost,
							Schema: exportResp.Document.Connection.Database,
							Table:  member.QueryPresentation.Query.QueryJson.Table,
						},
					},
				},
				Downstream: &entitiesv1.Identifier{
					Id: identifier,
				},
			})
		}
		if len(relationships) > 0 {
			if _, err := relationshipsapi.UpsertRelationships(ctx, &entitiescustomv1.UpsertRelationshipsRequest{
				Relationships: relationships,
			}); err != nil {
				panic(err)
			}
			fmt.Printf("Upserted custom Omni Dashboard entity relationships.\n")
		}
	}
}
