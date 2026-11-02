package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	deploymentrulesv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/monitors/automated_monitors/v1/automated_monitorsv1grpc"
	entitiesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/entities/v1"
	deploymentrulesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/monitors/automated_monitors/v1"
	queriesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/queries/v1"
	"github.com/google/uuid"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
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

	deploymentrulesapi := deploymentrulesv1grpc.NewDeploymentRulesServiceClient(conn)

	// List deployment rules.
	{
		list, err := deploymentrulesapi.ListDeploymentRules(ctx, &deploymentrulesv1.ListDeploymentRulesRequest{})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Deployment rules: %+v\n", list)
		fmt.Println("--------------------------------")
	}

	// Update deployment rule title.
	{
		_, err = deploymentrulesapi.BatchUpdateDeploymentRuleTitle(ctx, &deploymentrulesv1.BatchUpdateDeploymentRuleTitleRequest{
			DeploymentRules: []*deploymentrulesv1.UpdateDeploymentRuleTitleRequest{
				{
					Id:    "dep-rule-uuid-1",
					Title: "new title 1",
				},
			},
		})
		if err != nil {
			panic(err)
		}
	}

	// Get deployment overview.
	// Use an existing deployment rule id with a new or updated config to see the changes in the overview.
	// Use a new deployment rule id to see the overview of a new deployment rule.
	{
		fmt.Println("Getting deployment overview...")
		depRule := &deploymentrulesv1.MonitorsDeploymentRule{
			Id:    uuid.New().String(),
			Title: "test in folder",
			Config: &deploymentrulesv1.MonitorsDeploymentRule_QueryConfig{
				QueryConfig: &deploymentrulesv1.QueryConfig{
					Query: &deploymentrulesv1.EntitySelectionQuery{
						Operand: queriesv1.QueryOperand_QUERY_OPERAND_AND,
						Parts: []*deploymentrulesv1.EntitySelectionQuery_QueryPart{
							{
								Part: &deploymentrulesv1.EntitySelectionQuery_QueryPart_InFolder{
									InFolder: &queriesv1.InFolder{
										Path: []string{"folder-ch-prod::anomalies"},
									},
								},
							},
						},
					},
					Severity:    deploymentrulesv1.Severity_SEVERITY_ERROR,
					Sensitivity: deploymentrulesv1.Sensitivity_SENSITIVITY_BALANCED,
					MetricIds:   []deploymentrulesv1.MetricId{deploymentrulesv1.MetricId_METRIC_ID_ROW_COUNT},
				},
			},
		}
		overview, err := deploymentrulesapi.GetDeployOverview(ctx, &deploymentrulesv1.GetDeployOverviewRequest{DeploymentRule: depRule})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Deployment overview (in folder): %+v\n", overview)
		fmt.Println("--------------------------------")

		depRule = &deploymentrulesv1.MonitorsDeploymentRule{
			Id:    uuid.New().String(),
			Title: "test upstream",
			Config: &deploymentrulesv1.MonitorsDeploymentRule_QueryConfig{
				QueryConfig: &deploymentrulesv1.QueryConfig{
					Query: &deploymentrulesv1.EntitySelectionQuery{
						Operand: queriesv1.QueryOperand_QUERY_OPERAND_AND,
						Parts: []*deploymentrulesv1.EntitySelectionQuery_QueryPart{
							{
								Part: &deploymentrulesv1.EntitySelectionQuery_QueryPart_Query{
									Query: &deploymentrulesv1.EntitySelectionQuery{
										Operand: queriesv1.QueryOperand_QUERY_OPERAND_UPSTREAM,
										Parts: []*deploymentrulesv1.EntitySelectionQuery_QueryPart{
											{
												Part: &deploymentrulesv1.EntitySelectionQuery_QueryPart_WithType{
													WithType: &queriesv1.WithType{
														Types: []*queriesv1.WithType_Type{
															{
																EntityType: &queriesv1.WithType_Type_Default{
																	Default: entitiesv1.EntityType_ENTITY_TYPE_DBT_MODEL,
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Severity:    deploymentrulesv1.Severity_SEVERITY_ERROR,
					Sensitivity: deploymentrulesv1.Sensitivity_SENSITIVITY_BALANCED,
					MetricIds: []deploymentrulesv1.MetricId{
						deploymentrulesv1.MetricId_METRIC_ID_ROW_COUNT,
						deploymentrulesv1.MetricId_METRIC_ID_DELAY,
					},
				},
			},
		}
		overview, err = deploymentrulesapi.GetDeployOverview(ctx, &deploymentrulesv1.GetDeployOverviewRequest{DeploymentRule: depRule})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Deployment overview (upstream): %+v\n", overview)
		fmt.Println("--------------------------------")
	}

	// Deploy the deployment rule.
	{
		fmt.Println("Deploying deployment rule...")
		uuid := uuid.New().String()

		depRule := &deploymentrulesv1.MonitorsDeploymentRule{
			Id:    uuid,
			Title: "public api test",
			Config: &deploymentrulesv1.MonitorsDeploymentRule_QueryConfig{
				QueryConfig: &deploymentrulesv1.QueryConfig{
					Query: &deploymentrulesv1.EntitySelectionQuery{
						Operand: queriesv1.QueryOperand_QUERY_OPERAND_AND,
						Parts: []*deploymentrulesv1.EntitySelectionQuery_QueryPart{
							{
								Part: &deploymentrulesv1.EntitySelectionQuery_QueryPart_WithNameSearch{
									WithNameSearch: &queriesv1.WithNameSearch{
										SearchQuery: "anomalies_stats_daily",
									},
								},
							},
						},
					},
					Severity:    deploymentrulesv1.Severity_SEVERITY_ERROR,
					Sensitivity: deploymentrulesv1.Sensitivity_SENSITIVITY_BALANCED,
					MetricIds: []deploymentrulesv1.MetricId{
						deploymentrulesv1.MetricId_METRIC_ID_ROW_COUNT,
						deploymentrulesv1.MetricId_METRIC_ID_DELAY,
					},
				},
			},
		}
		_, err := deploymentrulesapi.DeployDeploymentRule(ctx, &deploymentrulesv1.DeployDeploymentRuleRequest{DeploymentRule: depRule})
		if err != nil {
			panic(err)
		}

		fmt.Println("Deployment rule deployed with id: ", uuid)
		fmt.Println("--------------------------------")
	}

	// Get the overview for deleting the deployment rule.
	{
		fmt.Println("Getting the overview for deleting the deployment rule...")
		overview, err := deploymentrulesapi.GetDeployDeleteOverview(
			ctx,
			&deploymentrulesv1.GetDeployDeleteOverviewRequest{Id: "dep-rule-uuid-1"},
		)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Delete deployment overview: %+v\n", overview)
		fmt.Println("--------------------------------")
	}

	// Delete the deployment rule.
	{
		fmt.Println("Deleting deployment rule...")
		_, err := deploymentrulesapi.DeleteDeploymentRule(
			ctx,
			&deploymentrulesv1.DeleteDeploymentRuleRequest{Id: "dep-rule-uuid-1"},
		)
		if err != nil {
			panic(err)
		}

		fmt.Println("Deployment rule deleted")
		fmt.Println("--------------------------------")
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
