package message

import (
	"context"
	"errors"
	"testing"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/testutil"
)

func TestRepository_Create(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	msg := &Message{
		ChannelID: ch.ID,
		UserID:    &owner.ID,
		Content:   "Hello, world!",
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
}

func TestRepository_Create_ThreadReply(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create parent message
	parent := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Parent message")

	// Create reply
	reply := &Message{
		ChannelID:      ch.ID,
		UserID:         &owner.ID,
		Content:        "Reply message",
		ThreadParentID: &parent.ID,
	}

	err := repo.Create(ctx, reply)
	if err != nil {
		t.Fatalf("Create() reply error = %v", err)
	}

	// Verify parent's reply_count was updated
	updatedParent, _ := repo.GetByID(ctx, parent.ID)
	if updatedParent.ReplyCount != 1 {
		t.Errorf("parent ReplyCount = %d, want 1", updatedParent.ReplyCount)
	}
	if updatedParent.LastReplyAt == nil {
		t.Error("expected parent LastReplyAt to be set")
	}
}

func TestRepository_GetByID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	created := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	msg, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if msg.ID != created.ID {
		t.Errorf("ID = %q, want %q", msg.ID, created.ID)
	}
	if msg.Content != "Hello" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello")
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, ErrMessageNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrMessageNotFound)
	}
}

func TestRepository_GetByIDWithUser(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	created := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	msg, err := repo.GetByIDWithUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByIDWithUser() error = %v", err)
	}

	if msg.UserDisplayName != "Owner" {
		t.Errorf("UserDisplayName = %q, want %q", msg.UserDisplayName, "Owner")
	}
}

func TestRepository_Update(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Original content")

	err := repo.Update(ctx, msg.ID, "Updated content")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, msg.ID)
	if updated.Content != "Updated content" {
		t.Errorf("Content = %q, want %q", updated.Content, "Updated content")
	}
	if updated.EditedAt == nil {
		t.Error("expected EditedAt to be set")
	}
}

func TestRepository_Delete(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "To be deleted")

	err := repo.Delete(ctx, msg.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	deleted, _ := repo.GetByID(ctx, msg.ID)
	if deleted.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
	if deleted.Content != "[deleted]" {
		t.Errorf("Content = %q, want %q", deleted.Content, "[deleted]")
	}
}

func TestRepository_Delete_AlreadyDeleted(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "To be deleted")

	repo.Delete(ctx, msg.ID)

	// Second delete should fail
	err := repo.Delete(ctx, msg.ID)
	if !errors.Is(err, ErrMessageNotFound) {
		t.Errorf("Delete() error = %v, want %v", err, ErrMessageNotFound)
	}
}

func TestRepository_List(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create multiple messages
	testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Message 1")
	testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Message 2")
	testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Message 3")

	result, err := repo.List(ctx, ch.ID, ListOptions{Limit: 10}, nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(result.Messages))
	}

	// Messages should be in DESC order (newest first)
	if result.Messages[0].Content != "Message 3" {
		t.Errorf("first message = %q, want %q", result.Messages[0].Content, "Message 3")
	}
}

func TestRepository_List_Pagination(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create 5 messages
	for i := 1; i <= 5; i++ {
		testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Message")
	}

	// Get first page
	result1, err := repo.List(ctx, ch.ID, ListOptions{Limit: 2}, nil)
	if err != nil {
		t.Fatalf("List() page 1 error = %v", err)
	}

	if len(result1.Messages) != 2 {
		t.Fatalf("page 1 len(Messages) = %d, want 2", len(result1.Messages))
	}
	if !result1.HasMore {
		t.Error("expected HasMore = true for page 1")
	}

	// Get second page using cursor
	result2, err := repo.List(ctx, ch.ID, ListOptions{
		Limit:  2,
		Cursor: result1.NextCursor,
	}, nil)
	if err != nil {
		t.Fatalf("List() page 2 error = %v", err)
	}

	if len(result2.Messages) != 2 {
		t.Fatalf("page 2 len(Messages) = %d, want 2", len(result2.Messages))
	}

	// Verify no duplicate messages
	if result2.Messages[0].ID == result1.Messages[0].ID || result2.Messages[0].ID == result1.Messages[1].ID {
		t.Error("page 2 contains duplicate messages from page 1")
	}
}

