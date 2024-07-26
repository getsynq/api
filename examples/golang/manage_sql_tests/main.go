package main

import (
	datacheckssqltestsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/datachecks/sqltests/v1/sqltestsv1grpc"
	datacheckssqltestsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/datachecks/sqltests/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	platformsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/platforms/v1"
	"context"
	"crypto/tls"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

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

	// Create sql tests.
	{
		upsertRes, err := sqltestapi.BatchUpsertSqlTests(ctx, &datacheckssqltestsv1.BatchUpsertSqlTestsRequest{
			SqlTests: []*datacheckssqltestsv1.SqlTest{
				{
					Platform: &platformsv1.DataPlatformIdentifier{
						Id: &platformsv1.DataPlatformIdentifier_Clickhouse{
							Clickhouse: &platformsv1.ClickhouseIdentifier{
								Host:   "xyz.clickhouse.cloud",
								Schema: "prod",
							},
						},
					},
					Id:             "/clickhouse/prod/check-created-at-not-null",
					Name:           "CH/alertsv2 message_id not null",
					SqlExpression:  "select * from alerts.alert_settings where workspace is null",
					RecurrenceRule: "FREQ=HOURLY;INTERVAL=3",
					SaveFailures:   true,
					Annotations: []*entitiesv1.Annotation{
						{
							Name:   "env",
							Values: []string{"prod"},
						},
						{
							Name:   "owners",
							Values: []string{"engineering", "karan"},
						},
					},
				},
				{
					Platform: &platformsv1.DataPlatformIdentifier{
						Id: &platformsv1.DataPlatformIdentifier_Postgres{
							Postgres: &platformsv1.PostgresIdentifier{
								Host:     "12.34.123.45",
								Database: "kernel-alertsv2",
							},
						},
					},
					Id:             "/postgres/prod/alerts/mutes/created-at-not-null",
					Name:           "Postgres/alerts mutes created_at not null",
					SqlExpression:  "select * from public.mutes where created_at is null",
					RecurrenceRule: "FREQ=HOURLY;INTERVAL=3",
					SaveFailures:   true,
					Annotations: []*entitiesv1.Annotation{
						{
							Name:   "env",
							Values: []string{"prod"},
						},
						{
							Name:   "owners",
							Values: []string{"petr"},
						},
					},
				},
			},
		})

		if err != nil {
			panic(err)
		}

		if len(upsertRes.Errors) > 0 {
			panic(fmt.Errorf("%s -> %v", upsertRes.Errors[0].Id, upsertRes.Errors[0].Reason))
		}

		if len(upsertRes.CreatedIds)+len(upsertRes.UpdatedIds) != 2 {
			panic("all tests were not created")
		}

		fmt.Printf("Created tests: %+v\n\n", append(upsertRes.CreatedIds, upsertRes.GetUpdatedIds()...))
	}

	// Update sql tests by IDs.
	{
		upsertRes, err := sqltestapi.BatchUpsertSqlTests(ctx, &datacheckssqltestsv1.BatchUpsertSqlTestsRequest{
			SqlTests: []*datacheckssqltestsv1.SqlTest{
				{
					Platform: &platformsv1.DataPlatformIdentifier{
						Id: &platformsv1.DataPlatformIdentifier_Clickhouse{
							Clickhouse: &platformsv1.ClickhouseIdentifier{
								Host:   "xyz.clickhouse.cloud",
								Schema: "prod",
							},
						},
					},
					Id:             "/clickhouse/prod/check-created-at-not-null",
					Name:           "CH/alertsv2 message_id not null",
					SqlExpression:  "select * from alerts.alert_settings where workspace is null",
					RecurrenceRule: "FREQ=HOURLY;INTERVAL=6",
					SaveFailures:   true,
					Annotations: []*entitiesv1.Annotation{
						{
							Name:   "env",
							Values: []string{"prod"},
						},
						{
							Name:   "owners",
							Values: []string{"engineering", "karan"},
						},
					},
				},
			},
		})

		if err != nil {
			panic(err)
		}

		if len(upsertRes.Errors) > 0 {
			panic(upsertRes.Errors[0].Reason)
		}

		if len(upsertRes.UpdatedIds) != 1 {
			panic("all tests were not updated")
		}

		fmt.Printf("Updated tests: %+v\n\n", upsertRes.UpdatedIds)
	}

	// List sql tests by annotations.
	{
		listResp, err := sqltestapi.ListSqlTests(ctx, &datacheckssqltestsv1.ListSqlTestsRequest{
			Annotations: []*entitiesv1.Annotation{
				{Name: "env", Values: []string{"prod"}},
			},
		})

		if err != nil {
			panic(err)
		}

		mappedTests := make(map[string]*datacheckssqltestsv1.SqlTest)
		for _, test := range listResp.SqlTests {
			mappedTests[test.Id] = test
		}

		if mappedTests["/clickhouse/prod/check-created-at-not-null"] == nil {
			panic("missing test /clickhouse/prod/check-created-at-not-null in list response")
		}

		if mappedTests["/postgres/prod/alerts/mutes/created-at-not-null"] == nil {
			panic("missing test /postgres/prod/alerts/mutes/created-at-not-null in list response")
		}

		fmt.Println("Fetched tests by annotation:")
		for id, test := range mappedTests {
			fmt.Printf("Test %s -> %s\n", id, test.String())
		}
		fmt.Println()
	}

	// Delete sql tests by IDs.
	{
		_, err := sqltestapi.BatchDeleteSqlTests(ctx, &datacheckssqltestsv1.BatchDeleteSqlTestsRequest{
			Ids: []string{"/postgres/prod/alerts/mutes/created-at-not-null", "/clickhouse/prod/check-created-at-not-null"},
		})

		if err != nil {
			panic(err)
		}

		fmt.Println("Deleted tests.")
	}

}
