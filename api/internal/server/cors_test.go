package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func newTestRouter(allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	if len(allowedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: allowedOrigins,
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			ExposedHeaders: []string{"X-Request-Id"},
			MaxAge:         86400,
		}))
	}

	r.Get("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return r
}

func TestCORS_PreflightAllowedOrigin(t *testing.T) {
	router := newTestRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin http://localhost:3000, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Fatalf("expected Access-Control-Max-Age 86400, got %q", got)
	}
}

func TestCORS_SimpleRequestAllowedOrigin(t *testing.T) {
	router := newTestRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin http://localhost:3000, got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	router := newTestRouter([]string{"http://localhost:3000"})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin, got %q", got)
	}
}

func TestCORS_NoMiddlewareWhenEmpty(t *testing.T) {
	router := newTestRouter(nil)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS headers when origins empty, got %q", got)
	}
}
