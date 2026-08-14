// Reads and replays the notifications Coalesce Quality sent to one of your
// integrations — a webhook, PagerDuty or Opsgenie — using
// synq.deliveries.v1.DeliveriesService.
//
// It answers, in order: what has this integration been sent, which of those
// failed, what exactly did we send and what came back, and does it work now.
//
// Reading needs a token with SCOPE_DELIVERY_READ; the redelivery at the end needs
// SCOPE_DELIVERY_EDIT.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	deliveriesv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/deliveries/v1/deliveriesv1grpc"
	deliveriesv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/deliveries/v1"
	synqv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The integration whose deliveries to read. Leave it empty to read the whole
// workspace's feed instead — every filter below works either way.
const integrationID = "d577b364-a867-11ed-b4b2-fe8020e7ba25"

func main() {
	ctx := context.Background()

	host := "developer.synq.io"
	port := "443"
	apiUrl := fmt.Sprintf("%s:%s", host, port)

	clientID := "foo"
	clientSecret := "bar"
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

	client := deliveriesv1grpc.NewDeliveriesServiceClient(conn)
	since := timestamppb.New(time.Now().Add(-24 * time.Hour))

	summarise(ctx, client, since)
	listRecent(ctx, client, since)

	failed := firstFailed(ctx, client, since)
	if failed == nil {
		fmt.Println("\nNo failed delivery in the last 24 hours.")
		return
	}

	showAttempts(ctx, client, failed)
	redeliver(ctx, client, failed)
}

