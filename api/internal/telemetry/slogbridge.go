package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// SlogBridge wraps an slog.Handler to inject trace_id and span_id from the
// context into every log record. If no span is active, the fields are omitted.
type SlogBridge struct {
	inner slog.Handler
}

// NewSlogBridge returns a new SlogBridge wrapping the given handler.
func NewSlogBridge(inner slog.Handler) *SlogBridge {
	return &SlogBridge{inner: inner}
}

func (h *SlogBridge) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *SlogBridge) Handle(ctx context.Context, record slog.Record) error {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		record.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record)
}

func (h *SlogBridge) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlogBridge{inner: h.inner.WithAttrs(attrs)}
}

func (h *SlogBridge) WithGroup(name string) slog.Handler {
	return &SlogBridge{inner: h.inner.WithGroup(name)}
}
