package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/config"
	"github.com/feather/api/internal/database"
	"github.com/feather/api/internal/email"
	"github.com/feather/api/internal/emoji"
	"github.com/feather/api/internal/file"
	"github.com/feather/api/internal/handler"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/notification"
	"github.com/feather/api/internal/presence"
	"github.com/feather/api/internal/ratelimit"
	"github.com/feather/api/internal/server"
	"github.com/feather/api/internal/signing"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/thread"
	"github.com/feather/api/internal/user"
	"github.com/feather/api/internal/workspace"
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
}

func New(cfg *config.Config) (*App, error) {
	// Open database
	db, err := database.Open(cfg.Database.Path)
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
	emojiRepo := emoji.NewRepository(db.DB)
	threadRepo := thread.NewRepository(db.DB)

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
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			_ = db.Close()
			return nil, err
		}
		cfg.Files.SigningSecret = hex.EncodeToString(b)
		log.Printf("No files.signing_secret configured; generated a random one (set FEATHER_FILES_SIGNING_SECRET for persistence across restarts)")
	}
	signer := signing.NewSigner(cfg.Files.SigningSecret)

	// Normalize publicURL to avoid double slashes in constructed URLs
	cfg.Server.PublicURL = strings.TrimRight(cfg.Server.PublicURL, "/")

	// Initialize SSE handler (kept separate as it requires streaming)
	sseHandler := sse.NewHandler(hub, workspaceRepo)

	// Initialize main handler implementing StrictServerInterface
	h := handler.New(handler.Dependencies{
		AuthService:         authService,
		SessionStore:        sessionStore,
		UserRepo:            userRepo,
		WorkspaceRepo:       workspaceRepo,
		ChannelRepo:         channelRepo,
		MessageRepo:         messageRepo,
		FileRepo:            fileRepo,
		ThreadRepo:          threadRepo,
		EmojiRepo:           emojiRepo,
		NotificationService: notificationService,
		Hub:                 hub,
		Signer:              signer,
		StoragePath:         cfg.Files.StoragePath,
		MaxUploadSize:       cfg.Files.MaxUploadSize,
		PublicURL:           cfg.Server.PublicURL,
	})

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

	// Create router with generated handlers
	router := server.NewRouter(h, sseHandler, sessionStore, limiter)

	// Create server
	srv := server.New(cfg.Server.Host, cfg.Server.Port, router)

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
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	// Start SSE hub
	go a.Hub.Run(ctx)

	// Start presence manager
	go a.PresenceManager.Start(ctx)

	// Start email worker
	go a.EmailWorker.Start(ctx)

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

	log.Printf("Feather backend starting on %s", a.Server.Addr())
	log.Printf("Database: %s", a.Config.Database.Path)
	log.Printf("File storage: %s", a.Config.Files.StoragePath)
	if a.EmailService.IsEnabled() {
		log.Printf("Email: enabled")
	} else {
		log.Printf("Email: disabled (no SMTP configured)")
	}

	return a.Server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	if err := a.Server.Shutdown(ctx); err != nil {
		return err
	}
	return a.DB.Close()
}
