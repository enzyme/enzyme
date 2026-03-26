package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/trace"
)

// NewSlogHandler returns an slog.Handler that injects trace_id and span_id
// into log records. When otelLogs is true, records are also sent to the OTel
// log pipeline via otelslog. The trace injection happens before fan-out so
// both the console handler and OTel handler see the enriched record.
func NewSlogHandler(inner slog.Handler, otelLogs bool, serviceName string) slog.Handler {
	if !otelLogs {
		return &traceInjector{inner: inner}
	}
	// traceInjector wraps the fanout so both handlers receive enriched records.
	return &traceInjector{
		inner: &fanoutHandler{
			console: inner,
			otel:    otelslog.NewHandler(serviceName),
		},
	}
}

// traceInjector injects trace_id and span_id from context into log records.
type traceInjector struct {
	inner slog.Handler
}

func (h *traceInjector) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *traceInjector) Handle(ctx context.Context, record slog.Record) error {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record)
}

func (h *traceInjector) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceInjector{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceInjector) WithGroup(name string) slog.Handler {
	return &traceInjector{inner: h.inner.WithGroup(name)}
}

// fanoutHandler sends each record to both the console and OTel handlers.
// Records are cloned before each handler to prevent shared-state corruption.
// Errors from one handler do not prevent the other from receiving the record.
type fanoutHandler struct {
	console slog.Handler
	otel    slog.Handler
}

func (h *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.console.Enabled(ctx, level) || h.otel.Enabled(ctx, level)
}

func (h *fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	// Clone before each handler so neither can corrupt the other's view.
	var firstErr error
	if h.console.Enabled(ctx, record.Level) {
		if err := h.console.Handle(ctx, record.Clone()); err != nil {
			firstErr = err
		}
	}
	if h.otel.Enabled(ctx, record.Level) {
		if err := h.otel.Handle(ctx, record.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &fanoutHandler{
		console: h.console.WithAttrs(attrs),
		otel:    h.otel.WithAttrs(attrs),
	}
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	return &fanoutHandler{
		console: h.console.WithGroup(name),
		otel:    h.otel.WithGroup(name),
	}
}
