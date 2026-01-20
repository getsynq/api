package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	alertsservicesv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/alerts/services/v1/servicesv1grpc"
	alertsservicesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/services/v1"
	alertsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	queriesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/queries/v1"
	"github.com/google/uuid"
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

	clientID := os.Getenv("SYNQ_CLIENT_ID")
	clientSecret := os.Getenv("SYNQ_CLIENT_SECRET")
	slackChannel := os.Getenv("SLACK_CHANNEL")

	if clientID == "" || clientSecret == "" {
		panic("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set")
	}
	if slackChannel == "" {
		panic("SLACK_CHANNEL must be set (e.g., 'C1234567890')")
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

	alertsApi := alertsservicesv1grpc.NewAlertsServiceClient(conn)

	// Define alert properties with owner
	alertFQN := "example.alerts.owned-alert"
	ownerPath := "team.data-platform"
	ownershipID := uuid.NewString()
	var createdAlertID string

	// Create a new alert with an owner
	{
		fmt.Println("=== Creating Alert with Owner ===")
		fmt.Printf("FQN: %s\n", alertFQN)
		fmt.Printf("Owner Path: %s\n", ownerPath)
		fmt.Printf("Ownership ID: %s\n\n", ownershipID)

		// Create entity query that matches ClickHouse tables
		entityQuery := &alertsv1.EntityGroupQuery{
			Parts: []*alertsv1.SelectionQuery{
				{
					Parts: []*alertsv1.SelectionQuery_QueryPart{
						{
							Part: &alertsv1.SelectionQuery_QueryPart_WithType{
								WithType: &queriesv1.WithType{
									Types: []*queriesv1.WithType_Type{
										{
											EntityType: &queriesv1.WithType_Type_Default{
												Default: entitiesv1.EntityType_ENTITY_TYPE_CLICKHOUSE_TABLE,
											},
										},
									},
								},
							},
						},
					},
					Operand: queriesv1.QueryOperand_QUERY_OPERAND_AND,
				},
			},
		}

		// Configure alert for FATAL severity failures
		alertSettings := &alertsv1.AlertSettings{
			Settings: &alertsv1.AlertSettings_EntityFailure{
				EntityFailure: &alertsv1.EntityFailureAlertSettings{
					Severities: []alertsv1.EntityFailureAlertSettings_Severity{
						alertsv1.EntityFailureAlertSettings_SEVERITY_FATAL,
					},
					NotifyUpstream:        false,
					AllowSqlTestAuditLink: true,
					Ongoing: &alertsv1.OngoingAlertsStrategy{
						Strategy: &alertsv1.OngoingAlertsStrategy_Disabled_{
							Disabled: &alertsv1.OngoingAlertsStrategy_Disabled{},
						},
					},
				},
			},
		}

		// Configure Slack target
		targets := []*alertsv1.AlertingTarget{
			{
				Target: &alertsv1.AlertingTarget_Slack{
					Slack: &alertsv1.SlackTarget{
						Channel: slackChannel,
					},
				},
			},
		}

		// Create the alert with owner information
		resp, err := alertsApi.Create(ctx, &alertsservicesv1.CreateRequest{
			Name:     "Owned Alert Example",
			Fqn:      alertFQN,
			Trigger:  entityQuery,
			Targets:  targets,
			Settings: alertSettings,
			Owner: &alertsv1.Alert_Owner{
				OwnerPath:   ownerPath,
				OwnershipId: ownershipID,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create alert with owner: %v", err))
		}

		createdAlertID = resp.Alert.Id
		fmt.Printf("✓ Alert created successfully: %s\n", createdAlertID)

		if resp.Alert.Owner == nil {
			panic("Alert was created but owner was not set")
		}

		fmt.Printf("  Owner Path: %s\n", resp.Alert.Owner.OwnerPath)
		fmt.Printf("  Ownership ID: %s\n\n", resp.Alert.Owner.OwnershipId)
	}

	// List alerts filtered by owner
	{
		fmt.Println("=== Listing Alerts by Owner ===")
		fmt.Printf("Filtering by owner: %s (ownership: %s)\n\n", ownerPath, ownershipID)

		resp, err := alertsApi.List(ctx, &alertsservicesv1.ListRequest{
			Owner: &alertsv1.Alert_Owner{
				OwnerPath:   ownerPath,
				OwnershipId: ownershipID,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to list alerts by owner: %v", err))
		}

		fmt.Printf("Found %d alert(s) for this owner:\n", len(resp.AlertsIds))
		for i, alertID := range resp.AlertsIds {
			fmt.Printf("  %d. %s\n", i+1, alertID)
		}
		fmt.Println()

		// Verify our alert is in the list
		found := false
		for _, alertID := range resp.AlertsIds {
			if alertID == createdAlertID {
				found = true
				break
			}
		}

		if found {
			fmt.Printf("✓ Our alert '%s' was found in the owner's alerts list\n\n", createdAlertID)
		} else {
			panic("Our alert was not found in the owner's alerts list")
		}
	}

	// Verify the alert owner by getting it directly
	{
		fmt.Println("=== Verifying Alert Owner ===")

		resp, err := alertsApi.BatchGet(ctx, &alertsservicesv1.BatchGetRequest{
			Identifiers: []*alertsservicesv1.AlertIdentifier{
				{
					Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
						Fqn: alertFQN,
					},
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to get alert: %v", err))
		}

		alert := resp.Alerts[alertFQN]
		if alert == nil {
			panic("Alert not found")
		}

		if alert.Owner == nil {
			panic("Alert has no owner")
		}

		if alert.Owner.OwnerPath != ownerPath {
			panic(fmt.Sprintf("Owner path mismatch: expected %s, got %s", ownerPath, alert.Owner.OwnerPath))
		}

		if alert.Owner.OwnershipId != ownershipID {
			panic(fmt.Sprintf("Ownership ID mismatch: expected %s, got %s", ownershipID, alert.Owner.OwnershipId))
		}

		fmt.Println("✓ Alert owner verified successfully")
		fmt.Printf("  Owner Path: %s\n", alert.Owner.OwnerPath)
		fmt.Printf("  Ownership ID: %s\n\n", alert.Owner.OwnershipId)
	}

	// Delete the created alert
	{
		fmt.Println("=== Cleaning Up: Deleting Alert ===")

		_, err := alertsApi.Delete(ctx, &alertsservicesv1.DeleteRequest{
			Identifier: &alertsservicesv1.AlertIdentifier{
				Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to delete alert: %v", err))
		}

		fmt.Printf("✓ Alert deleted successfully: %s\n", alertFQN)
	}

	// Verify alert was deleted
	{
		fmt.Println("\n=== Verifying Deletion ===")

		resp, err := alertsApi.BatchGet(ctx, &alertsservicesv1.BatchGetRequest{
			Identifiers: []*alertsservicesv1.AlertIdentifier{
				{
					Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
						Fqn: alertFQN,
					},
				},
			},
		})

		if err != nil {
			fmt.Printf("✓ Alert no longer exists (error getting it: %v)\n", err)
		} else if len(resp.Alerts) == 0 || resp.Alerts[alertFQN] == nil {
			fmt.Println("✓ Alert successfully deleted (not found in batch get)")
		} else {
			panic("✗ Warning: Alert may still exist")
		}
	}

	fmt.Println("\nDone! Alert with owner created, verified, and cleaned up successfully.")
}
