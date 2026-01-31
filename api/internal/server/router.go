package server

import (
	"net/http"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/file"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/workspace"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handlers struct {
	Auth      *auth.Handler
	Workspace *workspace.Handler
	Channel   *channel.Handler
	Message   *message.Handler
	SSE       *sse.Handler
	File      *file.Handler
}

func NewRouter(handlers Handlers, sessionManager *auth.SessionManager) http.Handler {
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

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes (no auth required)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", handlers.Auth.Register)
			r.Post("/login", handlers.Auth.Login)
			r.Post("/logout", handlers.Auth.Logout)
			r.Post("/forgot-password", handlers.Auth.ForgotPassword)
			r.Post("/reset-password", handlers.Auth.ResetPassword)
			r.Get("/me", handlers.Auth.Me)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(handlers.Auth.RequireAuth)

			// Workspaces
			r.Post("/workspaces/create", handlers.Workspace.Create)
			r.Route("/workspaces/{wid}", func(r chi.Router) {
				r.Post("/update", handlers.Workspace.Update)
				r.Get("/", handlers.Workspace.Get)
				r.Post("/members/list", handlers.Workspace.ListMembers)
				r.Post("/members/remove", handlers.Workspace.RemoveMember)
				r.Post("/members/update-role", handlers.Workspace.UpdateMemberRole)
				r.Post("/invites/create", handlers.Workspace.CreateInvite)

				// Channels within workspace
				r.Post("/channels/create", handlers.Channel.Create)
				r.Post("/channels/list", handlers.Channel.List)
				r.Post("/channels/dm", handlers.Channel.CreateDM)

				// SSE events
				r.Get("/events", handlers.SSE.Events)
				r.Post("/typing/start", handlers.SSE.StartTyping)
				r.Post("/typing/stop", handlers.SSE.StopTyping)
			})

			// Invites
			r.Post("/invites/{code}/accept", handlers.Workspace.AcceptInvite)

			// Channels
			r.Route("/channels/{id}", func(r chi.Router) {
				r.Post("/update", handlers.Channel.Update)
				r.Post("/archive", handlers.Channel.Archive)
				r.Post("/members/add", handlers.Channel.AddMember)
				r.Post("/members/list", handlers.Channel.ListMembers)
				r.Post("/join", handlers.Channel.Join)
				r.Post("/leave", handlers.Channel.Leave)

				// Messages
				r.Post("/messages/send", handlers.Message.Send)
				r.Post("/messages/list", handlers.Message.List)

				// File uploads
				r.Post("/files/upload", handlers.File.Upload)
			})

			// Messages
			r.Route("/messages/{id}", func(r chi.Router) {
				r.Post("/update", handlers.Message.Update)
				r.Post("/delete", handlers.Message.Delete)
				r.Post("/reactions/add", handlers.Message.AddReaction)
				r.Post("/reactions/remove", handlers.Message.RemoveReaction)
				r.Post("/thread/list", handlers.Message.ListThread)
			})

			// Files
			r.Get("/files/{id}/download", handlers.File.Download)
			r.Post("/files/{id}/delete", handlers.File.Delete)
		})
	})

	return r
}
