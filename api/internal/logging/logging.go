package logging

import (
	"log/slog"
	"os"

	"github.com/enzyme/api/internal/config"
	"github.com/enzyme/api/internal/telemetry"
)

// Setup configures the default slog logger based on the provided config.
// This also bridges the standard "log" package via slog.SetDefault (Go 1.22+).
// When otelLogs is true, log records are enriched with trace_id and span_id
// and forwarded to the OTel log pipeline.
func Setup(cfg config.LogConfig, otelLogs bool, serviceName string) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	handler = telemetry.NewSlogHandler(handler, otelLogs, serviceName)

	slog.SetDefault(slog.New(handler))
}
