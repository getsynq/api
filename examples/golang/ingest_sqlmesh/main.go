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
)

func main() {
	ctx := context.Background()

	host := "developer.synq.dev"
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	longLivedToken := "st-long-lived-token"

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
