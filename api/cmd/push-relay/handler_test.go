package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockFCM implements Dispatcher for testing.
type mockFCM struct {
	status string
	err    error
}

func (m *mockFCM) Send(_ context.Context, _ *NotifyRequest) (string, error) {
	return m.status, m.err
}

// mockAPNs implements Dispatcher for testing.
type mockAPNs struct {
	status string
	err    error
}

func (m *mockAPNs) Send(_ context.Context, _ *NotifyRequest) (string, error) {
	return m.status, m.err
}

// setupTestRouter creates a real router with mock dispatchers and a permissive rate limiter.
func setupTestRouter(fcm, apns Dispatcher) http.Handler {
	rl := NewRateLimiter(context.Background(), 100000, 100000)
	return newRouter(fcm, apns, rl)
}

func TestHandler_ValidFCMRequest(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, nil)
	body := `{"device_token":"tok123","platform":"fcm","title":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp NotifyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "sent" {
		t.Errorf("expected status \"sent\", got %q", resp.Status)
	}
}

func TestHandler_ValidAPNsRequest(t *testing.T) {
	handler := setupTestRouter(nil, &mockAPNs{status: "sent"})
	body := `{"device_token":"tok123","platform":"apns","title":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp NotifyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "sent" {
		t.Errorf("expected status \"sent\", got %q", resp.Status)
	}
}

func TestHandler_InvalidToken(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "invalid_token"}, nil)
	body := `{"device_token":"bad-tok","platform":"fcm","title":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp NotifyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "invalid_token" {
		t.Errorf("expected status \"invalid_token\", got %q", resp.Status)
	}
}

func TestHandler_DispatchError(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "error", err: fmt.Errorf("connection refused")}, nil)
	body := `{"device_token":"tok123","platform":"fcm","title":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}

	var resp NotifyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("expected status \"error\", got %q", resp.Status)
	}
	if resp.Error != "dispatch failed" {
		t.Errorf("expected generic error \"dispatch failed\", got %q", resp.Error)
	}
}

func TestHandler_MissingFields(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, nil)
	body := `{"platform":"fcm"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_OversizedBody(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, nil)
	// Create a body larger than 4 KB.
	largeBody := `{"device_token":"tok","platform":"fcm","title":"Hello","body":"` + strings.Repeat("x", 5000) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(largeBody))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_FCMNotConfigured(t *testing.T) {
	handler := setupTestRouter(nil, &mockAPNs{status: "sent"})
	body := `{"device_token":"tok123","platform":"fcm","title":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandler_APNsNotConfigured(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, nil)
	body := `{"device_token":"tok123","platform":"apns","title":"Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHealth_BothConfigured(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, &mockAPNs{status: "sent"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status \"ok\", got %q", resp.Status)
	}
	if !resp.FCM {
		t.Error("expected FCM true")
	}
	if !resp.APNs {
		t.Error("expected APNs true")
	}
}

func TestHealth_OnlyFCM(t *testing.T) {
	handler := setupTestRouter(&mockFCM{status: "sent"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status \"ok\", got %q", resp.Status)
	}
	if !resp.FCM {
		t.Error("expected FCM true")
	}
	if resp.APNs {
		t.Error("expected APNs false")
	}
}
