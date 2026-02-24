package handler

import (
	"context"
	"net/http"

	"github.com/enzyme/api/internal/auth"
	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/emoji"
	"github.com/enzyme/api/internal/file"
	"github.com/enzyme/api/internal/linkpreview"
	"github.com/enzyme/api/internal/message"
	"github.com/enzyme/api/internal/notification"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/scheduled"
	"github.com/enzyme/api/internal/signing"
	"github.com/enzyme/api/internal/sse"
	"github.com/enzyme/api/internal/thread"
	"github.com/enzyme/api/internal/user"
	"github.com/enzyme/api/internal/workspace"
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
	notificationService *notification.Service
	hub                 *sse.Hub
	signer              *signing.Signer
	storagePath         string
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
	NotificationService *notification.Service
	Hub                 *sse.Hub
	Signer              *signing.Signer
	StoragePath         string
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
		notificationService: deps.NotificationService,
		hub:                 deps.Hub,
		signer:              deps.Signer,
		storagePath:         deps.StoragePath,
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
