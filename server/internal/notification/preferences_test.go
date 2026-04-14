package notification

import (
	"context"
	"testing"

	"github.com/enzyme/server/internal/testutil"
)

func TestPreferencesRepository_Upsert_Insert(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")

	pref := &NotificationPreference{
		UserID:       user.ID,
		ChannelID:    ch.ID,
		NotifyLevel:  NotifyAll,
		EmailEnabled: true,
	}

	err := repo.Upsert(ctx, pref)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	if pref.ID == "" {
		t.Error("expected non-empty ID after insert")
	}
	if pref.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if pref.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}

	// Verify via Get
	got, err := repo.Get(ctx, user.ID, ch.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.NotifyLevel != NotifyAll {
		t.Errorf("NotifyLevel = %q, want %q", got.NotifyLevel, NotifyAll)
	}
}

func TestPreferencesRepository_Upsert_Update(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")

	// Insert
	pref := &NotificationPreference{
		UserID:       user.ID,
		ChannelID:    ch.ID,
		NotifyLevel:  NotifyAll,
		EmailEnabled: true,
	}
	if err := repo.Upsert(ctx, pref); err != nil {
		t.Fatalf("initial Upsert() error = %v", err)
	}
	originalID := pref.ID
	originalCreatedAt := pref.CreatedAt

	// Update via second upsert
	pref2 := &NotificationPreference{
		UserID:       user.ID,
		ChannelID:    ch.ID,
		NotifyLevel:  NotifyMentions,
		EmailEnabled: false,
	}
	if err := repo.Upsert(ctx, pref2); err != nil {
		t.Fatalf("update Upsert() error = %v", err)
	}

	// ID should be preserved (ON CONFLICT updates existing row)
	if pref2.ID != originalID {
		t.Errorf("ID changed: %q != %q", pref2.ID, originalID)
	}
	// CreatedAt should be preserved
	if !pref2.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt changed: %v != %v", pref2.CreatedAt, originalCreatedAt)
	}

	// Verify the update took effect
	got, err := repo.Get(ctx, user.ID, ch.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.NotifyLevel != NotifyMentions {
		t.Errorf("NotifyLevel = %q, want %q", got.NotifyLevel, NotifyMentions)
	}
	if got.EmailEnabled != false {
		t.Error("expected EmailEnabled = false")
	}
}

func TestPreferencesRepository_Upsert_PopulatesAllFields(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewPreferencesRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")

	pref := &NotificationPreference{
		UserID:       user.ID,
		ChannelID:    ch.ID,
		NotifyLevel:  NotifyNone,
		EmailEnabled: false,
	}

	if err := repo.Upsert(ctx, pref); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	// All fields should be populated from RETURNING
	if pref.ID == "" {
		t.Error("expected ID to be populated")
	}
	if pref.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", pref.UserID, user.ID)
	}
	if pref.ChannelID != ch.ID {
		t.Errorf("ChannelID = %q, want %q", pref.ChannelID, ch.ID)
	}
	if pref.NotifyLevel != NotifyNone {
		t.Errorf("NotifyLevel = %q, want %q", pref.NotifyLevel, NotifyNone)
	}
}
