package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	entitiesschemasv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/entities/schemas/v1/schemasv1grpc"
	entitiesschemasv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/schemas/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
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

	schemasServiceClient := entitiesschemasv1grpc.NewSchemasServiceClient(conn)

	getSchemaResponse, err := schemasServiceClient.GetSchema(ctx, &entitiesschemasv1.GetSchemaRequest{
		Id: &entitiesv1.Identifier{
			Id: &entitiesv1.Identifier_SynqPath{
				SynqPath: &entitiesv1.SynqPathIdentifier{
					Path: "ch-prod::default::runs",
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("getSchemaResponse:")
	fmt.Println(getSchemaResponse.String())
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
