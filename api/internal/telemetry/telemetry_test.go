package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/enzyme/api/internal/config"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNoop(t *testing.T) {
	tel := Noop()
	if tel == nil {
		t.Fatal("Noop() returned nil")
	}
	if tel.tracerProvider != nil {
		t.Fatal("Noop tracer provider should be nil")
	}
	if tel.meterProvider != nil {
		t.Fatal("Noop meter provider should be nil")
	}

	// Shutdown should not error
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatalf("Noop Shutdown() returned error: %v", err)
	}
}

func TestInit_InvalidProtocol(t *testing.T) {
	// Init should still work — the protocol validation is in config.Validate.
	// But if we pass "grpc" with a bad endpoint, the exporter creation should
	// still succeed (connection is lazy for gRPC).
	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		Protocol:    "grpc",
		SampleRate:  1.0,
		ServiceName: "test",
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with grpc should not fail eagerly: %v", err)
	}
	defer tel.Shutdown(context.Background())

	if tel.tracerProvider == nil {
		t.Fatal("tracer provider should not be nil")
	}
	if tel.meterProvider == nil {
		t.Fatal("meter provider should not be nil")
	}
}

func TestInit_HTTP(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4318",
		Protocol:    "http",
		SampleRate:  0.5,
		ServiceName: "test-http",
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with http should not fail eagerly: %v", err)
	}
	defer tel.Shutdown(context.Background())
}

func TestInit_SamplerZero(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		Protocol:    "grpc",
		SampleRate:  0,
		ServiceName: "test-zero",
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with zero sample rate should not fail: %v", err)
	}
	defer tel.Shutdown(context.Background())
}

func TestMiddleware_CreatesSpans(t *testing.T) {
	// Set up in-memory span exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())
	otel.SetTracerProvider(tp)

	// Create a simple handler wrapped with our middleware
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := Middleware()(inner)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Force flush
	tp.ForceFlush(context.Background())

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least one span to be created")
	}
}
