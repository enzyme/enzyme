package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/enzyme/api/internal/auth"
	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/config"
	"github.com/enzyme/api/internal/database"
	"github.com/enzyme/api/internal/email"
	"github.com/enzyme/api/internal/emoji"
	"github.com/enzyme/api/internal/file"
	"github.com/enzyme/api/internal/handler"
	"github.com/enzyme/api/internal/linkpreview"
	"github.com/enzyme/api/internal/message"
	"github.com/enzyme/api/internal/notification"
	"github.com/enzyme/api/internal/presence"
	"github.com/enzyme/api/internal/ratelimit"
	"github.com/enzyme/api/internal/scheduled"
	"github.com/enzyme/api/internal/server"
	"github.com/enzyme/api/internal/signing"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/thread"
	"github.com/enzyme/api/internal/user"
	"github.com/enzyme/api/internal/web"
	"github.com/enzyme/api/internal/workspace"
)

type App struct {
	Config              *config.Config
	DB                  *database.DB
	Server              *server.Server
	Hub                 *sse.Hub
	PresenceManager     *presence.Manager
	EmailService        *email.Service
	NotificationService *notification.Service
	EmailWorker         *notification.EmailWorker
	RateLimiter         *ratelimit.Limiter
	SessionStore        *auth.SessionStore
	LinkPreviewRepo     *linkpreview.Repository
	ScheduledWorker     *scheduled.Worker
}

