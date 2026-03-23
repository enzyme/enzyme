package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	// Configure logging.
	logLevel := envOr("RELAY_LOG_LEVEL", "info")
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	// Initialize FCM client (optional — relay can run with only one platform).
	var fcm *FCMClient
	if credFile := os.Getenv("RELAY_FCM_CREDENTIALS_FILE"); credFile != "" {
		var err error
		fcm, err = NewFCMClient(credFile)
		if err != nil {
			slog.Error("failed to initialize FCM client", "error", err)
			os.Exit(1)
		}
		slog.Info("FCM client initialized")
	} else {
		slog.Warn("RELAY_FCM_CREDENTIALS_FILE not set, FCM disabled")
	}

	// Initialize APNs client (optional).
	var apns *APNsClient
	if keyFile := os.Getenv("RELAY_APNS_KEY_FILE"); keyFile != "" {
		keyID := os.Getenv("RELAY_APNS_KEY_ID")
		teamID := os.Getenv("RELAY_APNS_TEAM_ID")
		bundleID := envOr("RELAY_APNS_BUNDLE_ID", "im.enzyme.mobile")
		production := envOr("RELAY_APNS_PRODUCTION", "true") == "true"

		if keyID == "" || teamID == "" {
			slog.Error("RELAY_APNS_KEY_ID and RELAY_APNS_TEAM_ID are required when RELAY_APNS_KEY_FILE is set")
			os.Exit(1)
		}

		var err error
		apns, err = NewAPNsClient(keyFile, keyID, teamID, bundleID, production)
		if err != nil {
			slog.Error("failed to initialize APNs client", "error", err)
			os.Exit(1)
		}
		slog.Info("APNs client initialized", "bundle_id", bundleID, "production", production)
	} else {
		slog.Warn("RELAY_APNS_KEY_FILE not set, APNs disabled")
	}

	if fcm == nil && apns == nil {
		slog.Error("at least one push platform must be configured (FCM or APNs)")
		os.Exit(1)
	}

	// Graceful shutdown context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Rate limiter.
	rateLimitStr := envOr("RELAY_RATE_LIMIT", "120")
	rateLimit, err := strconv.Atoi(rateLimitStr)
	if err != nil || rateLimit <= 0 {
		slog.Error("invalid RELAY_RATE_LIMIT", "value", rateLimitStr)
		os.Exit(1)
	}
	burstStr := envOr("RELAY_BURST", "5")
	burst, err := strconv.Atoi(burstStr)
	if err != nil || burst <= 0 {
		slog.Error("invalid RELAY_BURST", "value", burstStr)
		os.Exit(1)
	}
	rateLimiter := NewRateLimiter(ctx, rateLimit, burst)

	// Router.
	router := newRouter(fcm, apns, rateLimiter)

	// HTTP server.
	port := envOr("RELAY_PORT", "8090")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", "error", err)
		}
	}()

	slog.Info("push relay starting", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	// Wait for context cancellation to ensure clean shutdown logging.
	<-ctx.Done()
	slog.Info("push relay stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
