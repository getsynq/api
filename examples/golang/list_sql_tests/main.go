package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"os"

	datacheckssqltestsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/datachecks/sqltests/v1/sqltestsv1grpc"
	datacheckssqltestsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/datachecks/sqltests/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx := context.Background()

	token := os.Getenv("SYNQ_TOKEN")
	host := os.Getenv("API_ENDPOINT")

	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)
	parsedEndpoint, err := url.Parse(fmt.Sprintf("https://%s/", host))
	if err != nil {
		panic(err)
	}

	oauthTokenSource, err := LongLivedTokenSource(ctx, token, parsedEndpoint)
	if err != nil {
		panic(err)
	}
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(oauthTokenSource),
		grpc.WithAuthority(parsedEndpoint.Host),
	}

	conn, err := grpc.DialContext(ctx, apiUrl, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("Connected to API...\n\n")

	sqltestapi := datacheckssqltestsv1grpc.NewSqlTestsServiceClient(conn)

	// List sql tests by annotations.
	{
		listResp, err := sqltestapi.ListSqlTests(ctx, &datacheckssqltestsv1.ListSqlTestsRequest{})
		if err != nil {
			panic(err)
		}

		mappedTests := make(map[string]*datacheckssqltestsv1.SqlTest)
		for _, test := range listResp.SqlTests {
			mappedTests[test.Id] = test
		}

		fmt.Printf("Fetched %d tests:\n", len(mappedTests))
		for id, test := range mappedTests {
			fmt.Printf("Test %s -> %s\n", id, test.String())
		}
		fmt.Println()
	}

}
