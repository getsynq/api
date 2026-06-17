// Demonstrates the IntegrationsService features that warehouse-type
// integrations add on top of plain CRUD, using BigQuery as the example:
//
//	create -> read generated outputs -> refresh -> health (+ run history) -> delete
//
// Unlike dbt Cloud, a warehouse integration is "refreshable" (Capabilities.
// can_refresh == true) and the server derives read-only Outputs from the
// supplied credentials (for BigQuery, the service-account email to grant
// dataset access to).
//
// The service-account key can be invalid for this demo: create still succeeds
// and the management calls work end to end. A bad key simply surfaces later as
// an unhealthy status from GetIntegrationHealth.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	integrationsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/integrations/v1/integrationsv1grpc"
	integrationsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/integrations/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func main() {
	ctx := context.Background()

	host := getenv("API_ENDPOINT", "developer.synq.io")
	apiURL := fmt.Sprintf("%s:443", host)

	clientID := os.Getenv("SYNQ_CLIENT_ID")
	clientSecret := os.Getenv("SYNQ_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		panic("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set (scope: Manage Integrations)")
	}

	// BigQuery connection settings. Defaults are illustrative placeholders.
	// BIGQUERY_SA_KEY may point at a file (path) or hold the JSON inline; an
	// invalid key still lets every management call below succeed.
	projectID := getenv("BIGQUERY_PROJECT_ID", "example-project")
	region := getenv("BIGQUERY_REGION", "EU")
	saKey := loadServiceAccountKey()

	// --- Connect ---
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("https://%s/oauth2/token", host),
	}
	tokenSource := oauth.TokenSource{TokenSource: config.TokenSource(ctx)}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(tokenSource),
		grpc.WithAuthority(host),
	}
	conn, err := grpc.DialContext(ctx, apiURL, opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Printf("Connected to %s\n\n", host)

	client := integrationsv1grpc.NewIntegrationsServiceClient(conn)

	// --- Step 1: Create ---
	fmt.Println("=== Step 1: Create BigQuery integration ===")
	created, err := client.CreateIntegration(ctx, &integrationsv1.CreateIntegrationRequest{
		Title: "Example BigQuery",
		Config: &integrationsv1.IntegrationConfig{
			Config: &integrationsv1.IntegrationConfig_Bigquery{
				Bigquery: &integrationsv1.BigQueryCloudConf{
					ProjectId:         proto.String(projectID),
					Region:            proto.String(region),
					ServiceAccountKey: proto.String(saKey),
					// Datasets left empty: discover all visible datasets.
				},
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("create failed: %w", err))
	}
	integration := created.GetIntegration()
	id := integration.GetId()
	fmt.Printf("Created integration id=%s etag=%s\n", id, integration.GetEtag())
	fmt.Printf("Service account key in response (masked): %q\n\n",
		integration.GetConfig().GetBigquery().GetServiceAccountKey())

	// --- Step 2: Generated outputs + capabilities ---
	// Outputs are read-only values the server derives on create. For BigQuery
	// that is the service-account email you grant dataset access to.
	fmt.Println("=== Step 2: Outputs & capabilities ===")
	fmt.Printf("Service account to grant access: %q\n",
		integration.GetOutputs().GetBigquery().GetServiceAccountEmail())
	caps := integration.GetCapabilities()
	fmt.Printf("can_refresh=%v can_disable=%v can_delete=%v\n\n",
		caps.GetCanRefresh(), caps.GetCanDisable(), caps.GetCanDelete())

	// --- Step 3: Refresh ---
	// Only valid when Capabilities.can_refresh is true (warehouse types). Each
	// call enqueues a fresh ad-hoc refresh.
	fmt.Println("=== Step 3: Trigger refresh ===")
	if caps.GetCanRefresh() {
		if _, err := client.RefreshIntegration(ctx, &integrationsv1.RefreshIntegrationRequest{IntegrationId: id}); err != nil {
			panic(fmt.Errorf("refresh failed: %w", err))
		}
		fmt.Printf("Refresh enqueued\n\n")
	} else {
		fmt.Printf("Type does not support refresh\n\n")
	}

	// --- Step 4: Health + run history ---
	// Omitting pagination returns a bounded recent window (last 7 days).
	fmt.Println("=== Step 4: Health ===")
	health, err := client.GetIntegrationHealth(ctx, &integrationsv1.GetIntegrationHealthRequest{IntegrationId: id})
	if err != nil {
		panic(err)
	}
	fmt.Printf("status=%s healthy=%v message=%q\n",
		health.GetHealth().GetStatus(),
		health.GetHealth().GetHealthy(),
		health.GetHealth().GetMessage())
	for _, run := range health.GetRuns() {
		fmt.Printf("  run %s status=%s %q\n", run.GetRunId(), run.GetStatus(), run.GetMessage())
	}
	fmt.Println()

	// --- Step 5: Delete ---
	fmt.Println("=== Step 5: Delete ===")
	if _, err := client.DeleteIntegration(ctx, &integrationsv1.DeleteIntegrationRequest{IntegrationId: id}); err != nil {
		panic(fmt.Errorf("delete failed: %w", err))
	}
	_, getErr := client.GetIntegration(ctx, &integrationsv1.GetIntegrationRequest{IntegrationId: id})
	fmt.Printf("Deleted %s; Get after delete -> %s\n", id, status.Code(getErr))

	fmt.Println("\nDone: warehouse lifecycle (outputs, refresh, health) exercised end to end.")
}

// loadServiceAccountKey returns the BigQuery service-account key JSON. If
// BIGQUERY_SA_KEY points at an existing file it is read from disk, otherwise the
// value is used verbatim; an empty value falls back to a placeholder.
func loadServiceAccountKey() string {
	v := os.Getenv("BIGQUERY_SA_KEY")
	if v == "" {
		return `{"type":"service_account","project_id":"example-project"}`
	}
	if data, err := os.ReadFile(v); err == nil {
		return string(data)
	}
	return v
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
