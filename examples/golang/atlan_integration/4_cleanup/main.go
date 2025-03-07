package main

import (
	extensionsatlanintegrationsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/extensions/atlan/integrations/v1/integrationsv1grpc"
	extensionsatlanintegrationsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/extensions/atlan/integrations/v1"
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

	// Deactivate integraton.
	// This stops the scheduled synchronization.
	{
		_, err := integrationsApi.Activate(ctx, &extensionsatlanintegrationsv1.ActivateRequest{
			Activate: false,
		})
		if err != nil {
			panic(err)
		}
		fmt.Println("Deactivated Atlan integration.")
	}

	// Optionally delete integration.
	{
		_, err := integrationsApi.Remove(ctx, &extensionsatlanintegrationsv1.RemoveRequest{})
		if err != nil {
			panic(err)
		}
		fmt.Println("Removed Atlan integration.")
	}

	// Note that the domains and products created on SYNQ by the synchronization
	// are *NOT* automatically removed. You can remove them from SYNQ if you wish to.
}