func TestRepository_List_ExcludesThreadReplies(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create parent message
	parent := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Parent")

	// Create thread reply
	reply := &Message{
		ChannelID:      ch.ID,
		UserID:         &owner.ID,
		Content:        "Reply",
		ThreadParentID: &parent.ID,
	}
	repo.Create(ctx, reply)

	// List should only return the parent
	result, _ := repo.List(ctx, ch.ID, ListOptions{Limit: 10}, nil)
	if len(result.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(result.Messages))
	}
	if result.Messages[0].ID != parent.ID {
		t.Errorf("expected parent message, got %q", result.Messages[0].ID)
	}
}

func TestRepository_ListThread(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create parent message
	parent := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Parent")

	// Create thread replies
	for i := 1; i <= 3; i++ {
		reply := &Message{
			ChannelID:      ch.ID,
			UserID:         &owner.ID,
			Content:        "Reply",
			ThreadParentID: &parent.ID,
		}
		repo.Create(ctx, reply)
	}

	result, err := repo.ListThread(ctx, parent.ID, ListOptions{Limit: 10}, nil)
	if err != nil {
		t.Fatalf("ListThread() error = %v", err)
	}

	if len(result.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(result.Messages))
	}
}

func TestRepository_AddReaction(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	reaction, err := repo.AddReaction(ctx, msg.ID, owner.ID, "👍")
	if err != nil {
		t.Fatalf("AddReaction() error = %v", err)
	}

	if reaction.ID == "" {
		t.Error("expected non-empty ID")
	}
	if reaction.Emoji != "👍" {
		t.Errorf("Emoji = %q, want %q", reaction.Emoji, "👍")
	}
}

func TestRepository_AddReaction_Duplicate(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	repo.AddReaction(ctx, msg.ID, owner.ID, "👍")

	_, err := repo.AddReaction(ctx, msg.ID, owner.ID, "👍")
	if !errors.Is(err, ErrReactionExists) {
		t.Errorf("AddReaction() error = %v, want %v", err, ErrReactionExists)
	}
}

func TestRepository_RemoveReaction(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	repo.AddReaction(ctx, msg.ID, owner.ID, "👍")

	err := repo.RemoveReaction(ctx, msg.ID, owner.ID, "👍")
	if err != nil {
		t.Fatalf("RemoveReaction() error = %v", err)
	}
}

func TestRepository_RemoveReaction_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	err := repo.RemoveReaction(ctx, msg.ID, owner.ID, "👍")
	if !errors.Is(err, ErrReactionNotFound) {
		t.Errorf("RemoveReaction() error = %v, want %v", err, ErrReactionNotFound)
	}
}

func TestRepository_List_IncludesReactions(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	repo.AddReaction(ctx, msg.ID, owner.ID, "👍")
	repo.AddReaction(ctx, msg.ID, owner.ID, "❤️")

	result, _ := repo.List(ctx, ch.ID, ListOptions{Limit: 10}, nil)
	if len(result.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(result.Messages))
	}

	if len(result.Messages[0].Reactions) != 2 {
		t.Errorf("len(Reactions) = %d, want 2", len(result.Messages[0].Reactions))
	}
}

