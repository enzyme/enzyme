package telemetry

import (
	"testing"

	"github.com/enzyme/server/internal/config"
)

func TestOTLPHTTPTarget(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.TelemetryConfig
		expected string
	}{
		{
			name: "gRPC on standard port derives HTTP port 4318",
			cfg: config.TelemetryConfig{
				Endpoint: "localhost:4317",
				Protocol: "grpc",
				Insecure: true,
			},
			expected: "http://localhost:4318/v1/traces",
		},
		{
			name: "gRPC on non-standard port keeps same port (cloud provider)",
			cfg: config.TelemetryConfig{
				Endpoint: "api.honeycomb.io:443",
				Protocol: "grpc",
				Insecure: false,
			},
			expected: "https://api.honeycomb.io:443/v1/traces",
		},
		{
			name: "gRPC without port keeps endpoint as-is",
			cfg: config.TelemetryConfig{
				Endpoint: "tempo-us-central1.grafana.net",
				Protocol: "grpc",
				Insecure: false,
			},
			expected: "https://tempo-us-central1.grafana.net/v1/traces",
		},
		{
			name: "HTTP protocol uses endpoint directly",
			cfg: config.TelemetryConfig{
				Endpoint: "localhost:4318",
				Protocol: "http",
				Insecure: true,
			},
			expected: "http://localhost:4318/v1/traces",
		},
		{
			name: "HTTP with TLS",
			cfg: config.TelemetryConfig{
				Endpoint: "otel.example.com:443",
				Protocol: "http",
				Insecure: false,
			},
			expected: "https://otel.example.com:443/v1/traces",
		},
		{
			name: "explicit frontend_endpoint overrides derivation",
			cfg: config.TelemetryConfig{
				Endpoint:         "localhost:4317",
				Protocol:         "grpc",
				Insecure:         true,
				FrontendEndpoint: "collector.internal:4318",
			},
			expected: "http://collector.internal:4318/v1/traces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := otlpHTTPTarget(tt.cfg)
			if got != tt.expected {
				t.Errorf("otlpHTTPTarget() = %q, want %q", got, tt.expected)
			}
		})
	}
}
