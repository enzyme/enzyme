package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_Returns429(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — allowed
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Limit") != "1" {
		t.Fatal("expected X-RateLimit-Limit header")
	}

	// Second request — blocked
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestMiddleware_ResponseFormat(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:1234"

	// Exhaust limit
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Now check 429 response body
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json content type, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Fatal("expected X-RateLimit-Reset header")
	}

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Error.Code != "RATE_LIMITED" {
		t.Fatalf("expected code RATE_LIMITED, got %s", body.Error.Code)
	}
	if body.Error.Message == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestMiddleware_PassthroughUnmatchedRoutes(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	called := false
	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/workspaces", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler should have been called for unmatched route")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Limit") != "" {
		t.Fatal("unmatched route should not have rate limit headers")
	}
}

func TestMiddleware_NilLimiter(t *testing.T) {
	called := false
	handler := Middleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler should have been called with nil limiter")
	}
}

func TestMiddleware_StripsPortFromRemoteAddr(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two requests from the same IP but different ports should share a bucket
	req1 := httptest.NewRequest("POST", "/api/auth/login", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req1)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request should be allowed, got %d", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/auth/login", nil)
	req2.RemoteAddr = "10.0.0.1:54321"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req2)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from same IP (different port) should be blocked, got %d", rec.Code)
	}
}

func TestMiddleware_IPv6RemoteAddr(t *testing.T) {
	l := NewLimiter([]Rule{
		{Method: "POST", Path: "/api/auth/login", Limit: 1, Window: time.Minute},
	})

	handler := Middleware(l)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "[::1]:8080"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first IPv6 request should be allowed, got %d", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/auth/login", nil)
	req2.RemoteAddr = "[::1]:9090"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req2)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second IPv6 request (different port) should be blocked, got %d", rec.Code)
	}
}

func TestStripPort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.2.3.4:8080", "1.2.3.4"},
		{"[::1]:8080", "::1"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"1.2.3.4", "1.2.3.4"}, // no port — returned as-is
	}
	for _, tt := range tests {
		got := stripPort(tt.input)
		if got != tt.want {
			t.Errorf("stripPort(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
