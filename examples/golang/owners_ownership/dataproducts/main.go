// Data-products maintenance lifecycle using the Coalesce Quality
// DataproductsService (v2):
//
//	upsert -> batch-get -> set-definition -> add/remove parts -> list-members ->
//	list -> partial update (with etag) -> delete
//
// A data product is a named, owned grouping of assets with a membership
// definition. The id is a caller-supplied UUID, so every write is idempotent —
// re-running this example converges instead of duplicating.
//
// The membership definition is authored here in ResolverQL, the compact text
// query language (see lib/resolverql). The server compiles and stores it
// canonically and echoes it back on reads as `rendered_resolver_ql`.
//
// Prerequisites:
//   - SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET with scopes:
//     Read/Edit Data Products.
//   - Optionally API_ENDPOINT (defaults to the EU endpoint developer.synq.io;
//     for the US region use api.us.synq.io).
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	dpv2grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/dataproducts/v2/dataproductsv2grpc"
	dpv2 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/dataproducts/v2"
	synqv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/v1"
	"github.com/google/uuid"
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

	clientID := os.Getenv("SYNQ_CLIENT_ID")
	clientSecret := os.Getenv("SYNQ_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		panic("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set (scopes: Read/Edit Data Products)")
	}

	// --- Connect (OAuth2 client-credentials) ---
	cc := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     fmt.Sprintf("https://%s/oauth2/token", host),
	}
	conn, err := grpc.DialContext(ctx, host+":443",
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: cc.TokenSource(ctx)}),
		grpc.WithAuthority(host),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Printf("Connected to %s\n\n", host)

	client := dpv2grpc.NewDataproductsServiceClient(conn)

	// The caller owns the id. Using a fresh UUID here creates a new product;
	// re-using a known id would update that product in place.
	id := uuid.NewString()

	// --- Step 1: Create ---
	// Only `title` is required on create. `id` is caller-supplied. `folder` and
	// `priority` are optional metadata.
	fmt.Println("=== Step 1: Create data product ===")
	created, err := client.Upsert(ctx, &dpv2.UpsertRequest{
		Id:          id,
		Title:       proto.String("API Example — Orders"),
		Description: proto.String("Created by the owners_ownership Go example."),
		Folder:      proto.String("API Examples"),
		Priority:    dpv2.Dataproduct_PRIORITY_P1.Enum(),
	})
	if err != nil {
		panic(fmt.Errorf("create failed: %w", err))
	}
	dp := created.GetDataproduct()
	fmt.Printf("Created id=%s\n", dp.GetId())
	// entity_id is server-derived (dataproduct-<uuid>). This is the value other
	// APIs (lineage, ownership, alerts) accept as a reference to this product.
	fmt.Printf("entity_id=%s (use this to reference the product elsewhere)\n", dp.GetEntityId())
	fmt.Printf("etag=%s priority=%s\n\n", dp.GetEtag(), dp.GetPriority())

	// --- Step 2: Replace the whole definition (SetDefinition) ---
	// A definition is a list of parts OR'd together. Here a single ResolverQL
	// query part. Replace the placeholder query with one that matches assets in
	// your workspace (copy paths/ids from the Synq UI).
	fmt.Println("=== Step 2: Set definition (ResolverQL) ===")
	setResp, err := client.SetDefinition(ctx, &dpv2.SetDefinitionRequest{
		Id:   id,
		Etag: proto.String(dp.GetEtag()),
		Definition: &dpv2.DataproductDefinition{
			Parts: []*dpv2.DataproductDefinition_Part{
				{
					Id: uuid.NewString(),
					Part: &dpv2.DataproductDefinition_Part_Query{
						Query: &dpv2.DataproductQuery{
							ResolverQl: `with_type("table", filter=with_name("orders"))`,
						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("set definition failed: %w", err))
	}
	dp = setResp.GetDataproduct()
	printDefinition(dp)

	// --- Step 3: Add a second part (UpsertDefinitionPart) ---
	// Parts have caller-supplied ids so an upsert is idempotent: the same part id
	// replaces that part; a new id adds one. Definition writes bump the etag, so
	// pass the latest one.
	fmt.Println("=== Step 3: Add a second part ===")
	partID := uuid.NewString()
	addResp, err := client.UpsertDefinitionPart(ctx, &dpv2.UpsertDefinitionPartRequest{
		Id:   id,
		Etag: proto.String(dp.GetEtag()),
		Part: &dpv2.DataproductDefinition_Part{
			Id: partID,
			Part: &dpv2.DataproductDefinition_Part_Query{
				Query: &dpv2.DataproductQuery{
					ResolverQl: `with_type("view", filter=with_name("orders"))`,
				},
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("upsert part failed: %w", err))
	}
	dp = addResp.GetDataproduct()
	printDefinition(dp)

	// --- Step 4: Remove that part (RemoveDefinitionPart) ---
	fmt.Println("=== Step 4: Remove the part ===")
	rmResp, err := client.RemoveDefinitionPart(ctx, &dpv2.RemoveDefinitionPartRequest{
		Id:     id,
		PartId: partID,
		Etag:   proto.String(dp.GetEtag()),
	})
	if err != nil {
		panic(fmt.Errorf("remove part failed: %w", err))
	}
	dp = rmResp.GetDataproduct()
	printDefinition(dp)

	// --- Step 5: List resolved members ---
	// The definition resolves to concrete assets, returned as opaque entity ids.
	// Empty is normal if the placeholder query matches nothing in your workspace.
	fmt.Println("=== Step 5: List members ===")
	members, err := client.ListMembers(ctx, &dpv2.ListMembersRequest{Id: id, Pagination: &synqv1.Pagination{}})
	if err != nil {
		panic(fmt.Errorf("list members failed: %w", err))
	}
	fmt.Printf("Resolved %d member(s)\n", len(members.GetEntityIds()))
	for i, e := range members.GetEntityIds() {
		if i >= 5 {
			fmt.Println("  …")
			break
		}
		fmt.Printf("  - %s\n", e)
	}
	fmt.Println()

	// --- Step 6: BatchGet (exclude the definition for a lighter read) ---
	fmt.Println("=== Step 6: BatchGet ===")
	bg, err := client.BatchGet(ctx, &dpv2.BatchGetRequest{Ids: []string{id}, ExcludeDefinition: true})
	if err != nil {
		panic(fmt.Errorf("batch get failed: %w", err))
	}
	got := bg.GetDataproducts()[id]
	fmt.Printf("title=%q folder=%q priority=%s (definition excluded: %d parts)\n\n",
		got.GetTitle(), got.GetFolder(), got.GetPriority(), len(got.GetDefinition().GetParts()))

	// --- Step 7: Partial update with optimistic concurrency ---
	// Upsert only the fields you set; omitted fields are left unchanged. Pass the
	// etag you last read so a concurrent edit is not silently overwritten.
	fmt.Println("=== Step 7: Partial update (title only) with etag ===")
	staleEtag := dp.GetEtag()
	upd, err := client.Upsert(ctx, &dpv2.UpsertRequest{
		Id:    id,
		Title: proto.String("API Example — Orders (P2)"),
		// Only title + priority set; description/folder/definition untouched.
		Priority: dpv2.Dataproduct_PRIORITY_P2.Enum(),
		Etag:     proto.String(dp.GetEtag()),
	})
	if err != nil {
		panic(fmt.Errorf("update failed: %w", err))
	}
	dp = upd.GetDataproduct()
	fmt.Printf("Updated title=%q priority=%s new etag=%s\n", dp.GetTitle(), dp.GetPriority(), dp.GetEtag())

	// Re-using the now-stale etag is rejected — the concurrency guard.
	_, staleErr := client.Upsert(ctx, &dpv2.UpsertRequest{
		Id:    id,
		Title: proto.String("should be rejected"),
		Etag:  proto.String(staleEtag),
	})
	if status.Code(staleErr) == codes.Aborted {
		fmt.Printf("Stale etag correctly rejected with ABORTED (409)\n\n")
	} else {
		fmt.Printf("Unexpected result for stale etag: %v\n\n", staleErr)
	}

	// --- Step 8: Delete (purge to release the id) ---
	fmt.Println("=== Step 8: Delete ===")
	if _, err := client.Delete(ctx, &dpv2.DeleteRequest{Id: id, Purge: true, Etag: proto.String(dp.GetEtag())}); err != nil {
		panic(fmt.Errorf("delete failed: %w", err))
	}
	after, _ := client.BatchGet(ctx, &dpv2.BatchGetRequest{Ids: []string{id}})
	fmt.Printf("Deleted %s; BatchGet after delete returned %d product(s)\n", id, len(after.GetDataproducts()))

	fmt.Println("\nDone: data-product maintenance lifecycle exercised end to end.")
}

func printDefinition(dp *dpv2.Dataproduct) {
	parts := dp.GetDefinition().GetParts()
	fmt.Printf("definition has %d part(s), etag=%s\n", len(parts), dp.GetEtag())
	for _, p := range parts {
		switch {
		case p.GetQuery() != nil:
			fmt.Printf("  - part %s: %s\n", p.GetId(), p.GetQuery().GetRenderedResolverQl())
		case p.GetEntityId() != "":
			fmt.Printf("  - part %s: entity_id=%s\n", p.GetId(), p.GetEntityId())
		}
	}
	fmt.Println()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
