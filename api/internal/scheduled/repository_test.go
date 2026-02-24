package scheduled

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/testutil"
)

func setupTest(t *testing.T) (*Repository, *testutil.TestUser, *testutil.TestWorkspace, *testutil.TestChannel) {
	t.Helper()
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	return repo, user, ws, ch
}

func TestRepository_Create(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Hello later!",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	}

	err := repo.Create(ctx, msg)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if msg.ID == "" {
		t.Error("expected non-empty ID")
	}
	if msg.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if msg.Status != StatusPending {
		t.Errorf("Status = %q, want %q", msg.Status, StatusPending)
	}
}

func TestRepository_Create_WithAttachments(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:     ch.ID,
		UserID:        user.ID,
		Content:       "With files",
		AttachmentIDs: []string{"file1", "file2"},
		ScheduledFor:  time.Now().Add(1 * time.Hour),
	}

	err := repo.Create(ctx, msg)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(got.AttachmentIDs) != 2 {
		t.Errorf("AttachmentIDs length = %d, want 2", len(got.AttachmentIDs))
	}
}

func TestRepository_Create_WithThreadParent(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	parent := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Parent message")

	parentID := parent.ID
	msg := &ScheduledMessage{
		ChannelID:      ch.ID,
		UserID:         user.ID,
		Content:        "Thread reply",
		ThreadParentID: &parentID,
		ScheduledFor:   time.Now().Add(1 * time.Hour),
	}

	err := repo.Create(ctx, msg)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ThreadParentID == nil || *got.ThreadParentID != parentID {
		t.Errorf("ThreadParentID = %v, want %q", got.ThreadParentID, parentID)
	}
}

func TestRepository_GetByID(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Scheduled!",
		ScheduledFor: time.Now().Add(2 * time.Hour),
	}
	repo.Create(ctx, msg)

	got, err := repo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Content != "Scheduled!" {
		t.Errorf("Content = %q, want %q", got.Content, "Scheduled!")
	}
	if got.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", got.UserID, user.ID)
	}
	if got.ChannelID != ch.ID {
		t.Errorf("ChannelID = %q, want %q", got.ChannelID, ch.ID)
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}
	if got.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", got.RetryCount)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo, _, _, _ := setupTest(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if !errors.Is(err, ErrScheduledMessageNotFound) {
		t.Errorf("GetByID() error = %v, want ErrScheduledMessageNotFound", err)
	}
}

func TestRepository_Update(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Original",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	}
	repo.Create(ctx, msg)

	msg.Content = "Updated content"
	newTime := time.Now().Add(3 * time.Hour)
	msg.ScheduledFor = newTime

	err := repo.Update(ctx, msg)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Content != "Updated content" {
		t.Errorf("Content = %q, want %q", got.Content, "Updated content")
	}
}

func TestRepository_Delete(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "To delete",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	}
	repo.Create(ctx, msg)

	err := repo.Delete(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.GetByID(ctx, msg.ID)
	if !errors.Is(err, ErrScheduledMessageNotFound) {
		t.Errorf("GetByID after delete error = %v, want ErrScheduledMessageNotFound", err)
	}
}

func TestRepository_ListByUser(t *testing.T) {
	repo, user, ws, ch := setupTest(t)
	ctx := context.Background()

	// Create two scheduled messages
	for _, content := range []string{"First", "Second"} {
		repo.Create(ctx, &ScheduledMessage{
			ChannelID:    ch.ID,
			UserID:       user.ID,
			Content:      content,
			ScheduledFor: time.Now().Add(1 * time.Hour),
		})
	}

	messages, err := repo.ListByUser(ctx, user.ID, ws.ID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("ListByUser() returned %d messages, want 2", len(messages))
	}

	// Should include channel info
	if messages[0].ChannelName != "general" {
		t.Errorf("ChannelName = %q, want %q", messages[0].ChannelName, "general")
	}
	if messages[0].WorkspaceID != ws.ID {
		t.Errorf("WorkspaceID = %q, want %q", messages[0].WorkspaceID, ws.ID)
	}
}

func TestRepository_ListByUser_IncludesFailedMessages(t *testing.T) {
	repo, user, ws, ch := setupTest(t)
	ctx := context.Background()

	// Create a pending message
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Pending",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})

	// Create a message and mark it as failed
	failed := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Failed",
		ScheduledFor: time.Now().Add(-1 * time.Hour),
	}
	repo.Create(ctx, failed)
	repo.MarkFailed(ctx, failed.ID, "channel is archived")

	messages, err := repo.ListByUser(ctx, user.ID, ws.ID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("ListByUser() returned %d messages, want 2 (including failed)", len(messages))
	}
}

