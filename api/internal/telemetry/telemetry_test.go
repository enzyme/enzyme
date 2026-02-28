package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/enzyme/api/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// saveGlobalProviders saves the current global OTel providers and returns a
// restore function that resets them. Call defer restore() in every test that
// calls Init (which sets the global providers).
func saveGlobalProviders(t *testing.T) func() {
	t.Helper()
	origTP := otel.GetTracerProvider()
	origMP := otel.GetMeterProvider()
	origLP := global.GetLoggerProvider()
	return func() {
		otel.SetTracerProvider(origTP)
		otel.SetMeterProvider(origMP)
		global.SetLoggerProvider(origLP)
	}
}

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
	if tel.logProvider != nil {
		t.Fatal("Noop log provider should be nil")
	}

	// Shutdown should not error
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatalf("Noop Shutdown() returned error: %v", err)
	}
}

func TestInit_GRPC(t *testing.T) {
	restore := saveGlobalProviders(t)
	defer restore()

	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		Protocol:    "grpc",
		Insecure:    true,
		SampleRate:  1.0,
		ServiceName: "test",
		Traces:      true,
		Metrics:     true,
		Logs:        true,
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
	if tel.logProvider == nil {
		t.Fatal("log provider should not be nil")
	}
}

func TestInit_HTTP(t *testing.T) {
	restore := saveGlobalProviders(t)
	defer restore()

	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4318",
		Protocol:    "http",
		Insecure:    true,
		SampleRate:  0.5,
		ServiceName: "test-http",
		Traces:      true,
		Metrics:     true,
		Logs:        true,
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with http should not fail eagerly: %v", err)
	}
	defer tel.Shutdown(context.Background())
}

func TestInit_WithHeaders(t *testing.T) {
	restore := saveGlobalProviders(t)
	defer restore()

	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		Protocol:    "grpc",
		Insecure:    true,
		SampleRate:  1.0,
		ServiceName: "test-headers",
		Traces:      true,
		Metrics:     true,
		Logs:        true,
		Headers: map[string]string{
			"x-honeycomb-team": "test-key",
			"authorization":    "Bearer token",
		},
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with headers should not fail: %v", err)
	}
	defer tel.Shutdown(context.Background())

	if tel.tracerProvider == nil {
		t.Fatal("tracer provider should not be nil")
	}
	if tel.meterProvider == nil {
		t.Fatal("meter provider should not be nil")
	}
	if tel.logProvider == nil {
		t.Fatal("log provider should not be nil")
	}
}

func TestInit_WithHeaders_HTTP(t *testing.T) {
	restore := saveGlobalProviders(t)
	defer restore()

	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4318",
		Protocol:    "http",
		Insecure:    true,
		SampleRate:  1.0,
		ServiceName: "test-headers-http",
		Traces:      true,
		Metrics:     true,
		Logs:        true,
		Headers: map[string]string{
			"x-api-key": "test-key",
		},
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with http headers should not fail: %v", err)
	}
	defer tel.Shutdown(context.Background())
}

func TestInit_SamplerZero(t *testing.T) {
	restore := saveGlobalProviders(t)
	defer restore()

	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		Protocol:    "grpc",
		Insecure:    true,
		SampleRate:  0,
		ServiceName: "test-zero",
		Traces:      true,
		Metrics:     true,
		Logs:        true,
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with zero sample rate should not fail: %v", err)
	}
	defer tel.Shutdown(context.Background())
}

func TestInit_PerSignalDisable(t *testing.T) {
	restore := saveGlobalProviders(t)
	defer restore()

	cfg := config.TelemetryConfig{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		Protocol:    "grpc",
		Insecure:    true,
		SampleRate:  1.0,
		ServiceName: "test-partial",
		Traces:      true,
		Metrics:     false,
		Logs:        false,
	}
	tel, err := Init(cfg, "test-version")
	if err != nil {
		t.Fatalf("Init() with partial signals should not fail: %v", err)
	}
	defer tel.Shutdown(context.Background())

	if tel.tracerProvider == nil {
		t.Fatal("tracer provider should not be nil when traces=true")
	}
	if tel.meterProvider != nil {
		t.Fatal("meter provider should be nil when metrics=false")
	}
	if tel.logProvider != nil {
		t.Fatal("log provider should be nil when logs=false")
	}
}

func TestMiddleware_CreatesSpans(t *testing.T) {
	// Set up in-memory span exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	origTP := otel.GetTracerProvider()
	origMP := otel.GetMeterProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		otel.SetTracerProvider(origTP)
		otel.SetMeterProvider(origMP)
	}()

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
