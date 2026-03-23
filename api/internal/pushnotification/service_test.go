package pushnotification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/enzyme/api/internal/testutil"
)

func TestSendWithMockRelay(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	// Register two device tokens
	for _, tok := range []string{"token-1", "token-2"} {
		if err := repo.Upsert(ctx, &DeviceToken{
			UserID: user.ID, Token: tok, Platform: "fcm", DeviceID: "device-1",
		}); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	var mu sync.Mutex
	var receivedRequests []RelayRequest

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RelayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}
		mu.Lock()
		receivedRequests = append(receivedRequests, req)
		mu.Unlock()

		json.NewEncoder(w).Encode(RelayResponse{Status: "sent"})
	}))
	defer relay.Close()

	svc := NewService(repo, relay.URL)
	data := NotificationData{
		Title:       "@alice in #general",
		Body:        "Hello world",
		ChannelID:   "ch-1",
		MessageID:   "msg-1",
		WorkspaceID: "ws-1",
		ServerURL:   "https://chat.example.com",
	}

	ok := svc.Send(ctx, user.ID, data)
	if !ok {
		t.Fatal("expected Send to return true")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(receivedRequests) != 2 {
		t.Fatalf("expected 2 relay requests, got %d", len(receivedRequests))
	}

	// Verify payload structure
	req := receivedRequests[0]
	if req.Title != "@alice in #general" {
		t.Errorf("expected title '@alice in #general', got %q", req.Title)
	}
	if req.Data.ChannelID != "ch-1" {
		t.Errorf("expected channel_id 'ch-1', got %q", req.Data.ChannelID)
	}
	if req.Data.ServerURL != "https://chat.example.com" {
		t.Errorf("expected server_url 'https://chat.example.com', got %q", req.Data.ServerURL)
	}
}

func TestSendInvalidTokenCleanup(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "bad-token", Platform: "apns", DeviceID: "device-1",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(RelayResponse{Status: "invalid_token"})
	}))
	defer relay.Close()

	svc := NewService(repo, relay.URL)
	ok := svc.Send(ctx, user.ID, NotificationData{Title: "test", Body: "test"})
	if ok {
		t.Fatal("expected Send to return false when token is invalid")
	}

	// Verify token was deleted
	tokens, _ := repo.ListByUserID(ctx, user.ID)
	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens after invalid_token cleanup, got %d", len(tokens))
	}
}

func TestSendNoTokens(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	relayHit := false
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relayHit = true
		json.NewEncoder(w).Encode(RelayResponse{Status: "sent"})
	}))
	defer relay.Close()

	svc := NewService(repo, relay.URL)
	ok := svc.Send(ctx, user.ID, NotificationData{Title: "test", Body: "test"})
	if ok {
		t.Fatal("expected Send to return false when no tokens exist")
	}
	if relayHit {
		t.Fatal("expected relay to not be called when no tokens exist")
	}
}

func TestSendRelayError(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "token-1", Platform: "fcm", DeviceID: "device-1",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer relay.Close()

	svc := NewService(repo, relay.URL)
	ok := svc.Send(ctx, user.ID, NotificationData{Title: "test", Body: "test"})
	if ok {
		t.Fatal("expected Send to return false on relay error")
	}

	// Token should still exist (not deleted on relay error)
	tokens, _ := repo.ListByUserID(ctx, user.ID)
	if len(tokens) != 1 {
		t.Fatalf("expected token to still exist, got %d tokens", len(tokens))
	}
}