func TestRepository_Create_AlsoSendToChannel(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create parent message
	parent := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Parent message")

	// Create thread reply with also_send_to_channel
	reply := &Message{
		ChannelID:         ch.ID,
		UserID:            &owner.ID,
		Content:           "Broadcast reply",
		ThreadParentID:    &parent.ID,
		AlsoSendToChannel: true,
	}

	err := repo.Create(ctx, reply)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify the flag was persisted
	fetched, err := repo.GetByID(ctx, reply.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if !fetched.AlsoSendToChannel {
		t.Error("expected AlsoSendToChannel = true")
	}
	if fetched.ThreadParentID == nil || *fetched.ThreadParentID != parent.ID {
		t.Error("expected ThreadParentID to be set")
	}
}

func TestRepository_List_IncludesBroadcastReplies(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create parent message
	parent := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Parent")

	// Create a normal thread reply (should NOT appear in channel list)
	normalReply := &Message{
		ChannelID:      ch.ID,
		UserID:         &owner.ID,
		Content:        "Normal reply",
		ThreadParentID: &parent.ID,
	}
	repo.Create(ctx, normalReply)

	// Create a broadcast thread reply (should appear in channel list)
	broadcastReply := &Message{
		ChannelID:         ch.ID,
		UserID:            &owner.ID,
		Content:           "Broadcast reply",
		ThreadParentID:    &parent.ID,
		AlsoSendToChannel: true,
	}
	repo.Create(ctx, broadcastReply)

	// List should return parent + broadcast reply, but NOT normal reply
	result, err := repo.List(ctx, ch.ID, ListOptions{Limit: 10}, nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(result.Messages))
	}

	// Messages are DESC order, so broadcast reply is first
	if result.Messages[0].Content != "Broadcast reply" {
		t.Errorf("first message = %q, want %q", result.Messages[0].Content, "Broadcast reply")
	}
	if !result.Messages[0].AlsoSendToChannel {
		t.Error("expected broadcast reply to have AlsoSendToChannel = true")
	}
	if result.Messages[1].Content != "Parent" {
		t.Errorf("second message = %q, want %q", result.Messages[1].Content, "Parent")
	}
}

func TestRepository_Delete_ThreadReply_DecrementsReplyCount(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create parent message
	parent := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Parent")

	// Create two thread replies
	reply1 := &Message{
		ChannelID:      ch.ID,
		UserID:         &owner.ID,
		Content:        "Reply 1",
		ThreadParentID: &parent.ID,
	}
	repo.Create(ctx, reply1)

	reply2 := &Message{
		ChannelID:      ch.ID,
		UserID:         &owner.ID,
		Content:        "Reply 2",
		ThreadParentID: &parent.ID,
	}
	repo.Create(ctx, reply2)

	// Parent should have reply_count = 2
	parentMsg, _ := repo.GetByID(ctx, parent.ID)
	if parentMsg.ReplyCount != 2 {
		t.Fatalf("initial ReplyCount = %d, want 2", parentMsg.ReplyCount)
	}

	// Delete one reply
	err := repo.Delete(ctx, reply1.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Parent's reply_count should be decremented to 1
	parentMsg, _ = repo.GetByID(ctx, parent.ID)
	if parentMsg.ReplyCount != 1 {
		t.Errorf("ReplyCount after delete = %d, want 1", parentMsg.ReplyCount)
	}

	// Delete second reply
	err = repo.Delete(ctx, reply2.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Parent's reply_count should be 0
	parentMsg, _ = repo.GetByID(ctx, parent.ID)
	if parentMsg.ReplyCount != 0 {
		t.Errorf("ReplyCount after second delete = %d, want 0", parentMsg.ReplyCount)
	}
}

func TestRepository_Delete_NonThreadMessage_NoReplyCountChange(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	// Create two independent messages (no thread relationship)
	msg1 := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Message 1")
	msg2 := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Message 2")

	// Delete msg1 should not affect msg2
	err := repo.Delete(ctx, msg1.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	msg2Fetched, _ := repo.GetByID(ctx, msg2.ID)
	if msg2Fetched.ReplyCount != 0 {
		t.Errorf("unrelated message ReplyCount = %d, want 0", msg2Fetched.ReplyCount)
	}
}
