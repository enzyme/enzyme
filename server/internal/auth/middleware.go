package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	tokenKey  contextKey = "auth_token"
)

// TokenMiddleware extracts a bearer token, validates it, and sets user ID + token in context.
// Passes through if no token is present (does not reject).
func TokenMiddleware(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token != "" {
				userID, err := store.Validate(token)
				if err == nil && userID != "" {
					ctx := context.WithValue(r.Context(), userIDKey, userID)
					ctx = context.WithValue(ctx, tokenKey, token)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth rejects requests that have no authenticated user in context.
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUserID(r.Context()) == "" {
				writeError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Authentication required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WithUserID returns a context with the given user ID set (for testing).
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetUserID extracts the user ID from context.
func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey).(string)
	return userID
}

// WithToken returns a context with the given auth token set (for testing).
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// GetToken extracts the auth token from context.
func GetToken(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey).(string)
	return token
}

// extractBearerToken checks the Authorization header only.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
