package main

import (
	"context"
	"crypto/tls"
	"fmt"

	datacheckssqltestsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/datachecks/sqltests/v1/sqltestsv1grpc"
	"buf.build/gen/go/getsynq/api/grpc/go/synq/datachecks/v1/datachecksv1grpc"
	"buf.build/gen/go/getsynq/api/grpc/go/synq/entities/coordinates/v1/coordinatesv1grpc"
	custommonitorsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/monitors/custom_monitors/v1/custom_monitorsv1grpc"
	datacheckssqltestsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/datachecks/sqltests/v1"
	datachecksv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/datachecks/v1"
	coordinatesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/coordinates/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	custommonitorsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/monitors/custom_monitors/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// This example demonstrates two patterns for on-demand datachecks:
//
//  1. Create a SQL test without scheduling and trigger it on demand.
//  2. Create a custom monitor with the on_demand schedule and trigger it on demand.
//
// Both are triggered with a single TriggerDatachecks call. The response contains
// SqlTestResult entries for SQL tests and MonitorResult entries for custom monitors.
//
// Required auth scopes:
//   - SCOPE_ENTITY_READ — resolve table FQNs to entity identifiers
//   - SCOPE_DATACHECKS_SQLTESTS_EDIT — create/update SQL tests
//   - SCOPE_DATACHECKS_TRIGGER — trigger SQL tests and custom monitors
//   - SCOPE_MONITORS_CUSTOM_EDIT — create/update custom monitors

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

	conn, err := grpc.NewClient(apiUrl, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("Connected to API...\n\n")

	sqltestapi := datacheckssqltestsv1grpc.NewSqlTestsServiceClient(conn)

	// Resolve the target table FQN to an entity identifier.
	coordinatesClient := coordinatesv1grpc.NewDatabaseCoordinatesServiceClient(conn)
	coordResp, err := coordinatesClient.BatchIdsByCoordinates(ctx, &coordinatesv1.BatchIdsByCoordinatesRequest{
		SqlFqn: []string{"runs"},
	})
	if err != nil {
		panic(err)
	}
	if len(coordResp.MatchedCoordinates) == 0 {
		panic("no entity found for the given table FQN")
	}

	tableIdentifier := coordResp.MatchedCoordinates[0].Candidates[0].Identifiers[0]

	// Step 1: Create a SQL test without scheduling (on-demand — no Recurrence Rule).
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

	// Step 2: Create a custom monitor with on_demand schedule.
	//
	// Use the on_demand schedule to disable automatic scheduling — the monitor will only
	// run when explicitly triggered via TriggerDatachecks. This is the recommended pattern
	// for pipeline integration: create the monitor once, then call TriggerDatachecks after
	// each table update to run checks synchronously and fail the pipeline on violations.
	//
	// Compare schedule options:
	//   on_demand  — only runs when triggered via TriggerDatachecks (no automatic runs)
	//   daily      — runs automatically once per day at a configured time
	//   hourly     — runs automatically once per hour at a configured minute
	//
	// table_stats monitors row count, null rate, and uniqueness for all columns without
	// requiring a time partitioning column. This makes them suitable for any table.
	{
		customMonitorClient := custommonitorsv1grpc.NewCustomMonitorsServiceClient(conn)
		_, err := customMonitorClient.BatchCreateMonitor(ctx, &custommonitorsv1.BatchCreateMonitorRequest{
			Monitors: []*custommonitorsv1.MonitorDefinition{
				{
					Id:          "c3d4e5f6-a7b8-9012-cdef-123456789012",
					Name:        "Table stats on runs",
					MonitoredId: tableIdentifier,
					// table_stats does not require time_partitioning.
					Monitor: &custommonitorsv1.MonitorDefinition_TableStats{
						TableStats: &custommonitorsv1.MonitorTableStats{},
					},
					Mode: &custommonitorsv1.MonitorDefinition_AnomalyEngine{
						AnomalyEngine: &custommonitorsv1.ModeAnomalyEngine{
							Sensitivity: custommonitorsv1.Sensitivity_SENSITIVITY_BALANCED,
						},
					},
					// on_demand disables automatic scheduling. The monitor only runs
					// when explicitly triggered via TriggerDatachecks.
					Schedule: &custommonitorsv1.MonitorDefinition_OnDemand{
						OnDemand: &custommonitorsv1.ScheduleOnDemand{},
					},
				},
			},
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Custom monitor created\n\n")
	}

	// Step 3: Trigger all datachecks on the entity in a single call.
	//
	// TriggerDatachecks runs every datacheck — SQL tests and custom monitors — associated
	// with the given entities. Batching all entities into one call avoids rate limiting.
	// The response contains a DatacheckResult per check; use GetSqlTestResult() and
	// GetMonitorResult() to distinguish between the two types.
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

		for _, result := range checkRes.Results {
			if st := result.GetSqlTestResult(); st != nil {
				fmt.Printf("SQL test %s: status=%s\n", st.SqlTestId, result.Status)
			}
			if mr := result.GetMonitorResult(); mr != nil {
				fmt.Printf("Monitor %s: status=%s\n", mr.MonitorId, result.Status)
				for _, prediction := range mr.Predictions {
					fmt.Printf("  prediction: field=%s status=%s value=%.2f\n",
						prediction.Field, prediction.Status, prediction.Value)
				}
			}
			// Fail the pipeline if any check detected a violation.
			if result.Status == datachecksv1.DatacheckStatus_DATACHECK_STATUS_FAILED {
				panic("datacheck FAILED — aborting pipeline")
			}
		}
	}
}
