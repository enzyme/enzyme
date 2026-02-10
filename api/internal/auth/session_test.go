package auth

import (
	"testing"
	"time"

	"github.com/feather/api/internal/testutil"
)

func TestSessionStore_CreateAndValidate(t *testing.T) {
	db := testutil.TestDB(t)
	store := NewSessionStore(db, 24*time.Hour)

	token, err := store.Create("user-123")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	userID, err := store.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("expected user-123, got %s", userID)
	}
}

func TestSessionStore_ValidateExpired(t *testing.T) {
	db := testutil.TestDB(t)
	store := NewSessionStore(db, -1*time.Hour) // already expired

	token, err := store.Create("user-123")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = store.Validate(token)
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	db := testutil.TestDB(t)
	store := NewSessionStore(db, 24*time.Hour)

	token, err := store.Create("user-123")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Delete(token); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Validate(token)
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestSessionStore_DeleteExpired(t *testing.T) {
	db := testutil.TestDB(t)
	store := NewSessionStore(db, -1*time.Hour) // already expired

	_, err := store.Create("user-123")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.DeleteExpired(); err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}

	// Verify it was cleaned up by checking count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 sessions, got %d", count)
	}
}

func TestSessionStore_ValidateNonexistent(t *testing.T) {
	db := testutil.TestDB(t)
	store := NewSessionStore(db, 24*time.Hour)

	_, err := store.Validate("nonexistent-token")
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
