package logging

import (
	"log/slog"
	"os"

	"github.com/enzyme/api/internal/config"
	"github.com/enzyme/api/internal/telemetry"
)

// Setup configures the default slog logger based on the provided config.
// This also bridges the standard "log" package via slog.SetDefault (Go 1.22+).
// When telemetryEnabled is true, log records are enriched with trace_id and
// span_id from the active span context.
func Setup(cfg config.LogConfig, telemetryEnabled bool) {
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

	if telemetryEnabled {
		handler = telemetry.NewSlogBridge(handler)
	}

	slog.SetDefault(slog.New(handler))
}
