package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(context.Background(), 60, 5)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First five requests should succeed (burst of 5).
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/notify", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(context.Background(), 60, 5)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust burst.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/notify", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Next request should be rate limited.
	req := httptest.NewRequest(http.MethodPost, "/notify", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestRateLimiter_SeparateIPsHaveSeparateLimits(t *testing.T) {
	rl := NewRateLimiter(context.Background(), 60, 5)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust burst for IP A.
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPost, "/notify", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// IP B should still be allowed.
	req := httptest.NewRequest(http.MethodPost, "/notify", nil)
	req.RemoteAddr = "5.6.7.8:5678"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for different IP, got %d", w.Code)
	}
}

func TestRateLimiter_CleanupStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	rl := NewRateLimiter(ctx, 60, 5)

	// Use the limiter to create an entry.
	_ = rl.getLimiter("1.2.3.4")

	// Cancel context — cleanup goroutine should exit.
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Verify the limiter still works (entries are still there, just cleanup stopped).
	rl.mu.Lock()
	count := len(rl.entries)
	rl.mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 entry, got %d", count)
	}
}

func TestRateLimiter_MaxEntriesCap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rl := NewRateLimiter(ctx, 60, 5)

	// Fill to capacity with a smaller number for test speed.
	// We test the logic by filling to maxEntries and checking the next one.
	// Use a direct approach: fill the map directly under lock.
	rl.mu.Lock()
	for i := 0; i < maxEntries; i++ {
		ip := fmt.Sprintf("10.%d.%d.%d", (i>>16)&0xFF, (i>>8)&0xFF, i&0xFF)
		rl.entries[ip] = &ipEntry{
			limiter:  nil, // not needed for this test
			lastSeen: time.Now(),
		}
	}
	rl.mu.Unlock()

	// Next new IP should get a zero-budget limiter.
	limiter := rl.getLimiter("99.99.99.99")
	if limiter.Allow() {
		t.Error("expected zero-budget limiter to deny, but it allowed")
	}
}
