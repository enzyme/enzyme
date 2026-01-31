package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/feather/api/internal/app"
	"github.com/feather/api/internal/config"
)

func main() {
	// Setup CLI flags
	flags := config.SetupFlags()
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}

	// Get config path from flags
	configPath, _ := flags.GetString("config")

	// Load configuration
	cfg, err := config.Load(configPath, flags)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create application
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Error creating application: %v", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()

		// Give server time to shutdown gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := application.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	// Start application
	if err := application.Start(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}
