package main

import (
	extensionsatlanproviderv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/extensions/atlan/provider/v1/providerv1grpc"
	extensionsatlanproviderv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/extensions/atlan/provider/v1"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"os"
)

func main() {
	ctx := context.Background()

	// EU is developer.synq.io (the default). Set QUALITY_API_ENDPOINT to
	// api.us.synq.io for US or api.au.synq.io for AU.
	host := getenv("QUALITY_API_ENDPOINT", "developer.synq.io")
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	clientID := os.Getenv("QUALITY_CLIENT_ID")
	clientSecret := os.Getenv("QUALITY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		panic("QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET must be set")
	}
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

	atlanProviderApi := extensionsatlanproviderv1grpc.NewAtlanProviderServiceClient(conn)

	// Requires the valid integration that 1_setup_integration creates.

	// Fetch visible connections, products and domains.
	{
		resp, err := atlanProviderApi.GetAtlanConnections(ctx, &extensionsatlanproviderv1.GetAtlanConnectionsRequest{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%d Visible Atlan Connections:\n", len(resp.Connections))
		for _, connection := range resp.Connections {
			fmt.Printf(" -> %+v\n", connection)
		}
	}

	{
		resp, err := atlanProviderApi.GetAtlanDataProducts(ctx, &extensionsatlanproviderv1.GetAtlanDataProductsRequest{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%d Visible Atlan DataProducts:\n", len(resp.DataProducts))
		for _, dataProduct := range resp.DataProducts {
			fmt.Printf(" -> %+v\n", dataProduct)
		}
	}

	{
		resp, err := atlanProviderApi.GetAtlanDomains(ctx, &extensionsatlanproviderv1.GetAtlanDomainsRequest{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%d Visible Atlan DataDomains:\n", len(resp.Domains))
		for _, domain := range resp.Domains {
			fmt.Printf(" -> %+v\n", domain)
		}
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