func TestRepository_ListByUser_OrderedByScheduledFor(t *testing.T) {
	repo, user, ws, ch := setupTest(t)
	ctx := context.Background()

	// Create messages with different scheduled times (second one earlier)
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Later",
		ScheduledFor: time.Now().Add(2 * time.Hour),
	})
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Earlier",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})

	messages, _ := repo.ListByUser(ctx, user.ID, ws.ID)
	if messages[0].Content != "Earlier" {
		t.Errorf("first message Content = %q, want %q", messages[0].Content, "Earlier")
	}
}

func TestRepository_ListByUser_FiltersOtherUsers(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	alice := testutil.CreateTestUser(t, db, "alice@example.com", "Alice")
	bob := testutil.CreateTestUser(t, db, "bob@example.com", "Bob")
	ws := testutil.CreateTestWorkspace(t, db, alice.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, alice.ID, "general", channel.TypePublic)

	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       alice.ID,
		Content:      "Alice's message",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       bob.ID,
		Content:      "Bob's message",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})

	messages, _ := repo.ListByUser(ctx, alice.ID, ws.ID)
	if len(messages) != 1 {
		t.Fatalf("ListByUser() returned %d messages, want 1", len(messages))
	}
	if messages[0].Content != "Alice's message" {
		t.Errorf("Content = %q, want %q", messages[0].Content, "Alice's message")
	}
}

func TestRepository_ListDue(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	// Create one due message and one future message
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Due now",
		ScheduledFor: time.Now().Add(-10 * time.Minute),
	})
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Future",
		ScheduledFor: time.Now().Add(1 * time.Hour),
	})

	messages, err := repo.ListDue(ctx)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("ListDue() returned %d messages, want 1", len(messages))
	}
	if messages[0].Content != "Due now" {
		t.Errorf("Content = %q, want %q", messages[0].Content, "Due now")
	}
}

func TestRepository_ListDue_OnlyPendingMessages(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	// Create a due pending message
	repo.Create(ctx, &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Due pending",
		ScheduledFor: time.Now().Add(-10 * time.Minute),
	})

	// Create a due message that's already sending
	sending := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Due sending",
		ScheduledFor: time.Now().Add(-5 * time.Minute),
	}
	repo.Create(ctx, sending)
	repo.MarkSending(ctx, sending.ID)

	// Create a due message that failed
	failed := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Due failed",
		ScheduledFor: time.Now().Add(-5 * time.Minute),
	}
	repo.Create(ctx, failed)
	repo.MarkFailed(ctx, failed.ID, "some error")

	messages, err := repo.ListDue(ctx)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("ListDue() returned %d messages, want 1 (only pending)", len(messages))
	}
	if messages[0].Content != "Due pending" {
		t.Errorf("Content = %q, want %q", messages[0].Content, "Due pending")
	}
}

func TestRepository_CountByWorkspace(t *testing.T) {
	repo, user, ws, ch := setupTest(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		repo.Create(ctx, &ScheduledMessage{
			ChannelID:    ch.ID,
			UserID:       user.ID,
			Content:      "msg",
			ScheduledFor: time.Now().Add(1 * time.Hour),
		})
	}

	count, err := repo.CountByWorkspace(ctx, user.ID, ws.ID)
	if err != nil {
		t.Fatalf("CountByWorkspace() error = %v", err)
	}
	if count != 3 {
		t.Errorf("CountByWorkspace() = %d, want 3", count)
	}
}

func TestRepository_MarkSending(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Test",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)

	// First claim should succeed
	claimed, err := repo.MarkSending(ctx, msg.ID)
	if err != nil {
		t.Fatalf("MarkSending() error = %v", err)
	}
	if !claimed {
		t.Error("MarkSending() first call should return true")
	}

	// Verify status
	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusSending {
		t.Errorf("Status = %q, want %q", got.Status, StatusSending)
	}

	// Second claim should fail (already sending)
	claimed2, err := repo.MarkSending(ctx, msg.ID)
	if err != nil {
		t.Fatalf("MarkSending() second call error = %v", err)
	}
	if claimed2 {
		t.Error("MarkSending() second call should return false (already claimed)")
	}
}

func TestRepository_MarkFailed(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Will fail",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)

	err := repo.MarkFailed(ctx, msg.ID, "channel is archived")
	if err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", got.Status, StatusFailed)
	}
	if got.LastError != "channel is archived" {
		t.Errorf("LastError = %q, want %q", got.LastError, "channel is archived")
	}
}

func TestRepository_IncrementRetry(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Retry me",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)

	// Mark sending first, then increment retry
	repo.MarkSending(ctx, msg.ID)
	err := repo.IncrementRetry(ctx, msg.ID, "transient error")
	if err != nil {
		t.Fatalf("IncrementRetry() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q (reset to pending)", got.Status, StatusPending)
	}
	if got.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", got.RetryCount)
	}
	if got.LastError != "transient error" {
		t.Errorf("LastError = %q, want %q", got.LastError, "transient error")
	}

	// Increment again
	repo.MarkSending(ctx, msg.ID)
	repo.IncrementRetry(ctx, msg.ID, "another error")
	got, _ = repo.GetByID(ctx, msg.ID)
	if got.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", got.RetryCount)
	}
}

