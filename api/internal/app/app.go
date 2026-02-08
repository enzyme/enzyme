package app

import (
	"context"
	"log"

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
	"github.com/feather/api/internal/server"
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
	hub := sse.NewHub(db.DB)

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

	// Initialize session manager
	sessionManager := auth.NewSessionManager(db.DB, cfg.Auth.SessionDuration, cfg.Auth.SecureCookies)

	// Initialize SSE handler (kept separate as it requires streaming)
	sseHandler := sse.NewHandler(hub, workspaceRepo)

	// Initialize auth handler (needed for RequireAuth middleware on SSE routes)
	authHandler := auth.NewHandler(authService, sessionManager, workspaceRepo)

	// Initialize main handler implementing StrictServerInterface
	h := handler.New(handler.Dependencies{
		AuthService:         authService,
		SessionManager:      sessionManager,
		UserRepo:            userRepo,
		WorkspaceRepo:       workspaceRepo,
		ChannelRepo:         channelRepo,
		MessageRepo:         messageRepo,
		FileRepo:            fileRepo,
		ThreadRepo:          threadRepo,
		EmojiRepo:           emojiRepo,
		NotificationService: notificationService,
		Hub:                 hub,
		StoragePath:         cfg.Files.StoragePath,
		MaxUploadSize:       cfg.Files.MaxUploadSize,
	})

	// Create router with generated handlers
	router := server.NewRouter(h, sseHandler, authHandler, sessionManager)

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
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	// Start SSE hub
	go a.Hub.Run(ctx)

	// Start presence manager
	go a.PresenceManager.Start(ctx)

	// Start email worker
	go a.EmailWorker.Start(ctx)

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
