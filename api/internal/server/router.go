package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/enzyme/api/internal/auth"
	"github.com/enzyme/api/internal/handler"
	"github.com/enzyme/api/internal/moderation"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/ratelimit"
	"github.com/enzyme/api/internal/sse"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
)

// NewRouter creates a new HTTP router with all routes registered.
// If spaHandler is non-nil, it is mounted as a fallback for unmatched routes
// to serve the embedded web client.
func NewRouter(h *handler.Handler, sseHandler *sse.Handler, sessionStore *auth.SessionStore, moderationRepo *moderation.Repository, limiter *ratelimit.Limiter, allowedOrigins []string, spaHandler http.Handler) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	if len(allowedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: allowedOrigins,
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			ExposedHeaders: []string{"X-Request-Id"},
			MaxAge:         86400,
		}))
	}

	r.Use(ratelimit.Middleware(limiter))
	r.Use(auth.TokenMiddleware(sessionStore))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create strict middleware that adds request to context
	strictMiddleware := func(f strictnethttp.StrictHTTPHandlerFunc, operationID string) strictnethttp.StrictHTTPHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
			// Add the http.Request to context so handlers can access session
			ctx = handler.WithRequest(ctx, r)
			return f(ctx, w, r, request)
		}
	}

	banCheckMw := BanCheckMiddleware(moderationRepo)

	// Create the strict handler with middleware
	strictHandler := openapi.NewStrictHandlerWithOptions(h, []openapi.StrictMiddlewareFunc{strictMiddleware}, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(openapi.ApiErrorResponse{
				Error: openapi.ApiError{Code: "BAD_REQUEST", Message: err.Error()},
			})
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("unhandled handler error",
				"error", err.Error(),
				"method", r.Method,
				"path", r.URL.Path,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(openapi.ApiErrorResponse{
				Error: openapi.ApiError{Code: "INTERNAL_ERROR", Message: "An internal error occurred"},
			})
		},
	})

	// Mount generated API routes with /api base URL.
	// Ban check is applied as a per-handler middleware (runs after route matching,
	// so chi.URLParam is available).
	openapi.HandlerWithOptions(strictHandler, openapi.ChiServerOptions{
		BaseURL:     "/api",
		BaseRouter:  r,
		Middlewares: []openapi.MiddlewareFunc{banCheckMw},
	})

	// Mount SSE routes separately (not generated - requires streaming)
	r.Route("/api", func(r chi.Router) {
		// Public routes (no auth required)
		r.Get("/avatars/{filename}", h.ServeAvatar)
		r.Get("/workspace-icons/{workspaceId}/{filename}", h.ServeWorkspaceIcon)
		r.Get("/emojis/{workspaceId}/{filename}", h.ServeEmoji)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth())
			r.Use(banCheckMw)
			r.Get("/workspaces/{wid}/events", sseHandler.Events)
			r.Post("/workspaces/{wid}/typing/start", sseHandler.StartTyping)
			r.Post("/workspaces/{wid}/typing/stop", sseHandler.StopTyping)
		})
	})

	// Mount embedded SPA as fallback for all unmatched routes
	if spaHandler != nil {
		r.NotFound(spaHandler.ServeHTTP)
	}

	return r
}

// BanCheckMiddleware rejects workspace-scoped requests from banned users with 403.
func BanCheckMiddleware(moderationRepo *moderation.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wid := chi.URLParam(r, "wid")
			if wid == "" {
				next.ServeHTTP(w, r)
				return
			}

			userID := auth.GetUserID(r.Context())
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			ban, err := moderationRepo.GetActiveBan(r.Context(), wid, userID)
			if err != nil {
				slog.Error("ban check failed", "error", err, "workspace", wid, "user", userID)
				next.ServeHTTP(w, r)
				return
			}
			if ban != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "BANNED",
						"message": "You are banned from this workspace",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
