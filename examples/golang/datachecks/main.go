package main

import (
	"context"
	"crypto/tls"
	"fmt"

	datacheckssqltestsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/datachecks/sqltests/v1/sqltestsv1grpc"
	"buf.build/gen/go/getsynq/api/grpc/go/synq/datachecks/v1/datachecksv1grpc"
	"buf.build/gen/go/getsynq/api/grpc/go/synq/entities/coordinates/v1/coordinatesv1grpc"
	datacheckssqltestsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/datachecks/sqltests/v1"
	datachecksv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/datachecks/v1"
	coordinatesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/coordinates/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// This example demonstrates how to create a sql test and trigger it on demand.
// It uses the coordinates service to find the target asset (table runs) for the sql test.
// Then it creates the sql test and triggers it on demand.

func main() {
	ctx := context.Background()

	host := "developer.synq.io"
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	clientID := "foo"
	clientSecret := "bar"
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

	sqltestapi := datacheckssqltestsv1grpc.NewSqlTestsServiceClient(conn)

	// find target asset (table runs) for sql test
	coordinatesClient := coordinatesv1grpc.NewDatabaseCoordinatesServiceClient(conn)
	coordResp, err := coordinatesClient.BatchIdsByCoordinates(ctx, &coordinatesv1.BatchIdsByCoordinatesRequest{
		SqlFqn: []string{"runs"},
	})
	if err != nil {
		panic(err)
	}

	if len(coordResp.MatchedCoordinates) == 0 {
		fmt.Printf("No coordinates found\n")
	} else if len(coordResp.MatchedCoordinates) > 1 {
		fmt.Printf("Multiple coordinates found\n")
	}

	tableIdentifier := coordResp.MatchedCoordinates[0].Candidates[0].Identifiers[0]
	// Create sql tests without scheduling (triggering on demand -  no Recurrence Rule).
	{
		_, err := sqltestapi.BatchUpsertSqlTests(ctx, &datacheckssqltestsv1.BatchUpsertSqlTestsRequest{
			SqlTests: []*datacheckssqltestsv1.SqlTest{
				{
					Id:           "8eeebdc0-c033-4878-a387-a9c5e4d68cad",
					Name:         "Workspace not null",
					SaveFailures: false,
					Template: &datacheckssqltestsv1.Template{
						Identifier: tableIdentifier,
						Test: &datacheckssqltestsv1.Template_NotNullTest{
							NotNullTest: &datacheckssqltestsv1.NotNullTest{
								ColumnNames: []string{"workspace"},
							},
						},
					},
				},
			},
		})

		if err != nil {
			panic(err)
		}
	}

	// trigger the sql test
	{
		triggerClient := datachecksv1grpc.NewTriggerServiceClient(conn)
		checkRes, err := triggerClient.TriggerDatachecks(ctx, &datachecksv1.TriggerDatachecksRequest{
			EntityIds: []*entitiesv1.Identifier{
				tableIdentifier,
			},
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Check results: %+v\n\n", checkRes)
	}
}
