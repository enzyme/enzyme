package handler

import (
	"context"
	"net/http"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/emoji"
	"github.com/feather/api/internal/file"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/notification"
	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/signing"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/thread"
	"github.com/feather/api/internal/user"
	"github.com/feather/api/internal/workspace"
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
	threadRepo          *thread.Repository
	emojiRepo           *emoji.Repository
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
	ThreadRepo          *thread.Repository
	EmojiRepo           *emoji.Repository
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
		threadRepo:          deps.ThreadRepo,
		emojiRepo:           deps.EmojiRepo,
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
