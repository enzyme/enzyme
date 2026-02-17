package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feather/api/internal/app"
	"github.com/feather/api/internal/config"
	"github.com/feather/api/internal/logging"
)

func main() {
	// Setup CLI flags
	flags := config.SetupFlags()
	if err := flags.Parse(os.Args[1:]); err != nil {
		slog.Error("error parsing flags", "error", err)
		os.Exit(1)
	}

	// Get config path from flags
	configPath, _ := flags.GetString("config")

	// Load configuration
	cfg, err := config.Load(configPath, flags)
	if err != nil {
		slog.Error("error loading config", "error", err)
		os.Exit(1)
	}

	// Setup structured logging
	logging.Setup(cfg.Log)

	// Create application
	application, err := app.New(cfg)
	if err != nil {
		slog.Error("error creating application", "error", err)
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()

		// Give server time to shutdown gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := application.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", "error", err)
		}
	}()

	// Start application
	if err := application.Start(ctx); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
