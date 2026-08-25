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

	historyv1grpc "buf.build/gen/go/getsynq/api/grpc/go/synq/monitors/history/v1/historyv1grpc"
	historyv1 "buf.build/gen/go/getsynq/api/protocolbuffers/go/synq/monitors/history/v1"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Fetches prediction history for one monitor.
//
// Flags win over env vars of the same name.
//
// Required:
//
//	-client-id / SYNQ_CLIENT_ID, -client-secret / SYNQ_CLIENT_SECRET  — SCOPE_MONITORS_READ
//	-monitor-path / MONITOR_PATH  — e.g. monitor::ch-prod::default::runs::volume
//
// Optional:
//
//	-endpoint / API_ENDPOINT     — developer.synq.io (EU, default), api.us.synq.io (US), api.au.synq.io (AU)
//	-metrics-version / METRICS_VERSION  — default 1
//	-from / FROM, -to / TO       — RFC3339 (default: last 7 days → now)
//	-segments / SEGMENTS         — comma-separated; empty string is the unsegmented series.
//	                               omit to return all segments
func main() {
	clientID := flag.String("client-id", os.Getenv("SYNQ_CLIENT_ID"), "OAuth client id (SYNQ_CLIENT_ID)")
	clientSecret := flag.String("client-secret", os.Getenv("SYNQ_CLIENT_SECRET"), "OAuth client secret (SYNQ_CLIENT_SECRET)")
	monitorPath := flag.String("monitor-path", os.Getenv("MONITOR_PATH"), "monitor synq path (MONITOR_PATH)")
	host := flag.String("endpoint", envOr("API_ENDPOINT", "developer.synq.io"), "API host without port (API_ENDPOINT)")
	metricsVersion := flag.Int("metrics-version", envInt("METRICS_VERSION", 1), "metrics version (METRICS_VERSION)")
	fromRaw := flag.String("from", os.Getenv("FROM"), "RFC3339 start (FROM); default now-7d")
	toRaw := flag.String("to", os.Getenv("TO"), "RFC3339 end (TO); default now")
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
	if *monitorPath == "" {
		panic("-monitor-path / MONITOR_PATH must be set (e.g. monitor::<asset_path>::volume)")
	}

	to := time.Now().UTC()
	from := to.Add(-7 * 24 * time.Hour)
	if *fromRaw != "" {
		t, err := time.Parse(time.RFC3339, *fromRaw)
		if err != nil {
			panic(fmt.Sprintf("-from: %v", err))
		}
		from = t
	}
	if *toRaw != "" {
		t, err := time.Parse(time.RFC3339, *toRaw)
		if err != nil {
			panic(fmt.Sprintf("-to: %v", err))
		}
		to = t
	}

	var segments []string
	if segmentsSet {
		segments = strings.Split(*segmentsRaw, ",")
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

	resp, err := historyv1grpc.NewHistoryServiceClient(conn).History(ctx, &historyv1.HistoryRequest{
		MonitorPath:    *monitorPath,
		MetricsVersion: int32(*metricsVersion),
		Segments:       segments,
		From:           timestamppb.New(from),
		To:             timestamppb.New(to),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("monitor=%s version=%d from=%s to=%s predictions=%d\n",
		*monitorPath, *metricsVersion, from.Format(time.RFC3339), to.Format(time.RFC3339), len(resp.Predictions))

	for i, p := range resp.Predictions {
		vu, vl := "nil", "nil"
		if p.Vu != nil {
			vu = fmt.Sprintf("%.6g", p.Vu.Value)
		}
		if p.Vl != nil {
			vl = fmt.Sprintf("%.6g", p.Vl.Value)
		}
		fmt.Printf("%d  t=%s  m=%s  s=%q  v=%.6g  e=%.6g  st=%.6g  vl=%s  vu=%s  p=%s  f=%s\n",
			i, p.T.AsTime().UTC().Format(time.RFC3339), p.M, p.S, p.V, p.E, p.St, vl, vu, p.P, p.F)
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

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("%s: %v", key, err))
	}
	return n
}
