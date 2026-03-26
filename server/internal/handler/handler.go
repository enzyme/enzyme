package handler

import (
	"context"
	"net/http"

	"github.com/enzyme/server/internal/auth"
	"github.com/enzyme/server/internal/channel"
	"github.com/enzyme/server/internal/email"
	"github.com/enzyme/server/internal/emoji"
	"github.com/enzyme/server/internal/file"
	"github.com/enzyme/server/internal/linkpreview"
	"github.com/enzyme/server/internal/message"
	"github.com/enzyme/server/internal/moderation"
	"github.com/enzyme/server/internal/notification"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/pushnotification"
	"github.com/enzyme/server/internal/scheduled"
	"github.com/enzyme/server/internal/signing"
	"github.com/enzyme/server/internal/sse"
	"github.com/enzyme/server/internal/storage"
	"github.com/enzyme/server/internal/thread"
	"github.com/enzyme/server/internal/user"
	"github.com/enzyme/server/internal/workspace"
)

// Compile-time interface check
var _ openapi.StrictServerInterface = (*Handler)(nil)

// Handler implements the generated StrictServerInterface
type Handler struct {
	authService         *auth.Service
	sessionStore        *auth.SessionStore
	userRepo            *user.Repository
	workspaceRepo       *workspace.Repository
	channelRepo         *channel.Repository
	messageRepo         *message.Repository
	fileRepo            *file.Repository
	linkPreviewRepo     *linkpreview.Repository
	linkPreviewFetcher  *linkpreview.Fetcher
	threadRepo          *thread.Repository
	emojiRepo           *emoji.Repository
	scheduledRepo       *scheduled.Repository
	emailService        *email.Service
	notificationService *notification.Service
	pushTokenRepo       *pushnotification.Repository
	moderationRepo      *moderation.Repository
	hub                 *sse.Hub
	signer              *signing.Signer
	storage             storage.Storage
	maxUploadSize       int64
	publicURL           string
}

// Dependencies holds all dependencies for the Handler
type Dependencies struct {
	AuthService         *auth.Service
	SessionStore        *auth.SessionStore
	UserRepo            *user.Repository
	WorkspaceRepo       *workspace.Repository
	ChannelRepo         *channel.Repository
	MessageRepo         *message.Repository
	FileRepo            *file.Repository
	LinkPreviewRepo     *linkpreview.Repository
	LinkPreviewFetcher  *linkpreview.Fetcher
	ThreadRepo          *thread.Repository
	EmojiRepo           *emoji.Repository
	ScheduledRepo       *scheduled.Repository
	EmailService        *email.Service
	NotificationService *notification.Service
	PushTokenRepo       *pushnotification.Repository
	ModerationRepo      *moderation.Repository
	Hub                 *sse.Hub
	Signer              *signing.Signer
	Storage             storage.Storage
	MaxUploadSize       int64
	PublicURL           string
}

// New creates a new Handler with all dependencies
func New(deps Dependencies) *Handler {
	return &Handler{
		authService:         deps.AuthService,
		sessionStore:        deps.SessionStore,
		userRepo:            deps.UserRepo,
		workspaceRepo:       deps.WorkspaceRepo,
		channelRepo:         deps.ChannelRepo,
		messageRepo:         deps.MessageRepo,
		fileRepo:            deps.FileRepo,
		linkPreviewRepo:     deps.LinkPreviewRepo,
		linkPreviewFetcher:  deps.LinkPreviewFetcher,
		threadRepo:          deps.ThreadRepo,
		emojiRepo:           deps.EmojiRepo,
		scheduledRepo:       deps.ScheduledRepo,
		emailService:        deps.EmailService,
		notificationService: deps.NotificationService,
		pushTokenRepo:       deps.PushTokenRepo,
		moderationRepo:      deps.ModerationRepo,
		hub:                 deps.Hub,
		signer:              deps.Signer,
		storage:             deps.Storage,
		maxUploadSize:       deps.MaxUploadSize,
		publicURL:           deps.PublicURL,
	}
}

// Context key for storing the http.Request
type contextKey string

const requestKey contextKey = "httpRequest"

// WithRequest returns a context with the http.Request attached
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestKey, r)
}

// GetRequest extracts the http.Request from context
func GetRequest(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestKey).(*http.Request)
	return r
}

// getUserID gets the current user ID from context (set by TokenMiddleware)
func (h *Handler) getUserID(ctx context.Context) string {
	return auth.GetUserID(ctx)
}
