package main

import (
	ingestsqlmeshv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/ingest/sqlmesh/v1/sqlmeshv1grpc"
	ingestsqlmeshv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/sqlmesh/v1"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/getsynq/api/examples/golang/ingest_sqlmesh/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
)

func main() {
	ctx := context.Background()

	// EU is developer.synq.io (the default). Set QUALITY_API_ENDPOINT to
	// api.us.synq.io for US or api.au.synq.io for AU.
	host := getenv("QUALITY_API_ENDPOINT", "developer.synq.io")
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	// A long-lived API token, from the API tokens section of Settings > API.
	longLivedToken := os.Getenv("QUALITY_TOKEN")
	if longLivedToken == "" {
		panic("QUALITY_TOKEN must be set")
	}

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

	dbtServiceClient := ingestsqlmeshv1grpc.NewSqlMeshServiceClient(conn)

	res, err := dbtServiceClient.IngestMetadata(ctx, &ingestsqlmeshv1.IngestMetadataRequest{})
	if err != nil {
		panic(err)
	}

	fmt.Println("API worked:")
	fmt.Println(res.String())
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
