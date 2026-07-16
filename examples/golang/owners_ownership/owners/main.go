// Owners & ownership maintenance lifecycle — "alert routing as code" — using
// the Coalesce Quality OwnersService (v1):
//
//	create owner (+contacts) -> assign assets (ownership + alert config) ->
//	list -> partial update (with etag) -> delete
//
// An owner is a named responsible party with notification channels (contacts).
// An ownership assigns a set of assets to an owner and configures the alerts
// routed to it. Owner is the resource; ownership is its sub-resource — deleting
// an owner deletes its ownerships.
//
// Both ids are caller-supplied UUIDs, so every write is idempotent. Together
// they replace clicking owners and alert rules together in the UI: define the
// responsible party, its channels, what it owns, and how it should be alerted —
// all as code.
//
// To make the "assign a data product" path concrete, the example first creates
// a throwaway data product to own, and deletes it at the end.
//
// Prerequisites:
//   - SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET with scopes:
//     Read/Edit Owners, Read/Edit Ownership, Read/Edit Data Products.
//   - Optionally API_ENDPOINT (defaults to the EU endpoint developer.synq.io;
//     for the US region use api.us.synq.io).
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	dpv2grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/dataproducts/v2/dataproductsv2grpc"
	ownersv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/owners/v1/ownersv1grpc"
	alertsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/alerts/v1"
	dpv2 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/dataproducts/v2"
	ownersv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/owners/v1"
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
		panic("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set (scopes: Read/Edit Owners, Ownership, Data Products)")
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

	owners := ownersv1grpc.NewOwnersServiceClient(conn)
	products := dpv2grpc.NewDataproductsServiceClient(conn)

	// --- Setup: a data product to own ---
	// Ownerships can select assets by query, or point at a whole data product by
	// id. We create one here so the "assign a data product" path is concrete.
	fmt.Println("=== Setup: create a data product to own ===")
	productID := uuid.NewString()
	if _, err := products.Upsert(ctx, &dpv2.UpsertRequest{
		Id:       productID,
		Title:    proto.String("API Example — Owned Product"),
		Folder:   proto.String("API Examples"),
		Priority: dpv2.Dataproduct_PRIORITY_P1.Enum(),
		Definition: &dpv2.DataproductDefinition{
			Parts: []*dpv2.DataproductDefinition_Part{{
				Id:   uuid.NewString(),
				Part: &dpv2.DataproductDefinition_Part_Query{Query: &dpv2.DataproductQuery{ResolverQl: `with_type("table", filter=with_name("orders"))`}},
			}},
		},
	}); err != nil {
		panic(fmt.Errorf("create product failed: %w", err))
	}
	fmt.Printf("Created product %s\n\n", productID)

	// --- Step 1: Create an owner with notification channels ---
	// Contacts are the channels a fired alert is delivered to. An owner can hold
	// several of different kinds. Replace the placeholders below with real
	// channels/addresses for your workspace.
	fmt.Println("=== Step 1: Create owner (with contacts) ===")
	ownerID := uuid.NewString()
	upOwner, err := owners.UpsertOwner(ctx, &ownersv1.UpsertOwnerRequest{
		Id:    ownerID,
		Title: proto.String("API Example — Data Platform"),
		Contacts: &ownersv1.ContactList{Contacts: []*ownersv1.Contact{
			{ContactMethod: &ownersv1.Contact_Slack{Slack: &ownersv1.SlackChannelContact{Channel: "#data-alerts"}}},
			{ContactMethod: &ownersv1.Contact_Email{Email: &ownersv1.EmailContact{RecipientEmails: []string{"data-team@example.com"}}}},
		}},
	})
	if err != nil {
		panic(fmt.Errorf("create owner failed: %w", err))
	}
	owner := upOwner.GetOwner()
	fmt.Printf("Created owner id=%s\n", owner.GetId())
	// entity_id is server-derived (owner-<uuid>) — the value the alerts API and
	// other surfaces accept as an owner reference.
	fmt.Printf("entity_id=%s contacts=%d etag=%s\n\n", owner.GetEntityId(), len(owner.GetContacts()), owner.GetEtag())

	// --- Step 2: Assign the data product to the owner (ownership #1) ---
	// The ownership's AlertConfig is the "routing as code": which severities
	// fire, whether upstream issues count, and the repeat strategy.
	fmt.Println("=== Step 2: Ownership #1 — own the data product, with alert routing ===")
	ownershipProductID := uuid.NewString()
	up1, err := owners.UpsertOwnership(ctx, &ownersv1.UpsertOwnershipRequest{
		OwnerId: ownerID,
		Id:      ownershipProductID,
		Selection: &ownersv1.OwnershipSelection{
			Selection: &ownersv1.OwnershipSelection_DataproductId{DataproductId: productID},
		},
		Alert: &ownersv1.AlertConfig{
			Severities:     []synqv1.Severity{synqv1.Severity_SEVERITY_ERROR, synqv1.Severity_SEVERITY_FATAL},
			NotifyUpstream: true,
			Ongoing: &alertsv1.OngoingAlertsStrategy{
				Strategy: &alertsv1.OngoingAlertsStrategy_Disabled_{Disabled: &alertsv1.OngoingAlertsStrategy_Disabled{}},
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("create ownership #1 failed: %w", err))
	}
	printOwnership("ownership #1", up1.GetOwnership())

	// --- Step 3: A second ownership selecting assets by query (ownership #2) ---
	// Unlike a data-product definition (a leaf), an ownership query MAY reference
	// data products and domains — routing alerts for everything in a product or
	// domain is a first-class use.
	fmt.Println("=== Step 3: Ownership #2 — select by ResolverQL, daily digest ===")
	ownershipQueryID := uuid.NewString()
	up2, err := owners.UpsertOwnership(ctx, &ownersv1.UpsertOwnershipRequest{
		OwnerId: ownerID,
		Id:      ownershipQueryID,
		Selection: &ownersv1.OwnershipSelection{
			Selection: &ownersv1.OwnershipSelection_Query{
				Query: &ownersv1.OwnershipQuery{
					Name:       "Critical models",
					ResolverQl: `with_type("model", filter=with_name("revenue"))`,
				},
			},
		},
		Alert: &ownersv1.AlertConfig{
			Severities: []synqv1.Severity{synqv1.Severity_SEVERITY_FATAL},
			Ongoing: &alertsv1.OngoingAlertsStrategy{
				Strategy: &alertsv1.OngoingAlertsStrategy_Schedule_{
					Schedule: &alertsv1.OngoingAlertsStrategy_Schedule{Cron: "0 9 * * MON"},
				},
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("create ownership #2 failed: %w", err))
	}
	printOwnership("ownership #2", up2.GetOwnership())

	// --- Step 4: List the owner's ownerships ---
	fmt.Println("=== Step 4: List ownerships ===")
	list, err := owners.ListOwnerships(ctx, &ownersv1.ListOwnershipsRequest{OwnerId: ownerID, Pagination: &synqv1.Pagination{}})
	if err != nil {
		panic(fmt.Errorf("list ownerships failed: %w", err))
	}
	fmt.Printf("Owner has %d ownership(s)\n\n", len(list.GetOwnerships()))

	// --- Step 5: Partial owner update with optimistic concurrency ---
	// Contacts are replace-semantics behind a presence wrapper: omit `contacts`
	// to leave them unchanged, or send the full desired set to replace them.
	// Here we add a Microsoft Teams channel to the existing two contacts.
	fmt.Println("=== Step 5: Replace contacts (add MS Teams) with etag ===")
	staleEtag := owner.GetEtag()
	updOwner, err := owners.UpsertOwner(ctx, &ownersv1.UpsertOwnerRequest{
		Id:   ownerID,
		Etag: proto.String(owner.GetEtag()),
		Contacts: &ownersv1.ContactList{Contacts: []*ownersv1.Contact{
			{ContactMethod: &ownersv1.Contact_Slack{Slack: &ownersv1.SlackChannelContact{Channel: "#data-alerts"}}},
			{ContactMethod: &ownersv1.Contact_Email{Email: &ownersv1.EmailContact{RecipientEmails: []string{"data-team@example.com"}}}},
			{ContactMethod: &ownersv1.Contact_MsTeams{MsTeams: &ownersv1.MsTeamsContact{ChannelId: "19:example-teams-channel-id@thread.tacv2"}}},
		}},
	})
	if err != nil {
		panic(fmt.Errorf("update owner failed: %w", err))
	}
	owner = updOwner.GetOwner()
	fmt.Printf("Updated owner now has %d contacts, new etag=%s\n", len(owner.GetContacts()), owner.GetEtag())

	// Re-using the stale etag is rejected — the concurrency guard.
	_, staleErr := owners.UpsertOwner(ctx, &ownersv1.UpsertOwnerRequest{
		Id:    ownerID,
		Title: proto.String("should be rejected"),
		Etag:  proto.String(staleEtag),
	})
	if status.Code(staleErr) == codes.Aborted {
		fmt.Printf("Stale etag correctly rejected with ABORTED (409)\n\n")
	} else {
		fmt.Printf("Unexpected result for stale etag: %v\n\n", staleErr)
	}

	// --- How this ties to the alerts API ---
	// A fired issue on an owned asset is attributed via owner.entity_id and the
	// ownership id — the same values the alerts API reports and filters on.
	fmt.Println("=== Reference: how alerts point back here ===")
	fmt.Printf("owner entity_id : %s\n", owner.GetEntityId())
	fmt.Printf("ownership ids   : %s, %s\n\n", ownershipProductID, ownershipQueryID)

	// --- Cleanup ---
	// Deleting the owner removes its ownerships too; we still delete one first to
	// show the call. Purge releases the ids.
	fmt.Println("=== Cleanup ===")
	if _, err := owners.DeleteOwnership(ctx, &ownersv1.DeleteOwnershipRequest{Id: ownershipQueryID}); err != nil {
		panic(fmt.Errorf("delete ownership failed: %w", err))
	}
	if _, err := owners.DeleteOwner(ctx, &ownersv1.DeleteOwnerRequest{Id: ownerID, Purge: true}); err != nil {
		panic(fmt.Errorf("delete owner failed: %w", err))
	}
	if _, err := products.Delete(ctx, &dpv2.DeleteRequest{Id: productID, Purge: true}); err != nil {
		panic(fmt.Errorf("delete product failed: %w", err))
	}
	fmt.Printf("Deleted ownerships, owner %s, and product %s\n", ownerID, productID)

	fmt.Println("\nDone: owners & ownership lifecycle exercised end to end.")
}

func printOwnership(label string, o *ownersv1.Ownership) {
	fmt.Printf("%s id=%s etag=%s\n", label, o.GetId(), o.GetEtag())
	switch sel := o.GetSelection().GetSelection().(type) {
	case *ownersv1.OwnershipSelection_DataproductId:
		fmt.Printf("  selection: data product %s\n", sel.DataproductId)
	case *ownersv1.OwnershipSelection_Query:
		fmt.Printf("  selection: query %q -> %s\n", sel.Query.GetName(), sel.Query.GetRenderedResolverQl())
	}
	a := o.GetAlert()
	fmt.Printf("  alert: severities=%v notify_upstream=%v disabled=%v\n\n", a.GetSeverities(), a.GetNotifyUpstream(), a.GetIsDisabled())
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
