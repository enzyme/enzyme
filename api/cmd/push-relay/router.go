package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newRouter(fcm, apns Dispatcher, rateLimiter *RateLimiter, trustProxy bool) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	if trustProxy {
		r.Use(middleware.RealIP)
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{
			Status: "ok",
			FCM:    fcm != nil,
			APNs:   apns != nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	r.With(rateLimiter.Middleware).Post("/notify", (&notifyHandler{fcm: fcm, apns: apns}).ServeHTTP)

	return r
}