// summarise counts the window by outcome before paging through anything. This is
// what to alert your own monitoring on: a delivery failure rate you can read
// without walking the feed.
func summarise(ctx context.Context, client deliveriesv1grpc.DeliveriesServiceClient, since *timestamppb.Timestamp) {
	resp, err := client.SummariseDeliveries(ctx, &deliveriesv1.SummariseDeliveriesRequest{
		IntegrationId: proto(integrationID),
		Since:         since,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d deliveries since %s\n", resp.GetTotal(), resp.GetSince().AsTime().Format(time.RFC3339))
	for _, count := range resp.GetByOutcome() {
		fmt.Printf("  %-20s %d\n", count.GetOutcome(), count.GetCount())
	}
	// Only over the deliveries that sent nothing, and the reason is what says
	// whether that is a misconfiguration or expected.
	for _, count := range resp.GetBySkipReason() {
		fmt.Printf("  %-20s %d\n", count.GetSkipReason(), count.GetCount())
	}
	for _, count := range resp.GetByStatusClass() {
		fmt.Printf("  %-20s %d\n", count.GetStatusClass(), count.GetCount())
	}
}

// listRecent pages the feed, newest event first. Pass the previous page's
// PageInfo.last_id back as the cursor; an empty last_id means there is no more.
func listRecent(ctx context.Context, client deliveriesv1grpc.DeliveriesServiceClient, since *timestamppb.Timestamp) {
	fmt.Println("\nRecent deliveries:")

	var cursor string
	for page := 0; page < 5; page++ {
		resp, err := client.ListDeliveries(ctx, &deliveriesv1.ListDeliveriesRequest{
			IntegrationId: proto(integrationID),
			Since:         since,
			Pagination:    &synqv1.Pagination{Cursor: proto(cursor), PageSize: proto(int32(50))},
			// Which alert configuration selected this integration for the event —
			// the answer to "why did I get this". Off by default because it costs a
			// second read per delivery.
			IncludeRouting: true,
		})
		if err != nil {
			panic(err)
		}

		for _, delivery := range resp.GetDeliveries() {
			describe(delivery)
		}

		cursor = resp.GetPageInfo().GetLastId()
		if cursor == "" {
			return
		}
	}
}

// firstFailed narrows the same feed to one outcome. A delivery is FAILED when the
// destination rejected it outright or was still failing when the retries ran out —
// distinct from PENDING, where an attempt is still to come, which is why the
// outcome is filtered on rather than the status code.
func firstFailed(
	ctx context.Context,
	client deliveriesv1grpc.DeliveriesServiceClient,
	since *timestamppb.Timestamp,
) *deliveriesv1.Delivery {
	resp, err := client.ListDeliveries(ctx, &deliveriesv1.ListDeliveriesRequest{
		IntegrationId: proto(integrationID),
		Since:         since,
		Outcomes:      []deliveriesv1.Outcome{deliveriesv1.Outcome_OUTCOME_FAILED},
		Pagination:    &synqv1.Pagination{PageSize: proto(int32(1))},
	})
	if err != nil {
		panic(err)
	}
	if len(resp.GetDeliveries()) == 0 {
		return nil
	}
	return resp.GetDeliveries()[0]
}

// showAttempts prints what was sent and what came back, for every try.
func showAttempts(ctx context.Context, client deliveriesv1grpc.DeliveriesServiceClient, delivery *deliveriesv1.Delivery) {
	fmt.Printf("\nAttempts for %s:\n", delivery.GetId())

	resp, err := client.ListAttempts(ctx, &deliveriesv1.ListAttemptsRequest{DeliveryId: delivery.GetId()})
	if err != nil {
		panic(err)
	}

	for _, attempt := range resp.GetAttempts() {
		fmt.Printf("  attempt %d at %s took %s\n",
			attempt.GetAttempt(),
			attempt.GetAttemptedAt().AsTime().Format(time.RFC3339),
			attempt.GetDuration().AsDuration())

		// No response arrived at all — a connection failure, a TLS error, a
		// timeout. There is no status to read in that case.
		if attempt.GetError() != "" {
			fmt.Printf("    no response: %s\n", attempt.GetError())
			continue
		}

		http := attempt.GetHttp()
		fmt.Printf("    %s %s -> %d\n", http.GetMethod(), http.GetUrl(), http.GetStatusCode())
		printHeaders("    request ", http.GetRequestHeaders())
		printHeaders("    response", http.GetResponseHeaders())

		if http.GetResponseTruncated() {
			fmt.Printf("    response body (first bytes of %d): %s\n", http.GetResponseBytes(), http.GetResponseBody())
		} else {
			fmt.Printf("    response body: %s\n", http.GetResponseBody())
		}
	}
}

// printHeaders shows every header that was present, including the ones whose value
// is withheld. A withheld value still reports a fingerprint — stable for the same
// value — so two deliveries can be compared to tell whether a credential changed,
// which is the question without disclosing the answer. Signature headers are not
// withheld: they are a digest of the body, not the secret, and reproducing a
// failed verification needs the exact bytes.
func printHeaders(prefix string, headers []*deliveriesv1.Header) {
	for _, header := range headers {
		if header.GetWithheld() {
			fmt.Printf("%s %s: <withheld, fingerprint %s, secret version %s>\n",
				prefix, header.GetName(), header.GetFingerprint(), header.GetSecretVersion())
			continue
		}
		fmt.Printf("%s %s: %s\n", prefix, header.GetName(), header.GetValue())
	}
}

// redeliver sends a stored event again, which is how to confirm a fix — or that a
// rotated signing secret is accepted, since the replay is signed with the current
// one rather than the secret the original used.
//
// The replay is a new delivery with its own id, linked to the original through
// redelivery_of, and it appears in the feed like any other. The send is
// asynchronous, so poll for the outcome.
func redeliver(ctx context.Context, client deliveriesv1grpc.DeliveriesServiceClient, delivery *deliveriesv1.Delivery) {
	resp, err := client.Redeliver(ctx, &deliveriesv1.RedeliverRequest{DeliveryId: delivery.GetId()})
	if err != nil {
		// Redeliver is rate-limited per integration, so a replay loop cannot hammer
		// an endpoint. Over the limit it fails with RESOURCE_EXHAUSTED rather than
		// queueing: wait and try again.
		panic(err)
	}

	replayID := resp.GetDeliveryId()
	fmt.Printf("\nReplayed as %s\n", replayID)

	for attempt := 0; attempt < 10; attempt++ {
		time.Sleep(2 * time.Second)

		got, err := client.BatchGetDeliveries(ctx, &deliveriesv1.BatchGetDeliveriesRequest{Ids: []string{replayID}})
		if err != nil {
			panic(err)
		}

		// An id that does not exist, or has passed out of the 90-day retention
		// window, is absent from the response rather than an error.
		replay, found := got.GetDeliveries()[replayID]
		if !found {
			continue
		}
		if replay.GetOutcome() == deliveriesv1.Outcome_OUTCOME_PENDING {
			fmt.Printf("  pending, next attempt at %s\n", replay.GetNextAttemptAt().AsTime().Format(time.RFC3339))
			continue
		}

		describe(replay)
		return
	}

	fmt.Println("  still pending; read it later with BatchGetDeliveries")
}

func describe(delivery *deliveriesv1.Delivery) {
	fmt.Printf("  %s  %-18s %-28s %s\n",
		delivery.GetCreatedAt().AsTime().Format(time.RFC3339),
		delivery.GetOutcome(),
		delivery.GetEventType(),
		delivery.GetSubject().GetTitle())

	// Set exactly when nothing was sent, and each value names something specific
	// to change — an event type to subscribe to, an integration to enable.
	if delivery.GetSkipReason() != deliveriesv1.SkipReason_SKIP_REASON_UNSPECIFIED {
		fmt.Printf("      nothing sent: %s\n", delivery.GetSkipReason())
	} else {
		fmt.Printf("      %d attempt(s), last status %d, took %s\n",
			delivery.GetAttemptCount(), delivery.GetStatusCode(), delivery.GetDuration().AsDuration())
	}

	// Shared by every integration the same event reached, and the same value the
	// receiver got in the webhook payload — so filter ListDeliveries on it to see
	// where else one event went.
	fmt.Printf("      event %s\n", delivery.GetEventId())

	if replayed := delivery.GetRedeliveryOf(); replayed != "" {
		fmt.Printf("      replay of %s\n", replayed)
	}
	for _, matched := range delivery.GetMatchedAlertConfigs() {
		fmt.Printf("      matched alert config %s (%s)\n", matched.GetName(), matched.GetAlertConfigId())
	}
	if key := delivery.GetCorrelationKey(); key != "" {
		// A PagerDuty dedup_key or an Opsgenie alias: what the incident is called on
		// the destination's side.
		fmt.Printf("      correlation key %s\n", key)
	}
}

func proto[T any](value T) *T {
	return &value
}
