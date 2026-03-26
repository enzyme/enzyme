package telemetry

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/enzyme/server/internal/config"
)

const maxProxyBody = 1 << 20 // 1 MB

// NewOTLPProxy creates an http.Handler that forwards OTLP/HTTP trace
// requests from the browser to the configured collector.
func NewOTLPProxy(cfg config.TelemetryConfig) http.Handler {
	target := otlpHTTPTarget(cfg)
	client := &http.Client{Timeout: 10 * time.Second}

	slog.Info("OTLP trace proxy enabled", "target", target)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "application/x-protobuf") {
			http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
			return
		}

		body := http.MaxBytesReader(w, r.Body, maxProxyBody)
		defer body.Close()

		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if ct != "" {
			proxyReq.Header.Set("Content-Type", ct)
		}
		// Forward auth headers if the collector requires them.
		for k, v := range cfg.Headers {
			proxyReq.Header.Set(k, v)
		}

		resp, err := client.Do(proxyReq)
		if err != nil {
			slog.Warn("OTLP proxy error", "error", err)
			http.Error(w, "collector unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body) //nolint:errcheck
	})
}

// otlpHTTPTarget returns the full URL for the collector's OTLP/HTTP
// trace receiver.
//
// Resolution order:
//  1. Explicit frontend_endpoint config (if set)
//  2. Same endpoint as backend (works for cloud providers like Honeycomb
//     and Grafana Cloud that serve gRPC and HTTP on the same port)
//  3. For gRPC configs using the standard port 4317, swap to 4318
//     (the standard OTLP/HTTP port for local OTel Collectors)
func otlpHTTPTarget(cfg config.TelemetryConfig) string {
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}

	endpoint := cfg.FrontendEndpoint
	if endpoint == "" {
		endpoint = cfg.Endpoint
		if cfg.Protocol != "http" {
			// Only swap the port for the standard gRPC port (4317).
			// Cloud providers (Honeycomb, Grafana, Datadog) serve both
			// gRPC and HTTP on the same port (typically 443).
			host, port, err := net.SplitHostPort(endpoint)
			if err != nil {
				host = endpoint
				port = ""
			}
			if port == "4317" {
				endpoint = host + ":4318"
			}
		}
	}

	return scheme + "://" + endpoint + "/v1/traces"
}
