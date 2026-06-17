// Demonstrates the full lifecycle of an integration using the Coalesce Quality
// IntegrationsService, with a dbt Cloud connection as the example type:
//
//	create -> get -> list -> update (with etag) -> disable -> enable -> delete
//
// dbt Cloud is a "managed" type: Coalesce Quality stores the connection and
// syncs it in the background. The example deliberately works even with an
// invalid / disabled dbt Cloud token: every management call (create, update,
// enable/disable, delete) still succeeds, because those operate on the stored
// configuration — proving the API works end to end. Only the background sync
// would fail, which you can observe later via GetIntegrationHealth.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	integrationsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/integrations/v1/integrationsv1grpc"
	integrationsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/integrations/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

	// dbt Cloud connection settings. Defaults are illustrative — replace them
	// with your own (the token can be invalid/disabled; the management calls
	// below still succeed).
	dbtAccountID := getenv("DBT_CLOUD_ACCOUNT_ID", "12345")
	dbtProjectID := getenv("DBT_CLOUD_PROJECT_ID", "example-dbt-project")
	dbtToken := getenv("DBT_CLOUD_TOKEN", "dbtc_disabled-demo-token")
	dbtAPIEndpoint := getenv("DBT_CLOUD_API_ENDPOINT", "cloud.getdbt.com")
	// Tracked dbt Cloud job ids (comma-separated). Empty tracks every job the
	// token can see.
	dbtJobIDs := splitCSV(getenv("DBT_CLOUD_JOB_IDS", "100,200"))

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
	fmt.Println("=== Step 1: Create dbt Cloud integration ===")
	created, err := client.CreateIntegration(ctx, &integrationsv1.CreateIntegrationRequest{
		Title: "Example dbt Cloud",
		Config: &integrationsv1.IntegrationConfig{
			Config: &integrationsv1.IntegrationConfig_DbtCloud{
				DbtCloud: &integrationsv1.DbtCloudConf{
					AccountId:   proto.String(dbtAccountID),
					ProjectId:   proto.String(dbtProjectID),
					Token:       proto.String(dbtToken),
					ApiEndpoint: proto.String(dbtAPIEndpoint),
					JobIds:      dbtJobIDs,
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
	fmt.Printf("Tracked job ids: %v\n", integration.GetConfig().GetDbtCloud().GetJobIds())
	// The token is write-only: it is masked (empty) on every read.
	fmt.Printf("Token in response (masked): %q\n\n", integration.GetConfig().GetDbtCloud().GetToken())

	// --- Step 2: Get ---
	fmt.Println("=== Step 2: Get ===")
	got, err := client.GetIntegration(ctx, &integrationsv1.GetIntegrationRequest{IntegrationId: id})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Title=%q disabled=%v project_id=%q\n\n",
		got.GetIntegration().GetTitle(),
		got.GetIntegration().GetDisabled(),
		got.GetIntegration().GetConfig().GetDbtCloud().GetProjectId())

	// --- Step 3: List ---
	fmt.Println("=== Step 3: List all integrations ===")
	list, err := client.ListIntegrations(ctx, &integrationsv1.ListIntegrationsRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Workspace has %d integration(s)\n\n", len(list.GetIntegrations()))

	// --- Step 4: Patch the tracked job ids (with optimistic concurrency) ---
	// The config is replaced wholesale, so job_ids is replace-semantics: send the
	// FULL desired set, not a delta. Here we drop one job and add another. We omit
	// the token to keep the stored one, and pass the etag we last read so we only
	// update the version we saw (a stale etag is rejected with ABORTED).
	patchedJobIDs := []string{"100", "300"} // was {"100","200"}: keep 100, drop 200, add 300
	fmt.Println("=== Step 4: Patch tracked job ids ===")
	updated, err := client.UpdateIntegration(ctx, &integrationsv1.UpdateIntegrationRequest{
		IntegrationId: id,
		Title:         proto.String("Example dbt Cloud (updated)"),
		Etag:          proto.String(integration.GetEtag()),
		Config: &integrationsv1.IntegrationConfig{
			Config: &integrationsv1.IntegrationConfig_DbtCloud{
				DbtCloud: &integrationsv1.DbtCloudConf{
					AccountId:   proto.String(dbtAccountID),
					ProjectId:   proto.String(dbtProjectID),
					ApiEndpoint: proto.String(dbtAPIEndpoint),
					JobIds:      patchedJobIDs,
					// Token omitted -> the previously stored token is preserved.
				},
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("update failed: %w", err))
	}
	newEtag := updated.GetIntegration().GetEtag()
	fmt.Printf("Patched job ids: %v -> %v\n",
		dbtJobIDs, updated.GetIntegration().GetConfig().GetDbtCloud().GetJobIds())
	fmt.Printf("new etag=%s\n", newEtag)

	// Re-using the now-stale etag is rejected — this is the concurrency guard.
	_, staleErr := client.UpdateIntegration(ctx, &integrationsv1.UpdateIntegrationRequest{
		IntegrationId: id,
		Etag:          proto.String(integration.GetEtag()), // stale: we are at newEtag now
		Config:        updated.GetIntegration().GetConfig(),
	})
	if status.Code(staleErr) == codes.Aborted {
		fmt.Printf("Stale etag correctly rejected with ABORTED (409)\n\n")
	} else {
		fmt.Printf("Unexpected result for stale etag: %v\n\n", staleErr)
	}

	// --- Step 5: Disable then Enable ---
	fmt.Println("=== Step 5: Disable / Enable ===")
	dis, err := client.DisableIntegration(ctx, &integrationsv1.DisableIntegrationRequest{IntegrationId: id})
	if err != nil {
		panic(err)
	}
	fmt.Printf("disabled=%v can_enable=%v\n",
		dis.GetIntegration().GetDisabled(),
		dis.GetIntegration().GetCapabilities().GetCanEnable())
	en, err := client.EnableIntegration(ctx, &integrationsv1.EnableIntegrationRequest{IntegrationId: id})
	if err != nil {
		panic(err)
	}
	fmt.Printf("disabled=%v can_disable=%v\n\n",
		en.GetIntegration().GetDisabled(),
		en.GetIntegration().GetCapabilities().GetCanDisable())

	// --- Step 6: Health ---
	// dbt Cloud syncs in the background. Right after create there are usually no
	// runs yet (status UNSPECIFIED). If the token is invalid/disabled, a later
	// run shows ERROR here — that is the end-to-end proof the connection is live.
	fmt.Println("=== Step 6: Health ===")
	health, err := client.GetIntegrationHealth(ctx, &integrationsv1.GetIntegrationHealthRequest{IntegrationId: id})
	if err != nil {
		panic(err)
	}
	fmt.Printf("status=%s healthy=%v message=%q recent runs=%d\n\n",
		health.GetHealth().GetStatus(),
		health.GetHealth().GetHealthy(),
		health.GetHealth().GetMessage(),
		len(health.GetRuns()))

	// --- Step 7: Delete (with etag) ---
	fmt.Println("=== Step 7: Delete ===")
	currentEtag := en.GetIntegration().GetEtag()
	if _, err := client.DeleteIntegration(ctx, &integrationsv1.DeleteIntegrationRequest{
		IntegrationId: id,
		Etag:          proto.String(currentEtag),
	}); err != nil {
		panic(fmt.Errorf("delete failed: %w", err))
	}
	fmt.Printf("Deleted %s\n", id)

	// Get now returns NOT_FOUND.
	_, getErr := client.GetIntegration(ctx, &integrationsv1.GetIntegrationRequest{IntegrationId: id})
	fmt.Printf("Get after delete -> %s\n", status.Code(getErr))

	fmt.Println("\nDone: full management lifecycle exercised end to end.")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// splitCSV parses a comma-separated list, trimming spaces and dropping empties.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
