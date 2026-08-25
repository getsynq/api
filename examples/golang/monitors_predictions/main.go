package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	predictionsv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/monitors/predictions/v1/predictionsv1grpc"
	predictionsv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/monitors/predictions/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Fetches monitor predictions via GetMonitorPredictions.
//
// Flags win over env vars of the same name.
//
// Required:
//
//	-client-id / SYNQ_CLIENT_ID, -client-secret / SYNQ_CLIENT_SECRET  — SCOPE_MONITORS_READ
//	-synq-path / SYNQ_PATH  — monitor synq path, e.g. monitor::ch-prod::default::runs::volume
//
// Optional:
//
//	-endpoint / API_ENDPOINT     — developer.synq.io (EU, default), api.us.synq.io (US), api.au.synq.io (AU)
//	-metrics-version / METRICS_VERSION  — omit to use the monitor's current version
//	-from / FROM, -to / TO       — RFC3339; omit for server defaults (to=now, from=to-30d; max lookback 30d)
//	-segments / SEGMENTS         — comma-separated; empty string is the unsegmented series.
//	                               omit to return all segments
func main() {
	clientID := flag.String("client-id", os.Getenv("SYNQ_CLIENT_ID"), "OAuth client id (SYNQ_CLIENT_ID)")
	clientSecret := flag.String("client-secret", os.Getenv("SYNQ_CLIENT_SECRET"), "OAuth client secret (SYNQ_CLIENT_SECRET)")
	synqPath := flag.String("synq-path", envOr("SYNQ_PATH", os.Getenv("MONITOR_PATH")), "monitor synq path (SYNQ_PATH / MONITOR_PATH)")
	host := flag.String("endpoint", envOr("API_ENDPOINT", "developer.synq.io"), "API host without port (API_ENDPOINT)")
	metricsVersionRaw := flag.String("metrics-version", os.Getenv("METRICS_VERSION"), "metrics version (METRICS_VERSION); omit for current")
	fromRaw := flag.String("from", os.Getenv("FROM"), "RFC3339 start (FROM); omit for server default")
	toRaw := flag.String("to", os.Getenv("TO"), "RFC3339 end (TO); omit for now")
	segmentsRaw := flag.String("segments", os.Getenv("SEGMENTS"), "comma-separated segments (SEGMENTS); omit for all")
	flag.Parse()

	segmentsSet := envSet("SEGMENTS")
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "segments" {
			segmentsSet = true
		}
	})

	if *clientID == "" || *clientSecret == "" {
		panic("-client-id / SYNQ_CLIENT_ID and -client-secret / SYNQ_CLIENT_SECRET must be set")
	}
	if *synqPath == "" {
		panic("-synq-path / SYNQ_PATH must be set (e.g. monitor::<asset_path>::volume)")
	}

	req := &predictionsv1.GetMonitorPredictionsRequest{
		SynqPath: *synqPath,
	}

	if *metricsVersionRaw != "" {
		n, err := strconv.ParseInt(*metricsVersionRaw, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("-metrics-version: %v", err))
		}
		v := int32(n)
		req.MetricsVersion = &v
	}

	if *fromRaw != "" {
		t, err := time.Parse(time.RFC3339, *fromRaw)
		if err != nil {
			panic(fmt.Sprintf("-from: %v", err))
		}
		req.From = timestamppb.New(t)
	}
	if *toRaw != "" {
		t, err := time.Parse(time.RFC3339, *toRaw)
		if err != nil {
			panic(fmt.Sprintf("-to: %v", err))
		}
		req.To = timestamppb.New(t)
	}

	if segmentsSet {
		req.Segments = strings.Split(*segmentsRaw, ",")
	}

	ctx := context.Background()
	apiURL := fmt.Sprintf("%s:443", *host)
	tokenURL := fmt.Sprintf("https://%s/oauth2/token", *host)

	oauthTokenSource := oauth.TokenSource{TokenSource: (&clientcredentials.Config{
		ClientID:     *clientID,
		ClientSecret: *clientSecret,
		TokenURL:     tokenURL,
	}).TokenSource(ctx)}

	conn, err := grpc.NewClient(apiURL,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithPerRPCCredentials(oauthTokenSource),
		grpc.WithAuthority(*host),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("Connected to %s\n", *host)

	resp, err := predictionsv1grpc.NewMonitorPredictionsServiceClient(conn).GetMonitorPredictions(ctx, req)
	if err != nil {
		panic(err)
	}

	versionLabel := "current"
	if req.MetricsVersion != nil {
		versionLabel = strconv.Itoa(int(*req.MetricsVersion))
	}
	fromLabel, toLabel := "default", "default"
	if req.From != nil {
		fromLabel = req.From.AsTime().UTC().Format(time.RFC3339)
	}
	if req.To != nil {
		toLabel = req.To.AsTime().UTC().Format(time.RFC3339)
	}

	fmt.Printf("synq_path=%s version=%s from=%s to=%s predictions=%d\n",
		*synqPath, versionLabel, fromLabel, toLabel, len(resp.Predictions))

	for i, p := range resp.Predictions {
		seg := "<nil>"
		if p.Segment != nil {
			seg = fmt.Sprintf("%q", *p.Segment)
		}
		vu, vl := "nil", "nil"
		if p.UpperBound != nil {
			vu = fmt.Sprintf("%.6g", p.UpperBound.Value)
		}
		if p.LowerBound != nil {
			vl = fmt.Sprintf("%.6g", p.LowerBound.Value)
		}
		st := "nil"
		if p.Stddev != nil {
			st = fmt.Sprintf("%.6g", *p.Stddev)
		}
		pattern := ""
		if p.Pattern != nil {
			pattern = *p.Pattern
		}
		fmt.Printf("%d  t=%s  m=%s  s=%s  v=%.6g  e=%.6g  st=%s  vl=%s  vu=%s  p=%s  f=%s  corrected=%v\n",
			i, p.ScheduledAt.AsTime().UTC().Format(time.RFC3339), p.MetricId, seg, p.Value, p.Expected, st, vl, vu, pattern, p.Field, p.IsCorrected)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}
