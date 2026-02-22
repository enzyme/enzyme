package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/linkpreview"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/testutil"
	"github.com/oklog/ulid/v2"
)

func TestSendMessage_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	content := "hello"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id: "some-channel-id",
		Body: &openapi.SendMessageJSONRequestBody{
			Content: &content,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SendMessage401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}

func TestSendMessage_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "sender@test.com", "Sender")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	content := "Hello, world!"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id: ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{
			Content: &content,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SendMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message == nil {
		t.Fatal("expected message in response")
	}
	if r.Message.Content != "Hello, world!" {
		t.Errorf("content = %q, want %q", r.Message.Content, "Hello, world!")
	}
	if r.Message.UserId == nil || *r.Message.UserId != user.ID {
		t.Errorf("user_id = %v, want %q", r.Message.UserId, user.ID)
	}
}

func TestSendMessage_ArchivedChannel(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "archived", channel.TypePublic)

	// Archive the channel
	_, err := db.ExecContext(context.Background(),
		`UPDATE channels SET archived_at = datetime('now') WHERE id = ?`, ch.ID)
	if err != nil {
		t.Fatalf("archiving channel: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	content := "test"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id:   ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{Content: &content},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SendMessage400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestSendMessage_NotMember_PrivateChannel(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "secret", channel.TypePrivate)

	// other is workspace member but not channel member
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, other.ID)
	content := "test"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id:   ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{Content: &content},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SendMessage403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestSendMessage_PublicChannel_AutoJoin(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "public-ch", channel.TypePublic)

	// other is workspace member but not channel member
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, other.ID)
	content := "auto-joined!"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id:   ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{Content: &content},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SendMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.Content != "auto-joined!" {
		t.Errorf("content = %q, want %q", r.Message.Content, "auto-joined!")
	}
}

