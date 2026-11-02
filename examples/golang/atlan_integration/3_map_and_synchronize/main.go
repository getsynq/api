package main

import (
	extensionsatlanintegrationsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/extensions/atlan/integrations/v1/integrationsv1grpc"
	extensionsatlanworkflowsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/extensions/atlan/workflows/v1/workflowsv1grpc"
	extensionsatlanintegrationsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/extensions/atlan/integrations/v1"
	extensionsatlanworkflowsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/extensions/atlan/workflows/v1"
	platformsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/platforms/v1"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"os"
)

func main() {
	ctx := context.Background()

	// EU is developer.synq.io (the default). Set QUALITY_API_ENDPOINT to
	// api.us.synq.io for US or api.au.synq.io for AU.
	host := getenv("QUALITY_API_ENDPOINT", "developer.synq.io")
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	clientID := os.Getenv("QUALITY_CLIENT_ID")
	clientSecret := os.Getenv("QUALITY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		panic("QUALITY_CLIENT_ID and QUALITY_CLIENT_SECRET must be set")
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

	integrationsApi := extensionsatlanintegrationsv1grpc.NewAtlanIntegrationServiceClient(conn)
	workflowsApi := extensionsatlanworkflowsv1grpc.NewAtlanWorkflowServiceClient(conn)

	// Requires the valid integration that 1_setup_integration creates.

	// Map Atlan connections to Coalesce Quality integrations.
	// Use 2_fetch_atlan_resources to find visible connections.
	{
		_, err := workflowsApi.SetConnectionMappings(ctx, &extensionsatlanworkflowsv1.SetConnectionMappingsRequest{
			Mappings: []*extensionsatlanworkflowsv1.ConnectionMapping{
				{
					AtlanConnectionQualifiedName: "default/dbt/1",
					SynqDataPlatformIdentifier: &platformsv1.DataPlatformIdentifier{
						Id: &platformsv1.DataPlatformIdentifier_DbtCloud{
							DbtCloud: &platformsv1.DbtCloudIdentifier{
								ApiEndpoint: "cloud.getdbt.com",
								AccountId:   "1234",
								ProjectId:   "5678",
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

	// Activate integration once you are ready to synchronize.
	// This runs a synchronization on a regular schedule.
	{
		_, err := integrationsApi.Activate(ctx, &extensionsatlanintegrationsv1.ActivateRequest{
			Activate: true,
		})
		if err != nil {
			panic(err)
		}
	}

	// You can also start a synchronization manually.
	// It creates the data products and domains visible in Atlan in Coalesce Quality.
	{
		resp, err := workflowsApi.Synchronize(ctx, &extensionsatlanworkflowsv1.SynchronizeRequest{})
		if err != nil {
			panic(err)
		}
		b, _ := json.Marshal(resp.WorkflowRun)
		fmt.Printf("Synchronization result -> %s\n\n", string(b))
		if resp.HasErrors {
			panic("synchronization has errors")
		}
	}

	// Fetch mapped products and domains.
	{
		resp, err := workflowsApi.GetDomainMappings(ctx, &extensionsatlanworkflowsv1.GetDomainMappingsRequest{})
		if err != nil {
			panic(err)
		}
		b, _ := json.Marshal(resp.Mappings)
		fmt.Printf("Mapped domains -> %s\n\n", string(b))
	}
	{
		resp, err := workflowsApi.GetProductMappings(ctx, &extensionsatlanworkflowsv1.GetProductMappingsRequest{})
		if err != nil {
			panic(err)
		}
		b, _ := json.Marshal(resp.Mappings)
		fmt.Printf("Mapped products -> %s\n\n", string(b))
	}

	// Fetch synchronization runs history.
	{
		resp, err := workflowsApi.FetchRuns(ctx, &extensionsatlanworkflowsv1.FetchRunsRequest{
			From:  0,
			Limit: 10,
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Found %d runs.\n", len(resp.WorkflowRuns))
		for _, run := range resp.WorkflowRuns {
			b, _ := json.Marshal(run)
			fmt.Printf(" -> [%+v] (%+v) %s\n", run.StartedAt, run.Status, string(b))
		}
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
