package sse

import (
	"context"
	"testing"
	"time"

	"github.com/enzyme/server/internal/testutil"
	"github.com/oklog/ulid/v2"
)

func TestDeleteOldEvents(t *testing.T) {
	db := testutil.TestDB(t)

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	hub := NewHub(db, 1*time.Hour)

	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour).Format(time.RFC3339)
	recentTime := now.Add(-30 * time.Minute).Format(time.RFC3339)

	// Insert an old event (2 hours ago, beyond 1h retention)
	_, err := db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "evt-old", ws.ID, "message_created", `{"text":"old"}`, oldTime)
	if err != nil {
		t.Fatalf("inserting old event: %v", err)
	}

	// Insert a recent event (30 min ago, within 1h retention)
	_, err = db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "evt-recent", ws.ID, "message_created", `{"text":"recent"}`, recentTime)
	if err != nil {
		t.Fatalf("inserting recent event: %v", err)
	}

	// Run cleanup
	hub.CleanupOldEvents(context.Background())

	// Verify old event was deleted and recent one remains
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM workspace_events`).Scan(&count)
	if err != nil {
		t.Fatalf("counting events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event remaining, got %d", count)
	}

	var id string
	err = db.QueryRow(`SELECT id FROM workspace_events`).Scan(&id)
	if err != nil {
		t.Fatalf("querying remaining event: %v", err)
	}
	if id != "evt-recent" {
		t.Fatalf("expected remaining event to be evt-recent, got %s", id)
	}
}

func TestDeleteOldEventsNilDB(t *testing.T) {
	hub := NewHub(nil, 1*time.Hour)
	// Should not panic
	hub.CleanupOldEvents(context.Background())
}

func TestDeleteOldEventsZeroRetention(t *testing.T) {
	db := testutil.TestDB(t)

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	hub := NewHub(db, 0)

	// Insert an event
	_, err := db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "evt-1", ws.ID, "message_created", `{"text":"hello"}`, time.Now().UTC().Add(-2*time.Hour).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("inserting event: %v", err)
	}

	// Should be a no-op with zero retention
	hub.CleanupOldEvents(context.Background())

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM workspace_events`).Scan(&count)
	if err != nil {
		t.Fatalf("counting events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event remaining (no-op), got %d", count)
	}
}

func TestGetEventsSinceFiltersPrivateChannelEvents(t *testing.T) {
	db := testutil.TestDB(t)

	alice := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	bob := testutil.CreateTestUser(t, db, "bob@example.com", "Bob")
	ws := testutil.CreateTestWorkspace(t, db, alice.ID, "Test Workspace")

	// Alice creates a public channel and a private channel.
	// CreateTestChannel adds the creator as a member automatically.
	publicCh := testutil.CreateTestChannel(t, db, ws.ID, alice.ID, "general", "public")
	privateCh := testutil.CreateTestChannel(t, db, ws.ID, alice.ID, "secret", "private")

	// Add Bob to the public channel only
	membershipID := ulid.Make().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO channel_memberships (id, user_id, channel_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, membershipID, bob.ID, publicCh.ID, now, now)
	if err != nil {
		t.Fatalf("adding bob to public channel: %v", err)
	}

	hub := NewHub(db, 1*time.Hour)

	// Insert events: workspace-scoped, public channel, and private channel
	// Use ULIDs to ensure ordering: evt-1 < evt-2 < evt-3
	insertEvent := func(id, channelID, payload string) {
		var chID *string
		if channelID != "" {
			chID = &channelID
		}
		_, err := db.Exec(`
			INSERT INTO workspace_events (id, workspace_id, event_type, payload, channel_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, id, ws.ID, "message.new", payload, chID, now)
		if err != nil {
			t.Fatalf("inserting event %s: %v", id, err)
		}
	}

	insertEvent("01-workspace", "", `{"text":"workspace event"}`)
	insertEvent("02-public", publicCh.ID, `{"text":"public channel msg"}`)
	insertEvent("03-private", privateCh.ID, `{"text":"private channel msg"}`)

	// Alice (member of both channels) should see all 3 events
	aliceEvents, err := hub.GetEventsSince(ws.ID, "00", []string{publicCh.ID, privateCh.ID})
	if err != nil {
		t.Fatalf("GetEventsSince for alice: %v", err)
	}
	if len(aliceEvents) != 3 {
		t.Fatalf("alice: expected 3 events, got %d", len(aliceEvents))
	}

	// Bob (member of public channel only) should see workspace + public events, NOT private
	bobEvents, err := hub.GetEventsSince(ws.ID, "00", []string{publicCh.ID})
	if err != nil {
		t.Fatalf("GetEventsSince for bob: %v", err)
	}
	if len(bobEvents) != 2 {
		t.Fatalf("bob: expected 2 events, got %d", len(bobEvents))
	}
	if bobEvents[0].ID != "01-workspace" {
		t.Fatalf("bob: expected first event to be workspace event, got %s", bobEvents[0].ID)
	}
	if bobEvents[1].ID != "02-public" {
		t.Fatalf("bob: expected second event to be public channel event, got %s", bobEvents[1].ID)
	}
}

func TestGetEventsSinceZeroChannelUser(t *testing.T) {
	db := testutil.TestDB(t)

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")

	hub := NewHub(db, 1*time.Hour)

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert a workspace-scoped event and a channel-scoped event
	_, err := db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, channel_id, created_at)
		VALUES (?, ?, ?, ?, NULL, ?),
		       (?, ?, ?, ?, ?, ?)
	`, "01-ws", ws.ID, "presence.changed", `{"user_id":"x"}`, now,
		"02-ch", ws.ID, "message.new", `{"text":"hello"}`, ch.ID, now)
	if err != nil {
		t.Fatalf("inserting events: %v", err)
	}

	// User with no channel memberships should only see workspace-scoped events
	events, err := hub.GetEventsSince(ws.ID, "00", nil)
	if err != nil {
		t.Fatalf("GetEventsSince: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "01-ws" {
		t.Fatalf("expected workspace event, got %s", events[0].ID)
	}
}

func TestStoreEventChannelID(t *testing.T) {
	db := testutil.TestDB(t)

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")

	hub := NewHub(db, 1*time.Hour)

	// Store a workspace-scoped event (no channel_id)
	hub.storeEvent(ws.ID, "", Event{ID: "evt-ws", Type: "presence.changed", Data: map[string]string{"user_id": user.ID}})

	// Store a channel-scoped event (with channel_id)
	hub.storeEvent(ws.ID, ch.ID, Event{ID: "evt-ch", Type: "message.new", Data: map[string]string{"text": "hello"}})

	// Verify workspace event has NULL channel_id
	var chID *string
	err := db.QueryRow(`SELECT channel_id FROM workspace_events WHERE id = ?`, "evt-ws").Scan(&chID)
	if err != nil {
		t.Fatalf("querying workspace event: %v", err)
	}
	if chID != nil {
		t.Fatalf("workspace event should have NULL channel_id, got %v", *chID)
	}

	// Verify channel event has the correct channel_id
	err = db.QueryRow(`SELECT channel_id FROM workspace_events WHERE id = ?`, "evt-ch").Scan(&chID)
	if err != nil {
		t.Fatalf("querying channel event: %v", err)
	}
	if chID == nil {
		t.Fatal("channel event should have non-NULL channel_id")
	}
	if *chID != ch.ID {
		t.Fatalf("channel event channel_id = %s, want %s", *chID, ch.ID)
	}
}
