package thread

import (
	"context"
	"testing"

	"github.com/enzyme/server/internal/testutil"
)

func TestRepository_Subscribe_NewSubscription(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "thread parent")

	sub, err := repo.Subscribe(ctx, msg.ID, user.ID)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sub.ThreadParentID != msg.ID {
		t.Errorf("ThreadParentID = %q, want %q", sub.ThreadParentID, msg.ID)
	}
	if sub.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", sub.UserID, user.ID)
	}
	if sub.Status != "subscribed" {
		t.Errorf("Status = %q, want %q", sub.Status, "subscribed")
	}
	if sub.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestRepository_Subscribe_ResubscribeAfterUnsubscribe(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "thread parent")

	// Subscribe, then unsubscribe, then re-subscribe
	_, err := repo.Subscribe(ctx, msg.ID, user.ID)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	unsub, err := repo.Unsubscribe(ctx, msg.ID, user.ID)
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
	if unsub.Status != "unsubscribed" {
		t.Errorf("Status = %q, want %q", unsub.Status, "unsubscribed")
	}

	resub, err := repo.Subscribe(ctx, msg.ID, user.ID)
	if err != nil {
		t.Fatalf("re-Subscribe() error = %v", err)
	}
	if resub.Status != "subscribed" {
		t.Errorf("Status = %q, want %q", resub.Status, "subscribed")
	}
	// ID should be preserved from the original insert (ON CONFLICT updates, not replaces)
	if resub.ID != unsub.ID {
		t.Errorf("ID changed after re-subscribe: %q != %q", resub.ID, unsub.ID)
	}
}

func TestRepository_Unsubscribe_NewRow(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "thread parent")

	sub, err := repo.Unsubscribe(ctx, msg.ID, user.ID)
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
	if sub.Status != "unsubscribed" {
		t.Errorf("Status = %q, want %q", sub.Status, "unsubscribed")
	}
	if sub.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestRepository_GetSubscription_ReturnsNilForMissing(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	sub, err := repo.GetSubscription(ctx, "nonexistent", "nonexistent")
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if sub != nil {
		t.Error("expected nil subscription for missing row")
	}
}
