package sse

import (
	"testing"
	"time"

	"github.com/feather/api/internal/testutil"
)

func TestDeleteOldEvents(t *testing.T) {
	db := testutil.TestDB(t)

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	hub := NewHub(db, 1*time.Hour, 0)

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
	hub.deleteOldEvents()

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
	hub := NewHub(nil, 1*time.Hour, 0)
	// Should not panic
	hub.deleteOldEvents()
}

func TestDeleteOldEventsZeroRetention(t *testing.T) {
	db := testutil.TestDB(t)

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	hub := NewHub(db, 0, 0)

	// Insert an event
	_, err := db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, "evt-1", ws.ID, "message_created", `{"text":"hello"}`, time.Now().UTC().Add(-2*time.Hour).Format(time.RFC3339))
	if err != nil {
		t.Fatalf("inserting event: %v", err)
	}

	// Should be a no-op with zero retention
	hub.deleteOldEvents()

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM workspace_events`).Scan(&count)
	if err != nil {
		t.Fatalf("counting events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event remaining (no-op), got %d", count)
	}
}
