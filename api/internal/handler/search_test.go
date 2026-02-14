package handler

import (
	"context"
	"testing"
	"time"

	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/openapi"
	"github.com/feather/api/internal/testutil"
)

func TestSearchMessages_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	testutil.CreateTestMessage(t, db, ch.ID, user.ID, "hello world from feather")
	testutil.CreateTestMessage(t, db, ch.ID, user.ID, "another message about search")

	ctx := ctxWithUser(t, h, user.ID)
	query := "hello"
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: query},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(r.Messages))
	}
	if r.TotalCount != 1 {
		t.Errorf("total_count = %d, want 1", r.TotalCount)
	}
	if r.Query != query {
		t.Errorf("query = %q, want %q", r.Query, query)
	}
	if r.Messages[0].ChannelName != "general" {
		t.Errorf("channel_name = %q, want %q", r.Messages[0].ChannelName, "general")
	}
}

func TestSearchMessages_EmptyQuery(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "   "},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SearchMessages400JSONResponse); !ok {
		t.Fatalf("expected 400, got %T", resp)
	}
}

func TestSearchMessages_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  "ws-id",
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SearchMessages401JSONResponse); !ok {
		t.Fatalf("expected 401, got %T", resp)
	}
}

func TestSearchMessages_NonMember(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.SearchMessages403JSONResponse); !ok {
		t.Fatalf("expected 403, got %T", resp)
	}
}

func TestSearchMessages_PrivateChannelAccessControl(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	// Owner creates a private channel
	privCh := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "secret", channel.TypePrivate)
	testutil.CreateTestMessage(t, db, privCh.ID, owner.ID, "secret information here")

	// Other user searches - should not see private channel messages
	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "secret"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 0 {
		t.Fatalf("expected 0 messages (private channel), got %d", len(r.Messages))
	}
}

func TestSearchMessages_PublicChannelAccess(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	// Owner creates a public channel (other is not a member)
	pubCh := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "public-ch", channel.TypePublic)
	testutil.CreateTestMessage(t, db, pubCh.ID, owner.ID, "visible public message")

	// Other user should see public channel messages even without being a member
	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "visible"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 message (public channel), got %d", len(r.Messages))
	}
}

func TestSearchMessages_ChannelFilter(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch1 := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	ch2 := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "random", channel.TypePublic)

	testutil.CreateTestMessage(t, db, ch1.ID, user.ID, "findme in general")
	testutil.CreateTestMessage(t, db, ch2.ID, user.ID, "findme in random")

	ctx := ctxWithUser(t, h, user.ID)
	channelID := ch1.ID
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid: openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{
			Query:     "findme",
			ChannelId: &channelID,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 message (filtered by channel), got %d", len(r.Messages))
	}
	if r.Messages[0].ChannelName != "general" {
		t.Errorf("channel = %q, want %q", r.Messages[0].ChannelName, "general")
	}
}

func TestSearchMessages_UserFilter(t *testing.T) {
	h, db := testHandler(t)

	user1 := testutil.CreateTestUser(t, db, "user1@test.com", "User1")
	user2 := testutil.CreateTestUser(t, db, "user2@test.com", "User2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")
	addWorkspaceMember(t, db, user2.ID, ws.ID, "member")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user1.ID, "general", channel.TypePublic)
	addChannelMember(t, db, user2.ID, ch.ID, nil)

	testutil.CreateTestMessage(t, db, ch.ID, user1.ID, "tagged message from user1")
	testutil.CreateTestMessage(t, db, ch.ID, user2.ID, "tagged message from user2")

	ctx := ctxWithUser(t, h, user1.ID)
	userID := user2.ID
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid: openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{
			Query:  "tagged",
			UserId: &userID,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 message (filtered by user), got %d", len(r.Messages))
	}
}

func TestSearchMessages_DeletedExcluded(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	msg := testutil.CreateTestMessage(t, db, ch.ID, user.ID, "deleteme soon")

	// Soft-delete the message
	_, err := db.ExecContext(context.Background(),
		`UPDATE messages SET deleted_at = ?, content = '[deleted]' WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), msg.ID)
	if err != nil {
		t.Fatalf("deleting message: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "deleteme"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 0 {
		t.Fatalf("expected 0 messages (deleted), got %d", len(r.Messages))
	}
}

func TestSearchMessages_SystemMessagesExcluded(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Create a system message directly
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO messages (id, channel_id, user_id, content, type, reply_count, created_at, updated_at)
		VALUES ('sys-1', ?, ?, 'joined #general', 'system', 0, ?, ?)
	`, ch.ID, user.ID, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating system message: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid:  openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{Query: "joined"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 0 {
		t.Fatalf("expected 0 messages (system messages excluded), got %d", len(r.Messages))
	}
}

func TestSearchMessages_Pagination(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Create 5 messages
	for i := 0; i < 5; i++ {
		testutil.CreateTestMessage(t, db, ch.ID, user.ID, "paginated result message")
	}

	ctx := ctxWithUser(t, h, user.ID)
	limit := 2
	resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid: openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{
			Query: "paginated",
			Limit: &limit,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if len(r.Messages) != 2 {
		t.Fatalf("expected 2 messages (limit), got %d", len(r.Messages))
	}
	if r.TotalCount != 5 {
		t.Errorf("total_count = %d, want 5", r.TotalCount)
	}
	if !r.HasMore {
		t.Error("expected has_more = true")
	}

	// Fetch page 2
	offset := 2
	resp2, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
		Wid: openapi.WorkspaceId(ws.ID),
		Body: &openapi.SearchMessagesJSONRequestBody{
			Query:  "paginated",
			Limit:  &limit,
			Offset: &offset,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r2, ok := resp2.(openapi.SearchMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp2)
	}
	if len(r2.Messages) != 2 {
		t.Fatalf("expected 2 messages (page 2), got %d", len(r2.Messages))
	}
	if !r2.HasMore {
		t.Error("expected has_more = true on page 2")
	}
}

func TestSearchMessages_SpecialCharsDontCrash(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test WS")
	testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)

	// These should not crash the FTS5 engine
	queries := []string{
		`"unclosed quote`,
		`hello OR world`,
		`test*`,
		`{brackets}`,
		`col:value`,
		`NOT something`,
		`a AND b`,
	}

	for _, q := range queries {
		resp, err := h.SearchMessages(ctx, openapi.SearchMessagesRequestObject{
			Wid:  openapi.WorkspaceId(ws.ID),
			Body: &openapi.SearchMessagesJSONRequestBody{Query: q},
		})
		if err != nil {
			t.Fatalf("query %q caused error: %v", q, err)
		}
		if _, ok := resp.(openapi.SearchMessages200JSONResponse); !ok {
			t.Fatalf("query %q: expected 200, got %T", q, resp)
		}
	}
}
