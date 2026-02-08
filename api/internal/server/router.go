package server

import (
	"context"
	"net/http"

	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/handler"
	"github.com/feather/api/internal/sse"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

// NewRouter creates a new HTTP router with all routes registered
func NewRouter(h *handler.Handler, sseHandler *sse.Handler, authHandler *auth.Handler, sessionManager *auth.SessionManager) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(sessionManager.LoadAndSave)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create strict middleware that adds request to context
	strictMiddleware := func(f strictnethttp.StrictHTTPHandlerFunc, operationID string) strictnethttp.StrictHTTPHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
			// Add the http.Request to context so handlers can access session
			ctx = handler.WithRequest(ctx, r)
			return f(ctx, w, r, request)
		}
	}

	// Create the strict handler with middleware
	strictHandler := openapi.NewStrictHandlerWithOptions(h, []openapi.StrictMiddlewareFunc{strictMiddleware}, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"code":"BAD_REQUEST","message":"` + err.Error() + `"}}`))
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"` + err.Error() + `"}}`))
		},
	})

	// Mount generated API routes with /api base URL
	openapi.HandlerFromMuxWithBaseURL(strictHandler, r, "/api")

	// Mount SSE routes separately (not generated - requires streaming)
	r.Route("/api", func(r chi.Router) {
		// Public routes (no auth required)
		r.Get("/avatars/{filename}", h.ServeAvatar)
		r.Get("/workspace-icons/{workspaceId}/{filename}", h.ServeWorkspaceIcon)
		r.Get("/emojis/{workspaceId}/{filename}", h.ServeEmoji)

		r.Group(func(r chi.Router) {
			r.Use(authHandler.RequireAuth)
			r.Get("/workspaces/{wid}/events", sseHandler.Events)
			r.Post("/workspaces/{wid}/typing/start", sseHandler.StartTyping)
			r.Post("/workspaces/{wid}/typing/stop", sseHandler.StopTyping)
		})
	})

	return r
}
