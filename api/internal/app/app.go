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
	"github.com/enzyme/api/internal/moderation"
	"github.com/enzyme/api/internal/notification"
	"github.com/enzyme/api/internal/presence"
	"github.com/enzyme/api/internal/pushnotification"
	"github.com/enzyme/api/internal/ratelimit"
	"github.com/enzyme/api/internal/scheduled"
	"github.com/enzyme/api/internal/scheduler"
	"github.com/enzyme/api/internal/server"
	"github.com/enzyme/api/internal/signing"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/storage"
	"github.com/enzyme/api/internal/telemetry"
	"github.com/enzyme/api/internal/thread"
	"github.com/enzyme/api/internal/user"
	"github.com/enzyme/api/internal/version"
	"github.com/enzyme/api/internal/web"
	"github.com/enzyme/api/internal/workspace"
)

type App struct {
	Config                *config.Config
	DB                    *database.DB
	Server                *server.Server
	Hub                   *sse.Hub
	PresenceManager       *presence.Manager
	EmailService          *email.Service
	NotificationService   *notification.Service
	EmailWorker           *notification.EmailWorker
	RateLimiter           *ratelimit.Limiter
	SessionStore          *auth.SessionStore
	emailVerificationRepo *auth.EmailVerificationRepo
	LinkPreviewRepo       *linkpreview.Repository
	ScheduledWorker       *scheduled.Worker
	passwordResetRepo     *auth.PasswordResetRepo
	pushTokenRepo         *pushnotification.Repository
	moderationRepo        *moderation.Repository
	scheduler             *scheduler.Scheduler
	Telemetry             *telemetry.Telemetry
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

	// Initialize telemetry (before other components so they can use global providers)
	var tel *telemetry.Telemetry
	if cfg.Telemetry.Enabled {
		tel, err = telemetry.Init(cfg.Telemetry, version.Version)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initializing telemetry: %w", err)
		}
		slog.Info("telemetry enabled", "endpoint", cfg.Telemetry.Endpoint, "protocol", cfg.Telemetry.Protocol)
	} else {
		tel = telemetry.Noop()
	}

	// Initialize SSE hub
	hub := sse.NewHub(db.DB, cfg.SSE.EventRetention)

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
	emailVerificationRepo := auth.NewEmailVerificationRepo(db.DB)
	workspaceRepo := workspace.NewRepository(db.DB)
	channelRepo := channel.NewRepository(db.DB)
	messageRepo := message.NewRepository(db.DB)
	fileRepo := file.NewRepository(db.DB)
	linkPreviewRepo := linkpreview.NewRepository(db.DB)
	linkPreviewFetcher := linkpreview.NewFetcher(linkPreviewRepo)
	emojiRepo := emoji.NewRepository(db.DB)
	threadRepo := thread.NewRepository(db.DB)
	scheduledRepo := scheduled.NewRepository(db.DB)
	moderationRepo := moderation.NewRepository(db.DB)

	// Initialize services
	authService := auth.NewService(userRepo, passwordResetRepo, emailVerificationRepo, cfg.Auth.BcryptCost)

	// Initialize notification service
	notificationPrefsRepo := notification.NewPreferencesRepository(db.DB)
	notificationPendingRepo := notification.NewPendingRepository(db.DB)
	notificationService := notification.NewService(notificationPrefsRepo, notificationPendingRepo, channelRepo, hub)
	notificationService.SetThreadSubscriptionProvider(threadRepo)

	// Initialize push notification service
	var pushTokenRepo *pushnotification.Repository
	if cfg.PushNotifications.Enabled {
		pushTokenRepo = pushnotification.NewRepository(db.DB)
		pushService := pushnotification.NewService(pushTokenRepo, cfg.PushNotifications.RelayURL)
		notificationService.SetPushService(&pushSenderAdapter{pushService}, cfg.Server.PublicURL)
		slog.Info("push notifications enabled", "relay_url", cfg.PushNotifications.RelayURL)
	}

	// Initialize email worker
	emailWorker := notification.NewEmailWorker(notificationPendingRepo, userRepo, emailService, hub)

	// Initialize session store
	sessionStore := auth.NewSessionStore(db.DB, cfg.Auth.SessionDuration)

	// Initialize storage backend
	var store storage.Storage
	switch cfg.Storage.Type {
	case "local":
		store = storage.NewLocal(cfg.Storage.Local.Path)
	case "s3":
		s3Store, err := storage.NewS3(cfg.Storage.S3)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("initializing S3 storage: %w", err)
		}
		if err := s3Store.CheckConnectivity(context.Background()); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("S3 connectivity check: %w", err)
		}
		store = s3Store
	case "off":
		// store remains nil — upload endpoints return 403
	}

	// Initialize file URL signer (only needed for local storage)
	if cfg.Storage.Type == "local" && cfg.Storage.Local.SigningSecret == "" {
		secretPath := filepath.Join(filepath.Dir(cfg.Database.Path), ".signing_secret")
		if data, err := os.ReadFile(secretPath); err == nil && len(data) > 0 {
			cfg.Storage.Local.SigningSecret = strings.TrimSpace(string(data))
		} else {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				_ = db.Close()
				return nil, err
			}
			cfg.Storage.Local.SigningSecret = hex.EncodeToString(b)
			if err := os.MkdirAll(filepath.Dir(secretPath), 0700); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("creating data directory: %w", err)
			}
			if err := os.WriteFile(secretPath, []byte(cfg.Storage.Local.SigningSecret+"\n"), 0600); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("writing signing secret: %w", err)
			}
			slog.Info("generated signing secret", "path", secretPath)
		}
	}
	signingSecret := cfg.Storage.Local.SigningSecret
	signer := signing.NewSigner(signingSecret)

	// Normalize publicURL to avoid double slashes in constructed URLs
	cfg.Server.PublicURL = strings.TrimRight(cfg.Server.PublicURL, "/")

	// Initialize SSE handler (kept separate as it requires streaming)
	sseHandler := sse.NewHandler(hub, workspaceRepo, channelRepo, cfg.SSE.HeartbeatInterval, cfg.SSE.ClientBufferSize)

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
		EmailService:        emailService,
		NotificationService: notificationService,
		PushTokenRepo:       pushTokenRepo,
		ModerationRepo:      moderationRepo,
		Hub:                 hub,
		Signer:              signer,
		Storage:             store,
		MaxUploadSize:       cfg.Storage.MaxUploadSize,
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
			{Method: "POST", Path: "/api/auth/verify-email", Limit: cfg.RateLimit.VerifyEmail.Limit, Window: cfg.RateLimit.VerifyEmail.Window},
			{Method: "POST", Path: "/api/auth/resend-verification", Limit: cfg.RateLimit.ResendVerification.Limit, Window: cfg.RateLimit.ResendVerification.Window},
		}
		limiter = ratelimit.NewLimiter(rules)
	}

	// Create embedded SPA handler if web client is bundled
	var spaHandler http.Handler
	if web.HasContent() {
		spaHandler = web.Handler(cfg.Telemetry.Enabled && cfg.Telemetry.Traces)
		slog.Info("embedded web client enabled")
	} else {
		slog.Info("embedded web client not found, serve frontend separately")
	}

	// Create OTLP proxy for frontend telemetry
	var otlpProxy http.Handler
	if cfg.Telemetry.Enabled && cfg.Telemetry.Traces {
		otlpProxy = telemetry.NewOTLPProxy(cfg.Telemetry)
	}

	// Create router with generated handlers
	router := server.NewRouter(h, sseHandler, sessionStore, moderationRepo, limiter, cfg.Server.AllowedOrigins, cfg.Telemetry.Enabled, spaHandler, otlpProxy)

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
		Config:                cfg,
		DB:                    db,
		Server:                srv,
		Hub:                   hub,
		PresenceManager:       presenceManager,
		EmailService:          emailService,
		NotificationService:   notificationService,
		EmailWorker:           emailWorker,
		RateLimiter:           limiter,
		SessionStore:          sessionStore,
		emailVerificationRepo: emailVerificationRepo,
		LinkPreviewRepo:       linkPreviewRepo,
		ScheduledWorker:       scheduledWorker,
		passwordResetRepo:     passwordResetRepo,
		pushTokenRepo:         pushTokenRepo,
		moderationRepo:        moderationRepo,
		scheduler:             scheduler.New(),
		Telemetry:             tel,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	// Init components that need startup work
	a.PresenceManager.Init()

	// Start SSE hub (needs its own goroutine for client register/unregister channels)
	go a.Hub.Run(ctx)

	// Register periodic tasks
	s := a.scheduler

	if a.RateLimiter != nil {
		s.Register(scheduler.Task{Name: "rate-limiter-cleanup", Interval: 10 * time.Minute, Fn: func(ctx context.Context) error { a.RateLimiter.Cleanup(); return nil }})
	}
	s.Register(scheduler.Task{Name: "session-cleanup", Interval: time.Hour, Fn: func(ctx context.Context) error { return a.SessionStore.DeleteExpired() }})
	s.Register(scheduler.Task{Name: "link-preview-cleanup", Interval: 24 * time.Hour, Fn: func(ctx context.Context) error { return a.LinkPreviewRepo.CleanExpiredCache(ctx) }})

	if a.Config.SSE.CleanupInterval > 0 {
		s.Register(scheduler.Task{Name: "sse-event-cleanup", Interval: a.Config.SSE.CleanupInterval, Fn: a.Hub.CleanupOldEvents, RunOnStart: true})
	}

	s.Register(scheduler.Task{Name: "presence-check", Interval: 10 * time.Second, Fn: a.PresenceManager.CheckPresence})
	s.Register(scheduler.Task{Name: "scheduled-messages", Interval: 30 * time.Second, Fn: a.ScheduledWorker.ProcessDue})
	s.Register(scheduler.Task{Name: "expired-ban-cleanup", Interval: time.Hour, Fn: a.moderationRepo.CleanupExpiredBans})
	s.Register(scheduler.Task{Name: "sqlite-optimize", Interval: 24 * time.Hour, Fn: func(ctx context.Context) error { _, err := a.DB.Exec("PRAGMA optimize(0x10002)"); return err }})

	if a.EmailService.IsEnabled() {
		s.Register(scheduler.Task{Name: "email-notifications", Interval: time.Minute, Fn: a.EmailWorker.ProcessPending})
		s.Register(scheduler.Task{Name: "password-reset-cleanup", Interval: 24 * time.Hour, Fn: a.passwordResetRepo.DeleteExpired})
		s.Register(scheduler.Task{Name: "email-verification-cleanup", Interval: 24 * time.Hour, Fn: a.emailVerificationRepo.DeleteExpired})
	}

	if a.pushTokenRepo != nil {
		s.Register(scheduler.Task{Name: "push-token-cleanup", Interval: 24 * time.Hour, Fn: func(ctx context.Context) error {
			n, err := a.pushTokenRepo.CleanupStale(ctx, time.Now().Add(-90*24*time.Hour))
			if err == nil && n > 0 {
				slog.Info("cleaned up stale push tokens", "count", n)
			}
			return err
		}})
	}

	s.Start(ctx)

	var storageInfo string
	switch a.Config.Storage.Type {
	case "local":
		storageInfo = "local:" + a.Config.Storage.Local.Path
	case "s3":
		storageInfo = "s3:" + a.Config.Storage.S3.Endpoint + "/" + a.Config.Storage.S3.Bucket
	default:
		storageInfo = a.Config.Storage.Type
	}
	slog.Info("starting enzyme backend",
		"addr", a.Server.Addr(),
		"database", a.Config.Database.Path,
		"storage", storageInfo,
		"tls", a.Server.TLSMode(),
		"email", a.EmailService.IsEnabled(),
	)

	return a.Server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	// Stop scheduler first so in-flight tasks finish before DB closes
	a.scheduler.Stop(ctx)

	if err := a.Server.Shutdown(ctx); err != nil {
		return err
	}
	// Flush telemetry before closing database
	if err := a.Telemetry.Shutdown(ctx); err != nil {
		slog.Error("telemetry shutdown error", "error", err)
	}
	return a.DB.Close()
}

// pushSenderAdapter bridges pushnotification.Service to the notification.PushSender interface.
type pushSenderAdapter struct {
	svc *pushnotification.Service
}

func (a *pushSenderAdapter) Send(ctx context.Context, userID string, data notification.PushNotificationData) bool {
	return a.svc.Send(ctx, userID, pushnotification.NotificationData{
		Title:       data.Title,
		Body:        data.Body,
		ChannelID:   data.ChannelID,
		MessageID:   data.MessageID,
		WorkspaceID: data.WorkspaceID,
		ServerURL:   data.ServerURL,
	})
}
