package main

import (
	ingestdbtv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/ingest/dbt/v1/dbtv1grpc"
	ingestdbtv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/ingest/dbt/v1"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/getsynq/api/examples/golang/ingest_dbt/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx := context.Background()

	host := "developer.synq.io"
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

	dbtServiceClient := ingestdbtv1grpc.NewDbtServiceClient(conn)

	res, err := dbtServiceClient.IngestInvocation(ctx, &ingestdbtv1.IngestInvocationRequest{
		Args:     []string{"run"},
		ExitCode: 1,
		StdErr:   []byte("Command not found"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("API worked:")
	fmt.Println(res.String())
}