func New(cfg *config.Config) (*App, error) {
	// Open database
	db, err := database.Open(cfg.Database.Path, database.Options{
		MaxOpenConns: cfg.Database.MaxOpenConns,
		BusyTimeout:  cfg.Database.BusyTimeout,
		CacheSize:    cfg.Database.CacheSize,
		MmapSize:     cfg.Database.MmapSize,
	})
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Initialize SSE hub
	hub := sse.NewHub(db.DB, cfg.SSE.EventRetention, cfg.SSE.CleanupInterval)

	// Initialize presence manager
	presenceManager := presence.NewManager(db.DB, hub)

	// Initialize email service
	emailService, err := email.NewService(cfg.Email, cfg.Server.PublicURL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	// Initialize repositories
	userRepo := user.NewRepository(db.DB)
	passwordResetRepo := auth.NewPasswordResetRepo(db.DB)
	workspaceRepo := workspace.NewRepository(db.DB)
	channelRepo := channel.NewRepository(db.DB)
	messageRepo := message.NewRepository(db.DB)
	fileRepo := file.NewRepository(db.DB)
	linkPreviewRepo := linkpreview.NewRepository(db.DB)
	linkPreviewFetcher := linkpreview.NewFetcher(linkPreviewRepo)
	emojiRepo := emoji.NewRepository(db.DB)
	threadRepo := thread.NewRepository(db.DB)
	scheduledRepo := scheduled.NewRepository(db.DB)

	// Initialize services
	authService := auth.NewService(userRepo, passwordResetRepo, cfg.Auth.BcryptCost)

	// Initialize notification service
	notificationPrefsRepo := notification.NewPreferencesRepository(db.DB)
	notificationPendingRepo := notification.NewPendingRepository(db.DB)
	notificationService := notification.NewService(notificationPrefsRepo, notificationPendingRepo, channelRepo, hub)
	notificationService.SetThreadSubscriptionProvider(threadRepo)

	// Initialize email worker
	emailWorker := notification.NewEmailWorker(notificationPendingRepo, userRepo, emailService, hub)

	// Initialize session store
	sessionStore := auth.NewSessionStore(db.DB, cfg.Auth.SessionDuration)

	// Initialize file URL signer
	if cfg.Files.SigningSecret == "" {
		// Derive secret file path from database directory
		secretPath := filepath.Join(filepath.Dir(cfg.Database.Path), ".signing_secret")
		if data, err := os.ReadFile(secretPath); err == nil && len(data) > 0 {
			cfg.Files.SigningSecret = strings.TrimSpace(string(data))
		} else {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				_ = db.Close()
				return nil, err
			}
			cfg.Files.SigningSecret = hex.EncodeToString(b)
			if err := os.MkdirAll(filepath.Dir(secretPath), 0700); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("creating data directory: %w", err)
			}
			if err := os.WriteFile(secretPath, []byte(cfg.Files.SigningSecret+"\n"), 0600); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("writing signing secret: %w", err)
			}
			slog.Info("generated signing secret", "path", secretPath)
		}
	}
	signer := signing.NewSigner(cfg.Files.SigningSecret)

	// Normalize publicURL to avoid double slashes in constructed URLs
	cfg.Server.PublicURL = strings.TrimRight(cfg.Server.PublicURL, "/")

	// Initialize SSE handler (kept separate as it requires streaming)
	sseHandler := sse.NewHandler(hub, workspaceRepo, cfg.SSE.HeartbeatInterval, cfg.SSE.ClientBufferSize)

	// Initialize main handler implementing StrictServerInterface
	h := handler.New(handler.Dependencies{
		AuthService:         authService,
		SessionStore:        sessionStore,
		UserRepo:            userRepo,
		WorkspaceRepo:       workspaceRepo,
		ChannelRepo:         channelRepo,
		MessageRepo:         messageRepo,
		FileRepo:            fileRepo,
		LinkPreviewRepo:     linkPreviewRepo,
		LinkPreviewFetcher:  linkPreviewFetcher,
		ThreadRepo:          threadRepo,
		EmojiRepo:           emojiRepo,
		ScheduledRepo:       scheduledRepo,
		NotificationService: notificationService,
		Hub:                 hub,
		Signer:              signer,
		StoragePath:         cfg.Files.StoragePath,
		MaxUploadSize:       cfg.Files.MaxUploadSize,
		PublicURL:           cfg.Server.PublicURL,
	})

	// Initialize scheduled message worker
	scheduledWorker := scheduled.NewWorker(scheduledRepo, h)

	// Build rate limiter (nil if disabled)
	var limiter *ratelimit.Limiter
	if cfg.RateLimit.Enabled {
		rules := []ratelimit.Rule{
			{Method: "POST", Path: "/api/auth/login", Limit: cfg.RateLimit.Login.Limit, Window: cfg.RateLimit.Login.Window},
			{Method: "POST", Path: "/api/auth/register", Limit: cfg.RateLimit.Register.Limit, Window: cfg.RateLimit.Register.Window},
			{Method: "POST", Path: "/api/auth/forgot-password", Limit: cfg.RateLimit.ForgotPassword.Limit, Window: cfg.RateLimit.ForgotPassword.Window},
			{Method: "POST", Path: "/api/auth/reset-password", Limit: cfg.RateLimit.ResetPassword.Limit, Window: cfg.RateLimit.ResetPassword.Window},
		}
		limiter = ratelimit.NewLimiter(rules)
	}

	// Create embedded SPA handler if web client is bundled
	var spaHandler http.Handler
	if web.HasContent() {
		spaHandler = web.Handler()
		slog.Info("embedded web client enabled")
	} else {
		slog.Info("embedded web client not found, serve frontend separately")
	}

	// Create router with generated handlers
	router := server.NewRouter(h, sseHandler, sessionStore, limiter, cfg.Server.AllowedOrigins, spaHandler)

	// Build TLS options
	tlsOpts := server.TLSOptions{
		Mode:     cfg.Server.TLS.Mode,
		CertFile: cfg.Server.TLS.CertFile,
		KeyFile:  cfg.Server.TLS.KeyFile,
		Domain:   cfg.Server.TLS.Auto.Domain,
		Email:    cfg.Server.TLS.Auto.Email,
		CacheDir: cfg.Server.TLS.Auto.CacheDir,
	}
	if tlsOpts.Mode == "auto" {
		if err := os.MkdirAll(tlsOpts.CacheDir, 0700); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("creating TLS cache directory: %w", err)
		}
	}

	// Create server
	srv := server.New(cfg.Server.Host, cfg.Server.Port, router, tlsOpts,
		cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout)

	return &App{
		Config:              cfg,
		DB:                  db,
		Server:              srv,
		Hub:                 hub,
		PresenceManager:     presenceManager,
		EmailService:        emailService,
		NotificationService: notificationService,
		EmailWorker:         emailWorker,
		RateLimiter:         limiter,
		SessionStore:        sessionStore,
		LinkPreviewRepo:     linkPreviewRepo,
		ScheduledWorker:     scheduledWorker,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	// Start SSE hub
	go a.Hub.Run(ctx)

	// Start presence manager
	go a.PresenceManager.Start(ctx)

	// Start email worker
	go a.EmailWorker.Start(ctx)

	// Start scheduled message worker
	go a.ScheduledWorker.Start(ctx)

	// Start rate limiter cleanup
	if a.RateLimiter != nil {
		go func() {
			ticker := time.NewTicker(10 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					a.RateLimiter.Cleanup()
				}
			}
		}()
	}

	// Start expired session cleanup
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = a.SessionStore.DeleteExpired()
			}
		}
	}()

	// Start link preview cache cleanup (daily)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = a.LinkPreviewRepo.CleanExpiredCache(context.Background())
			}
		}
	}()

	slog.Info("starting enzyme backend",
		"addr", a.Server.Addr(),
		"database", a.Config.Database.Path,
		"file_storage", a.Config.Files.StoragePath,
		"tls", a.Server.TLSMode(),
		"email", a.EmailService.IsEnabled(),
	)

	return a.Server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.Server.Shutdown(ctx); err != nil {
		return err
	}
	return a.DB.Close()
}
