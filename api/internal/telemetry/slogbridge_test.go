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

func TestSlogBridge_WithSpan(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	bridge := NewSlogBridge(inner)
	logger := slog.New(bridge)

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

func TestSlogBridge_WithoutSpan(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	bridge := NewSlogBridge(inner)
	logger := slog.New(bridge)

	logger.InfoContext(context.Background(), "no span message")

	output := buf.String()
	if strings.Contains(output, "trace_id") {
		t.Fatalf("expected no trace_id without span, got: %s", output)
	}
	if !strings.Contains(output, "no span message") {
		t.Fatalf("expected original message in log output, got: %s", output)
	}
}

func TestSlogBridge_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	bridge := NewSlogBridge(inner)
	logger := slog.New(bridge)

	childLogger := logger.With("key", "value")
	childLogger.InfoContext(context.Background(), "with attrs")

	output := buf.String()
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Fatalf("expected key/value attrs in log output, got: %s", output)
	}
}

func TestSlogBridge_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	bridge := NewSlogBridge(inner)
	logger := slog.New(bridge)

	grouped := logger.WithGroup("mygroup")
	grouped.InfoContext(context.Background(), "grouped message", "field", "val")

	output := buf.String()
	if !strings.Contains(output, "mygroup") {
		t.Fatalf("expected group in log output, got: %s", output)
	}
}