func TestSendMessage_EmptyContent_NoAttachments(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	empty := "   "
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id:   ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{Content: &empty},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SendMessage400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestSendMessage_ThreadReply(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	parent := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Parent message")

	ctx := ctxWithUser(t, h, user.ID)
	content := "thread reply"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id: ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{
			Content:        &content,
			ThreadParentId: &parent.ID,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SendMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.ThreadParentId == nil || *r.Message.ThreadParentId != parent.ID {
		t.Errorf("thread_parent_id = %v, want %q", r.Message.ThreadParentId, parent.ID)
	}
}

func TestSendMessage_CannotReplyToReply(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	parent := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Parent")

	// Create a reply (a message with thread_parent_id set)
	replyID := ulid.Make().String()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO messages (id, channel_id, user_id, content, thread_parent_id, reply_count, created_at, updated_at)
		VALUES (?, ?, ?, 'Reply', ?, 0, datetime('now'), datetime('now'))
	`, replyID, ch.ID, user.ID, parent.ID)
	if err != nil {
		t.Fatalf("creating reply: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	content := "reply to reply"
	resp, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id: ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{
			Content:        &content,
			ThreadParentId: &replyID,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SendMessage400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestDeleteMessage_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Delete me")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.DeleteMessage(ctx, openapi.DeleteMessageRequestObject{
		Id: msg.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteMessage200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestDeleteMessage_AdminCanDelete(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	author := testutil.CreateTestUser(t, db, "author@test.com", "Author")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	addWorkspaceMember(t, db, author.ID, ws.ID, "member")
	msg := testutil.CreateTestMessage(t, db, ch.ID, author.ID, "Author's message")

	// Owner (admin) deletes author's message
	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.DeleteMessage(ctx, openapi.DeleteMessageRequestObject{
		Id: msg.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteMessage200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestDeleteMessage_NotAuthor(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	author := testutil.CreateTestUser(t, db, "author@test.com", "Author")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	addWorkspaceMember(t, db, author.ID, ws.ID, "member")
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")
	msg := testutil.CreateTestMessage(t, db, ch.ID, author.ID, "Author's message")

	// other (regular member) tries to delete author's message
	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.DeleteMessage(ctx, openapi.DeleteMessageRequestObject{
		Id: msg.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteMessage403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestDeleteMessage_SystemMessage(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Create a system message
	sysMsgID := ulid.Make().String()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO messages (id, channel_id, content, type, reply_count, created_at, updated_at)
		VALUES (?, ?, 'User joined', 'system', 0, datetime('now'), datetime('now'))
	`, sysMsgID, ch.ID)
	if err != nil {
		t.Fatalf("creating system message: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.DeleteMessage(ctx, openapi.DeleteMessageRequestObject{
		Id: sysMsgID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteMessage403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateMessage_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Original")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id: msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{
			Content: "Updated content",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.Content != "Updated content" {
		t.Errorf("content = %q, want %q", r.Message.Content, "Updated content")
	}
}

func TestUpdateMessage_NotAuthor(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	author := testutil.CreateTestUser(t, db, "author@test.com", "Author")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	addWorkspaceMember(t, db, author.ID, ws.ID, "member")
	msg := testutil.CreateTestMessage(t, db, ch.ID, author.ID, "Author's message")

	// Owner tries to edit author's message (not allowed, only author can edit)
	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id:   msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{Content: "hacked"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateMessage403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateMessage_EmptyContent(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Original")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id:   msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{Content: "   "},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateMessage400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestListMessages_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	resp, err := h.ListMessages(ctx, openapi.ListMessagesRequestObject{
		Id: "some-channel-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ListMessages401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}

func TestListMessages_NotMember_Private(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "secret", channel.TypePrivate)

	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.ListMessages(ctx, openapi.ListMessagesRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ListMessages403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestAddReaction_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "React to me")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.AddReaction(ctx, openapi.AddReactionRequestObject{
		Id: msg.ID,
		Body: &openapi.AddReactionJSONRequestBody{
			Emoji: "👍",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.AddReaction200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Reaction == nil {
		t.Fatal("expected reaction in response")
	}
	if r.Reaction.Emoji != "👍" {
		t.Errorf("emoji = %q, want %q", r.Reaction.Emoji, "👍")
	}
}

func TestRemoveReaction_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "React to me")

	ctx := ctxWithUser(t, h, user.ID)

	// Add reaction first
	_, err := h.AddReaction(ctx, openapi.AddReactionRequestObject{
		Id:   msg.ID,
		Body: &openapi.AddReactionJSONRequestBody{Emoji: "👍"},
	})
	if err != nil {
		t.Fatalf("adding reaction: %v", err)
	}

	// Remove it
	resp, err := h.RemoveReaction(ctx, openapi.RemoveReactionRequestObject{
		Id:   msg.ID,
		Body: &openapi.RemoveReactionJSONRequestBody{Emoji: "👍"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.RemoveReaction200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestGetMessage_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Hello")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.GetMessage(ctx, openapi.GetMessageRequestObject{
		Id: msg.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.GetMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message == nil {
		t.Fatal("expected message in response")
	}
	if r.Message.Content != "Hello" {
		t.Errorf("content = %q, want %q", r.Message.Content, "Hello")
	}
}

func TestGetMessage_PrivateNonMember(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "secret", channel.TypePrivate)
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Private message")

	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.GetMessage(ctx, openapi.GetMessageRequestObject{
		Id: msg.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.GetMessage404JSONResponse); !ok {
		t.Fatalf("expected 404 response, got %T", resp)
	}
}

func TestListThread_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	parent := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Thread parent")

	// Create a reply via SendMessage to properly set up thread
	ctx := ctxWithUser(t, h, user.ID)
	replyContent := "Thread reply"
	_, err := h.SendMessage(ctx, openapi.SendMessageRequestObject{
		Id: ch.ID,
		Body: &openapi.SendMessageJSONRequestBody{
			Content:        &replyContent,
			ThreadParentId: &parent.ID,
		},
	})
	if err != nil {
		t.Fatalf("sending reply: %v", err)
	}

	resp, err := h.ListThread(ctx, openapi.ListThreadRequestObject{
		Id: parent.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.ListThread200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if len(r.Messages) == 0 {
		t.Error("expected at least one thread reply")
	}
}

func TestAddReaction_Duplicate(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "React to me")

	ctx := ctxWithUser(t, h, user.ID)

	// Add reaction first time
	_, err := h.AddReaction(ctx, openapi.AddReactionRequestObject{
		Id:   msg.ID,
		Body: &openapi.AddReactionJSONRequestBody{Emoji: "👍"},
	})
	if err != nil {
		t.Fatalf("first reaction: %v", err)
	}

	// Add same reaction again — returns an error (duplicate)
	_, err = h.AddReaction(ctx, openapi.AddReactionRequestObject{
		Id:   msg.ID,
		Body: &openapi.AddReactionJSONRequestBody{Emoji: "👍"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate reaction")
	}
}

func TestUpdateMessage_LinkPreview_SameURL(t *testing.T) {
	h, db := testHandlerWithLinkPreviews(t, &http.Client{})

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Check https://example.com")

	// Seed cache and create preview for the message
	seedLinkPreviewCache(t, db, "https://example.com", "Example")
	ctx := ctxWithUser(t, h, user.ID)
	// Send to trigger link preview creation
	h.linkPreviewRepo.CreatePreview(ctx, &linkpreview.Preview{
		MessageID: msg.ID,
		URL:       "https://example.com",
		Title:     "Example",
	})

	// Edit message text but keep same URL
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id: msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{
			Content: "Updated text https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.LinkPreview == nil {
		t.Fatal("expected link preview to be preserved")
	}
	if r.Message.LinkPreview.Url != "https://example.com" {
		t.Errorf("url = %q, want %q", r.Message.LinkPreview.Url, "https://example.com")
	}
}

func TestUpdateMessage_LinkPreview_URLRemoved(t *testing.T) {
	h, db := testHandlerWithLinkPreviews(t, &http.Client{})

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Check https://example.com")

	// Create existing preview
	ctx := ctxWithUser(t, h, user.ID)
	h.linkPreviewRepo.CreatePreview(ctx, &linkpreview.Preview{
		MessageID: msg.ID,
		URL:       "https://example.com",
		Title:     "Example",
	})

	// Edit to remove the URL
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id: msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{
			Content: "No link here anymore",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.LinkPreview != nil {
		t.Errorf("expected link preview to be removed, got %+v", r.Message.LinkPreview)
	}

	// Verify deleted from database
	preview, err := h.linkPreviewRepo.GetForMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetForMessage: %v", err)
	}
	if preview != nil {
		t.Errorf("expected preview to be deleted from DB, got %+v", preview)
	}
}

func TestUpdateMessage_LinkPreview_URLChanged(t *testing.T) {
	h, db := testHandlerWithLinkPreviews(t, &http.Client{})

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "Check https://old.com")

	// Create existing preview for old URL
	ctx := ctxWithUser(t, h, user.ID)
	h.linkPreviewRepo.CreatePreview(ctx, &linkpreview.Preview{
		MessageID: msg.ID,
		URL:       "https://old.com",
		Title:     "Old",
	})

	// Seed cache for the new URL so it resolves synchronously
	seedLinkPreviewCache(t, db, "https://new.com", "New")

	// Edit to change the URL
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id: msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{
			Content: "Check https://new.com instead",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.LinkPreview == nil {
		t.Fatal("expected new link preview")
	}
	if r.Message.LinkPreview.Url != "https://new.com" {
		t.Errorf("url = %q, want %q", r.Message.LinkPreview.Url, "https://new.com")
	}
	if *r.Message.LinkPreview.Title != "New" {
		t.Errorf("title = %q, want %q", *r.Message.LinkPreview.Title, "New")
	}

	// Verify old preview is gone from database
	preview, err := h.linkPreviewRepo.GetForMessage(ctx, msg.ID)
	if err != nil {
		t.Fatalf("GetForMessage: %v", err)
	}
	if preview == nil {
		t.Fatal("expected new preview in DB")
	}
	if preview.URL != "https://new.com" {
		t.Errorf("DB preview url = %q, want %q", preview.URL, "https://new.com")
	}
}

func TestUpdateMessage_LinkPreview_URLAdded(t *testing.T) {
	h, db := testHandlerWithLinkPreviews(t, &http.Client{})

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "No link here")

	// Seed cache so the fetch resolves synchronously
	seedLinkPreviewCache(t, db, "https://added.com", "Added")

	// Edit to add a URL
	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.UpdateMessage(ctx, openapi.UpdateMessageRequestObject{
		Id: msg.ID,
		Body: &openapi.UpdateMessageJSONRequestBody{
			Content: "Now with https://added.com",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateMessage200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Message.LinkPreview == nil {
		t.Fatal("expected link preview to be added")
	}
	if r.Message.LinkPreview.Url != "https://added.com" {
		t.Errorf("url = %q, want %q", r.Message.LinkPreview.Url, "https://added.com")
	}
	if *r.Message.LinkPreview.Title != "Added" {
		t.Errorf("title = %q, want %q", *r.Message.LinkPreview.Title, "Added")
	}
}
