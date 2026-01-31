package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := h.sessionManager.GetUserID(r)
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}

// OptionalAuth adds user ID to context if authenticated, but doesn't require it
func (h *Handler) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := h.sessionManager.GetUserID(r)
		if userID != "" {
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
