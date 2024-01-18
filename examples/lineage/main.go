package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corev1 "github.com/getsynq/api/core/v1"
	schemasv1 "github.com/getsynq/api/schemas/v1"
)

func main() {
	// TODO: add connection mechanism here
	conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("could not connect to server")
	}

	client := schemasv1.NewLineageServiceClient(conn)

	// Add example.
	resp, err := client.GetLineage(context.Background(), &schemasv1.GetLineageRequest{
		StartPoint: &schemasv1.GetLineageStartPoint{
			From: &schemasv1.GetLineageStartPoint_Entities{
				Entities: &schemasv1.EntitiesStartPoint{
					Entities: []*corev1.EntityRef{
						{
							Path: "asset_path1",
							Type: corev1.EntityType_ENTITY_TYPE_DBT_MODEL,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("error fetching lineage -> %v", err)
	}

	fmt.Printf("Lineage -> %+v", resp.Lineage)
}
