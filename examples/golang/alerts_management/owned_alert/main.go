package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	alertsservicesv2grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/alerts/services/v2/servicesv2grpc"
	alertsservicesv2 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/services/v2"
	alertsv2 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/v2"
	synqv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/v1"
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

	alertsApi := alertsservicesv2grpc.NewAlertsServiceClient(conn)

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

		// A ResolverQL trigger, the alternative to global_alert's structured query; it takes precedence when both are set.
		trigger := &alertsv2.Trigger{
			ResolverQl: `with_type("clickhouse_table")`,
		}

		// Configure alert for FATAL severity failures
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

		// The owner lives on the alert kind, fixed at creation time.
		resp, err := alertsApi.Create(ctx, &alertsservicesv2.CreateRequest{
			Name: "Owned Alert Example",
			Fqn:  alertFQN,
			Kind: &alertsservicesv2.CreateRequest_IssueLifecycle{
				IssueLifecycle: &alertsv2.IssueLifecycleAlert{
					Trigger:  trigger,
					Targets:  targets,
					Settings: settings,
					Owner: &alertsv2.Owner{
						OwnerPath:   ownerPath,
						OwnershipId: ownershipID,
					},
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create alert with owner: %v", err))
		}

		createdAlertID = resp.Alert.Id
		fmt.Printf("✓ Alert created successfully: %s\n", createdAlertID)

		owner := resp.Alert.GetIssueLifecycle().GetOwner()
		if owner == nil {
			panic("Alert was created but owner was not set")
		}

		fmt.Printf("  Owner Path: %s\n", owner.OwnerPath)
		fmt.Printf("  Ownership ID: %s\n\n", owner.OwnershipId)
	}

	// List alerts filtered by owner
	{
		fmt.Println("=== Listing Alerts by Owner ===")
		fmt.Printf("Filtering by owner: %s (ownership: %s)\n\n", ownerPath, ownershipID)

		resp, err := alertsApi.List(ctx, &alertsservicesv2.ListRequest{
			Owner: &alertsv2.Owner{
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

		alert := resp.Alerts[alertFQN]
		if alert == nil {
			panic("Alert not found")
		}

		owner := alert.GetIssueLifecycle().GetOwner()
		if owner == nil {
			panic("Alert has no owner")
		}

		if owner.OwnerPath != ownerPath {
			panic(fmt.Sprintf("Owner path mismatch: expected %s, got %s", ownerPath, owner.OwnerPath))
		}

		if owner.OwnershipId != ownershipID {
			panic(fmt.Sprintf("Ownership ID mismatch: expected %s, got %s", ownershipID, owner.OwnershipId))
		}

		fmt.Println("✓ Alert owner verified successfully")
		fmt.Printf("  Owner Path: %s\n", owner.OwnerPath)
		fmt.Printf("  Ownership ID: %s\n\n", owner.OwnershipId)
	}

	// Delete the created alert
	{
		fmt.Println("=== Cleaning Up: Deleting Alert ===")

		_, err := alertsApi.Delete(ctx, &alertsservicesv2.DeleteRequest{
			Identifier: &alertsservicesv2.AlertIdentifier{
				Identifier: &alertsservicesv2.AlertIdentifier_Fqn{
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
			fmt.Printf("✓ Alert no longer exists (error getting it: %v)\n", err)
		} else if len(resp.Alerts) == 0 || resp.Alerts[alertFQN] == nil {
			fmt.Println("✓ Alert successfully deleted (not found in batch get)")
		} else {
			panic("✗ Warning: Alert may still exist")
		}
	}

	fmt.Println("\nDone! Alert with owner created, verified, and cleaned up successfully.")
}
