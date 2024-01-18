package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corev1 "github.com/getsynq/api/core/v1"
	schemasv1 "github.com/getsynq/api/schemas/v1"
)

const (
	apiUrl       = "https://api.synq.io/"
	clientId     = os.Getenv("SYNQ_CLIENT_ID")
	clientSecret = os.Getenv("SYNQ_CLIENT_SECRET")
)

// Bearer authentication
type bearerAuth struct {
	token string
}

// Convert user credentials to request metadata.
func (b bearerAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + b.token,
	}, nil
}

// Specify whether channel security is required to pass these credentials.
func (b bearerAuth) RequireTransportSecurity() bool {
	return false
}

func main() {
	// Configure credentials.
	token, err := getToken()
	if err != nil {
		panic(err)
	}
	fmt.Println(token)
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(bearerAuth{
			token: token,
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(apiUrl, opts...)
	if err != nil {
		log.Fatalf("could not connect to server -> %+v", err)
	}

	client := schemasv1.NewLineageServiceClient(conn)

	// Fetch lineage.
	resp, err := client.GetLineage(context.Background(), &schemasv1.GetLineageRequest{
		StartPoint: &schemasv1.GetLineageStartPoint{
			From: &schemasv1.GetLineageStartPoint_Entities{
				Entities: &schemasv1.EntitiesStartPoint{
					Entities: []*corev1.EntityRef{
						{
							Path: "asset_path1",
							Type: corev1.EntityType_ENTITY_TYPE_UNSPECIFIED,
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

func getToken() (string, error) {
	type oauth2Token struct {
		AccessToken string `json:"access_token"`
	}

	v := &url.Values{}
	v.Set("client_id", clientID)
	v.Set("client_secret", clientSecret)
	v.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", fmt.Sprintf("%soauth2/token", apiUrl), strings.NewReader(v.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	token := &oauth2Token{}
	if err := json.NewDecoder(res.Body).Decode(token); err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching token. status code %d", res.StatusCode)
	}

	return token.AccessToken, nil
}
