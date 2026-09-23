package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"slices"

	alertsservicesv2grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/alerts/services/v2/servicesv2grpc"
	alertsservicesv2 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/services/v2"
	alertsv2 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/v2"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	queriesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/queries/v1"
	synqv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/v1"
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

	alertsApi := alertsservicesv2grpc.NewAlertsServiceClient(conn)

	// Define the FQN
	alertFQN := "example.alerts.critical-failures"
	var alertId string // To store created alert ID

	// ========================================
	// STEP 1: CREATE ALERT
	// ========================================
	{
		fmt.Println("=== Step 1: Creating Alert ===")

		// A structured query trigger; owned_alert shows the ResolverQL alternative.
		trigger := &alertsv2.Trigger{
			Query: &queriesv1.Query{
				Parts: []*queriesv1.Query_QueryPart{
					{
						Part: &queriesv1.Query_QueryPart_WithType{
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
		}

		// Configure alert for FATAL severity failures only
		settings := &alertsv2.IssueAlertSettings{
			Severities: []synqv1.Severity{
				synqv1.Severity_SEVERITY_FATAL,
			},
			NotifyUpstream:        false,
			AllowSqlTestAuditLink: true,
			Ongoing: &alertsv2.OngoingAlertsStrategy{
				Strategy: &alertsv2.OngoingAlertsStrategy_Disabled_{
					Disabled: &alertsv2.OngoingAlertsStrategy_Disabled{},
				},
			},
			Grouping: &alertsv2.IssueGroupingStrategy{
				Strategy: &alertsv2.IssueGroupingStrategy_SystemDetected_{
					SystemDetected: &alertsv2.IssueGroupingStrategy_SystemDetected{},
				},
			},
		}

		// Configure Slack target
		targets := []*alertsv2.AlertingTarget{
			{
				Target: &alertsv2.AlertingTarget_Slack{
					Slack: &alertsv2.SlackTarget{
						Channel: slackChannel,
					},
				},
			},
		}

		resp, err := alertsApi.Create(ctx, &alertsservicesv2.CreateRequest{
			Name: "Critical Failures Alert",
			Fqn:  alertFQN,
			Kind: &alertsservicesv2.CreateRequest_IssueLifecycle{
				IssueLifecycle: &alertsv2.IssueLifecycleAlert{
					Trigger:  trigger,
					Targets:  targets,
					Settings: settings,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create alert: %v", err))
		}

		alertId = resp.Alert.Id
		fmt.Printf("✓ Created alert: %s\n", resp.Alert.Id)

		// The structured selection reads back as canonical ResolverQL.
		fmt.Printf("  Selection (ResolverQL): %s\n", resp.Alert.GetIssueLifecycle().GetTrigger().GetRenderedResolverQl())
	}

	// List all alerts and find the one we created
	{
		fmt.Println("\n--- Listing All Alerts ---")

		resp, err := alertsApi.List(ctx, &alertsservicesv2.ListRequest{})
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
		resp, err := alertsApi.BatchGet(ctx, &alertsservicesv2.BatchGetRequest{
			Identifiers: []*alertsservicesv2.AlertIdentifier{
				{
					Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
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
		originalAlert := resp.Alerts[alertFQN]
		fmt.Printf("✓ Retrieved alert: %s\n", originalAlert.Id)

		updatedName := "Critical and Error Failures Alert"
		if _, err := alertsApi.Rename(ctx, &alertsservicesv2.RenameRequest{
			Identifier: &alertsservicesv2.AlertIdentifier{
				Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			Name: updatedName,
		}); err != nil {
			panic(fmt.Sprintf("Failed to rename alert: %v", err))
		}
		fmt.Println("✓ Renamed alert")

		// UpdateSettings is intent-specific: it leaves the trigger unchanged.
		updatedSettings := originalAlert.GetIssueLifecycle().GetSettings()
		if updatedSettings == nil {
			panic("Alert is not an issue lifecycle alert")
		}
		updatedSettings.Severities = append(updatedSettings.Severities, synqv1.Severity_SEVERITY_ERROR)

		updateResp, err := alertsApi.UpdateSettings(ctx, &alertsservicesv2.UpdateSettingsRequest{
			Identifier: &alertsservicesv2.AlertIdentifier{
				Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			Settings: &alertsservicesv2.UpdateSettingsRequest_IssueLifecycle{
				IssueLifecycle: updatedSettings,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to update alert settings: %v", err))
		}
		fmt.Printf("✓ Updated alert: %s\n", updateResp.Alert.Id)

		// Validate that updated alert has both severities
		issueSettings := updateResp.Alert.GetIssueLifecycle().GetSettings()
		if issueSettings == nil {
			panic("Updated alert is not an issue lifecycle alert")
		}
		hasFatal := slices.Contains(issueSettings.Severities, synqv1.Severity_SEVERITY_FATAL)
		hasError := slices.Contains(issueSettings.Severities, synqv1.Severity_SEVERITY_ERROR)
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
		_, err := alertsApi.ToggleEnabled(ctx, &alertsservicesv2.ToggleEnabledRequest{
			Identifier: &alertsservicesv2.AlertIdentifier{
				Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			IsEnabled: false,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to disable alert: %v", err))
		}

		// Verify disabled
		resp, err := alertsApi.BatchGet(ctx, &alertsservicesv2.BatchGetRequest{
			Identifiers: []*alertsservicesv2.AlertIdentifier{
				{
					Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
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
		_, err = alertsApi.ToggleEnabled(ctx, &alertsservicesv2.ToggleEnabledRequest{
			Identifier: &alertsservicesv2.AlertIdentifier{
				Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
			IsEnabled: true,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to enable alert: %v", err))
		}

		// Verify enabled
		resp, err = alertsApi.BatchGet(ctx, &alertsservicesv2.BatchGetRequest{
			Identifiers: []*alertsservicesv2.AlertIdentifier{
				{
					Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
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
		resp, err := alertsApi.BatchGet(ctx, &alertsservicesv2.BatchGetRequest{
			Identifiers: []*alertsservicesv2.AlertIdentifier{
				{
					Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
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
		_, err = alertsApi.Delete(ctx, &alertsservicesv2.DeleteRequest{
			Identifier: &alertsservicesv2.AlertIdentifier{
				Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
					Fqn: alertFQN,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to delete alert: %v", err))
		}

		fmt.Println("✓ Alert deleted successfully")

		// Verify alert is deleted
		resp, err = alertsApi.BatchGet(ctx, &alertsservicesv2.BatchGetRequest{
			Identifiers: []*alertsservicesv2.AlertIdentifier{
				{
					Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
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