func TestRepository_ResetStuckSending(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	// Create a message and mark it as sending
	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Stuck",
		ScheduledFor: time.Now().Add(-10 * time.Minute),
	}
	repo.Create(ctx, msg)
	repo.MarkSending(ctx, msg.ID)

	// Wait so that updated_at is strictly in the past relative to the threshold.
	// RFC3339 has second precision, so we need a full 2s to guarantee crossing
	// the second boundary and making the threshold strictly past updated_at.
	time.Sleep(2 * time.Second)

	// With a 1-second threshold, the message should be considered stale
	count, err := repo.ResetStuckSending(ctx, 1*time.Second)
	if err != nil {
		t.Fatalf("ResetStuckSending() error = %v", err)
	}
	if count != 1 {
		t.Errorf("ResetStuckSending() reset %d messages, want 1", count)
	}

	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}
	if got.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1 (incremented by reset)", got.RetryCount)
	}
}

func TestRepository_ResetStuckSending_IgnoresRecent(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Just started",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)
	repo.MarkSending(ctx, msg.ID)

	// With 5 minute threshold, the message is not stale yet
	count, err := repo.ResetStuckSending(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ResetStuckSending() error = %v", err)
	}
	if count != 0 {
		t.Errorf("ResetStuckSending() reset %d messages, want 0 (too recent)", count)
	}
}

func TestRepository_RemoveAttachmentID(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:     ch.ID,
		UserID:        user.ID,
		Content:       "With files",
		AttachmentIDs: []string{"file1", "file2", "file3"},
		ScheduledFor:  time.Now().Add(1 * time.Hour),
	}
	repo.Create(ctx, msg)

	affected, err := repo.RemoveAttachmentID(ctx, "file2")
	if err != nil {
		t.Fatalf("RemoveAttachmentID() error = %v", err)
	}
	if len(affected) != 1 {
		t.Fatalf("RemoveAttachmentID() affected %d messages, want 1", len(affected))
	}
	if len(affected[0].AttachmentIDs) != 2 {
		t.Errorf("AttachmentIDs length = %d, want 2", len(affected[0].AttachmentIDs))
	}

	// Verify the change persisted
	got, _ := repo.GetByID(ctx, msg.ID)
	if len(got.AttachmentIDs) != 2 {
		t.Errorf("persisted AttachmentIDs length = %d, want 2", len(got.AttachmentIDs))
	}
	for _, id := range got.AttachmentIDs {
		if id == "file2" {
			t.Error("file2 should have been removed")
		}
	}
}

func TestRepository_RemoveAttachmentID_NoMatch(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:     ch.ID,
		UserID:        user.ID,
		Content:       "With files",
		AttachmentIDs: []string{"file1"},
		ScheduledFor:  time.Now().Add(1 * time.Hour),
	}
	repo.Create(ctx, msg)

	affected, err := repo.RemoveAttachmentID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("RemoveAttachmentID() error = %v", err)
	}
	if len(affected) != 0 {
		t.Errorf("RemoveAttachmentID() affected %d messages, want 0", len(affected))
	}
}

func TestRepository_RemoveAttachmentID_IgnoresFailedMessages(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:     ch.ID,
		UserID:        user.ID,
		Content:       "Failed with files",
		AttachmentIDs: []string{"file1"},
		ScheduledFor:  time.Now().Add(-1 * time.Hour),
	}
	repo.Create(ctx, msg)
	repo.MarkFailed(ctx, msg.ID, "some error")

	affected, err := repo.RemoveAttachmentID(ctx, "file1")
	if err != nil {
		t.Fatalf("RemoveAttachmentID() error = %v", err)
	}
	if len(affected) != 0 {
		t.Errorf("RemoveAttachmentID() affected %d messages, want 0 (should ignore failed)", len(affected))
	}
}

func TestRepository_Update_ResetsStatusToPending(t *testing.T) {
	repo, user, _, ch := setupTest(t)
	ctx := context.Background()

	msg := &ScheduledMessage{
		ChannelID:    ch.ID,
		UserID:       user.ID,
		Content:      "Will fail then retry",
		ScheduledFor: time.Now().Add(-1 * time.Minute),
	}
	repo.Create(ctx, msg)
	repo.MarkFailed(ctx, msg.ID, "some error")

	// User edits the message (e.g. via retry) — should reset to pending
	msg.ScheduledFor = time.Now().Add(1 * time.Hour)
	err := repo.Update(ctx, msg)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := repo.GetByID(ctx, msg.ID)
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q after update", got.Status, StatusPending)
	}
}
