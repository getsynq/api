package main

import (
	extensionsatlanintegrationsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/extensions/atlan/integrations/v1/integrationsv1grpc"
	extensionsatlanintegrationsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/extensions/atlan/integrations/v1"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"os"
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

	fmt.Printf("Connected to API...\n\n")

	integrationsApi := extensionsatlanintegrationsv1grpc.NewAtlanIntegrationServiceClient(conn)

	// Upsert tenant info and validate.
	{
		tenantUrl := os.Getenv("ATLAN_TENANT_URL")
		tenantApiToken := os.Getenv("ATLAN_API_TOKEN")
		resp, err := integrationsApi.Upsert(ctx, &extensionsatlanintegrationsv1.UpsertRequest{
			AtlanTenantUrl: tenantUrl,
			AtlanApiToken:  tenantApiToken,
		})
		if err != nil {
			panic(err)
		}
		if !resp.Integration.IsValid {
			panic("integration not valid: connection to atlan failed")
		}
	}

	// Fetch integration.
	{
		resp, err := integrationsApi.Get(ctx, &extensionsatlanintegrationsv1.GetRequest{})
		if err != nil {
			panic(err)
		}
		b, _ := json.Marshal(resp.Integration)
		fmt.Printf("Existing integration -> %+v", string(b))
	}

}
