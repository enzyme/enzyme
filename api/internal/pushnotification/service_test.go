package pushnotification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

	var receivedRequests []RelayRequest

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RelayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}
		receivedRequests = append(receivedRequests, req)

		json.NewEncoder(w).Encode(RelayResponse{Status: "sent"})
	}))
	defer relay.Close()

	svc := NewService(repo, relay.URL)
	data := NotificationData{
		Title:          "@alice in #general",
		Body:           "Hello world",
		ChannelID:      "ch-1",
		MessageID:      "msg-1",
		WorkspaceID:    "ws-1",
		ChannelName:    "general",
		ThreadParentID: "msg-parent-1",
		ServerURL:      "https://chat.example.com",
	}

	ok := svc.Send(ctx, user.ID, data)
	if !ok {
		t.Fatal("expected Send to return true")
	}

	if len(receivedRequests) != 2 {
		t.Fatalf("expected 2 relay requests, got %d", len(receivedRequests))
	}

	for _, req := range receivedRequests {
		if req.Title != "@alice in #general" {
			t.Errorf("expected title '@alice in #general', got %q", req.Title)
		}
		if req.Data.ChannelID != "ch-1" {
			t.Errorf("expected channel_id 'ch-1', got %q", req.Data.ChannelID)
		}
		if req.Data.ChannelName != "general" {
			t.Errorf("expected channel_name 'general', got %q", req.Data.ChannelName)
		}
		if req.Data.ThreadParentID != "msg-parent-1" {
			t.Errorf("expected thread_parent_id 'msg-parent-1', got %q", req.Data.ThreadParentID)
		}
		if req.Data.ServerURL != "https://chat.example.com" {
			t.Errorf("expected server_url 'https://chat.example.com', got %q", req.Data.ServerURL)
		}
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

func TestSendOmitsEmptyOptionalFields(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := context.Background()

	if err := repo.Upsert(ctx, &DeviceToken{
		UserID: user.ID, Token: "token-1", Platform: "fcm", DeviceID: "device-1",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var rawBody []byte

	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		rawBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		json.NewEncoder(w).Encode(RelayResponse{Status: "sent"})
	}))
	defer relay.Close()

	svc := NewService(repo, relay.URL)
	// Send with empty ThreadParentID and ChannelName — both should be omitted from JSON.
	data := NotificationData{
		Title:       "@alice in #general",
		Body:        "Hello",
		ChannelID:   "ch-1",
		MessageID:   "msg-1",
		WorkspaceID: "ws-1",
		ServerURL:   "https://chat.example.com",
	}

	ok := svc.Send(ctx, user.ID, data)
	if !ok {
		t.Fatal("expected Send to return true")
	}

	bodyStr := string(rawBody)
	if strings.Contains(bodyStr, "thread_parent_id") {
		t.Errorf("expected thread_parent_id to be omitted from JSON, got: %s", bodyStr)
	}
	if strings.Contains(bodyStr, "channel_name") {
		t.Errorf("expected channel_name to be omitted from JSON, got: %s", bodyStr)
	}
}
