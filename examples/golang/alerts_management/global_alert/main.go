package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"slices"

	alertsservicesv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/alerts/services/v1/servicesv1grpc"
	alertsservicesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/services/v1"
	alertsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/v1"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	queriesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/queries/v1"
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

	// Define the FQN
	alertFQN := "example.alerts.critical-failures"
	var alertId string // To store created alert ID
	var originalAlert *alertsv1.Alert

	// ========================================
	// STEP 1: CREATE ALERT
	// ========================================
	{
		fmt.Println("=== Step 1: Creating Alert ===")

		// Create entity query using WithType to match ClickHouse tables
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

		// Configure alert for FATAL severity failures only
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

		resp, err := alertsApi.Create(ctx, &alertsservicesv1.CreateRequest{
			Name:     "Critical Failures Alert",
			Fqn:      alertFQN,
			Trigger:  entityQuery,
			Targets:  targets,
			Settings: alertSettings,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create alert: %v", err))
		}

		alertId = resp.Alert.Id
		fmt.Printf("✓ Created alert: %s\n", resp.Alert.Id)
	}

	// List all alerts and find the one we created
	{
		fmt.Println("\n--- Listing All Alerts ---")

		resp, err := alertsApi.List(ctx, &alertsservicesv1.ListRequest{})
		if err != nil {
			panic(fmt.Sprintf("Failed to list alerts: %v", err))
		}

		if slices.Contains(resp.AlertsIds, alertId) {
			fmt.Printf("✓ Found created alert in list: %s\n", alertId)
		}
	}

	// ========================================
	// STEP 2: UPDATE ALERT
	// ========================================
	{
		fmt.Println("\n=== Step 2: Updating Alert ===")

		// First, get the alert by FQN
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

		if resp.Alerts == nil || resp.Alerts[alertFQN] == nil {
			panic(fmt.Sprintf("No alert found with FQN: %s", alertFQN))
		}
		originalAlert = resp.Alerts[alertFQN]
		fmt.Printf("✓ Retrieved alert: %s\n", originalAlert.Id)

		// Update to include both FATAL and ERROR severities
		updatedSettings := originalAlert.Settings.GetEntityFailure()
		if updatedSettings == nil {
			panic("Alert is not an Entity Failure alert")
		}
		updatedSettings.Severities = append(updatedSettings.Severities, alertsv1.EntityFailureAlertSettings_SEVERITY_ERROR)

		// Update with new name to reflect both severities
		updatedName := "Critical and Error Failures Alert"

		updateResp, err := alertsApi.Update(ctx, &alertsservicesv1.UpdateRequest{
			Identifier: &alertsservicesv1.AlertIdentifier{
				Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			Name: &updatedName,
			Settings: &alertsv1.AlertSettings{
				Settings: &alertsv1.AlertSettings_EntityFailure{
					EntityFailure: updatedSettings,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to update alert: %v", err))
		}
		updatedAlert := updateResp.Alert
		fmt.Printf("✓ Updated alert: %s\n", updatedAlert.Id)

		// Validate that updated alert has both severities
		entityFailureSettings := updatedAlert.Settings.GetEntityFailure()
		if entityFailureSettings == nil {
			panic("Updated alert is not an Entity Failure alert")
		}
		hasFatal := false
		hasError := false
		for _, severity := range entityFailureSettings.Severities {
			if severity == alertsv1.EntityFailureAlertSettings_SEVERITY_FATAL {
				hasFatal = true
			}
			if severity == alertsv1.EntityFailureAlertSettings_SEVERITY_ERROR {
				hasError = true
			}
		}
		if !hasFatal || !hasError {
			panic("Updated alert does not have both FATAL and ERROR severities")
		}
		fmt.Println("✓ Verified alert now has both FATAL and ERROR severities")
	}

	// ========================================
	// STEP 3: TOGGLE ALERT (DISABLE/ENABLE)
	// ========================================
	{
		fmt.Println("\n=== Step 3: Toggling Alert ===")

		// Disable the alert
		fmt.Println("\n--- Disabling Alert ---")
		_, err := alertsApi.ToggleEnabled(ctx, &alertsservicesv1.ToggleEnabledRequest{
			Identifier: &alertsservicesv1.AlertIdentifier{
				Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			IsEnabled: false,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to disable alert: %v", err))
		}

		// Verify disabled
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

		for _, alert := range resp.Alerts {
			if alert.IsDisabled {
				fmt.Println("✓ Alert is disabled")
			} else {
				panic("✗ Alert is still enabled")
			}
		}

		// Re-enable the alert
		fmt.Println("\n--- Re-enabling Alert ---")
		_, err = alertsApi.ToggleEnabled(ctx, &alertsservicesv1.ToggleEnabledRequest{
			Identifier: &alertsservicesv1.AlertIdentifier{
				Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			IsEnabled: true,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to enable alert: %v", err))
		}

		// Verify enabled
		resp, err = alertsApi.BatchGet(ctx, &alertsservicesv1.BatchGetRequest{
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

		for _, alert := range resp.Alerts {
			if !alert.IsDisabled {
				fmt.Println("✓ Alert is enabled")
			} else {
				panic("✗ Alert is still disabled")
			}
		}
	}

	// ========================================
	// STEP 4: DELETE ALERT
	// ========================================
	{
		fmt.Println("\n=== Step 4: Deleting Alert ===")

		// Verify alert exists before deletion
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

		if len(resp.Alerts) == 0 {
			fmt.Println("Alert not found - may have been already deleted")
			return
		}

		// Delete the alert by FQN
		_, err = alertsApi.Delete(ctx, &alertsservicesv1.DeleteRequest{
			Identifier: &alertsservicesv1.AlertIdentifier{
				Identifier: &alertsservicesv1.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to delete alert: %v", err))
		}

		fmt.Println("✓ Alert deleted successfully")

		// Verify alert is deleted
		resp, err = alertsApi.BatchGet(ctx, &alertsservicesv1.BatchGetRequest{
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

		if len(resp.Alerts) == 0 {
			fmt.Println("✓ Verified alert deletion (not found in batch get)")
		} else {
			panic("✗ Warning: Alert may still exist")
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Done! Complete alert lifecycle demonstrated:")
	fmt.Println("  1. Created alert")
	fmt.Println("  2. Updated alert settings")
	fmt.Println("  3. Toggled alert on/off")
	fmt.Println("  4. Deleted alert")
	fmt.Println("========================================")
}
