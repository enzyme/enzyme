package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSlogHandler_WithSpan(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := NewSlogHandler(inner, false, "test")
	logger := slog.New(handler)

	// Create a real span context
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger.InfoContext(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "trace_id") {
		t.Fatalf("expected trace_id in log output, got: %s", output)
	}
	if !strings.Contains(output, "span_id") {
		t.Fatalf("expected span_id in log output, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Fatalf("expected original message in log output, got: %s", output)
	}
}

func TestSlogHandler_WithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := NewSlogHandler(inner, false, "test")
	logger := slog.New(handler)

	logger.InfoContext(context.Background(), "no span message")

	output := buf.String()
	if strings.Contains(output, "trace_id") {
		t.Fatalf("expected no trace_id without span, got: %s", output)
	}
	if !strings.Contains(output, "no span message") {
		t.Fatalf("expected original message in log output, got: %s", output)
	}
}

func TestSlogHandler_OtelFanout(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	// otel=true creates the fanoutHandler path (OTel bridge uses no-op provider)
	handler := NewSlogHandler(inner, true, "test-service")
	logger := slog.New(handler)

	logger.Info("fanout message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "fanout message") {
		t.Fatalf("expected message in console output: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Fatalf("expected attrs in console output: %s", output)
	}
}

func TestSlogHandler_OtelFanout_WithSpan(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := NewSlogHandler(inner, true, "test-service")
	logger := slog.New(handler)

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger.InfoContext(ctx, "traced fanout")

	output := buf.String()
	// traceInjector wraps fanout, so console output should have trace_id
	if !strings.Contains(output, "trace_id") {
		t.Fatalf("expected trace_id in console output with otel fanout: %s", output)
	}
	if !strings.Contains(output, "traced fanout") {
		t.Fatalf("expected message in console output: %s", output)
	}
}
