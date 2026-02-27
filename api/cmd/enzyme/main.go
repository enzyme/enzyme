package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/enzyme/api/internal/app"
	"github.com/enzyme/api/internal/config"
	"github.com/enzyme/api/internal/database"
	"github.com/enzyme/api/internal/logging"
	"github.com/enzyme/api/internal/seed"
)

func main() {
	// Check for subcommands before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		runSeed(os.Args[2:])
		return
	}

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
	logging.Setup(cfg.Log, cfg.Telemetry.Enabled)

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

func runSeed(args []string) {
	// Parse flags (supports --config, --database.path, etc.)
	flags := config.SetupFlags()
	if err := flags.Parse(args); err != nil {
		slog.Error("error parsing flags", "error", err)
		os.Exit(1)
	}

	configPath, _ := flags.GetString("config")

	cfg, err := config.Load(configPath, flags)
	if err != nil {
		slog.Error("error loading config", "error", err)
		os.Exit(1)
	}

	logging.Setup(cfg.Log, cfg.Telemetry.Enabled)

	// Open database and run migrations (no full app startup)
	db, err := database.Open(cfg.Database.Path, database.Options{
		MaxOpenConns: cfg.Database.MaxOpenConns,
		BusyTimeout:  cfg.Database.BusyTimeout,
		CacheSize:    cfg.Database.CacheSize,
		MmapSize:     cfg.Database.MmapSize,
	})
	if err != nil {
		slog.Error("error opening database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		slog.Error("error running migrations", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := seed.Run(ctx, db.DB); err != nil {
		slog.Error("error seeding database", "error", err)
		os.Exit(1)
	}
}
