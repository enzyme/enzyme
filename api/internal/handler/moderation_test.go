package handler

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/enzyme/api/internal/moderation"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/testutil"
)

// testHandlerWithModeration creates a handler with moderation repo wired in.
func testHandlerWithModeration(t *testing.T) (*Handler, *sql.DB) {
	t.Helper()
	h, db := testHandler(t)
	h.moderationRepo = moderation.NewRepository(db)
	return h, db
}

func TestBlockUser_Success(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	target := testutil.CreateTestUser(t, db, "target@test.com", "Target")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")
	addWorkspaceMember(t, db, target.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.BlockUser(ctx, openapi.BlockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BlockUserJSONRequestBody{UserId: target.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BlockUser200JSONResponse); !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
}

func TestBlockUser_CannotBlockAdmin(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.BlockUser(ctx, openapi.BlockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BlockUserJSONRequestBody{UserId: admin.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BlockUser403JSONResponse); !ok {
		t.Fatalf("expected 403 when blocking admin, got %T", resp)
	}
}

func TestBlockUser_CannotBlockOwner(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.BlockUser(ctx, openapi.BlockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BlockUserJSONRequestBody{UserId: owner.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BlockUser403JSONResponse); !ok {
		t.Fatalf("expected 403 when blocking owner, got %T", resp)
	}
}

func TestBlockUser_AdminCanBlockMember(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, admin.ID)
	resp, err := h.BlockUser(ctx, openapi.BlockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BlockUserJSONRequestBody{UserId: member.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BlockUser200JSONResponse); !ok {
		t.Fatalf("expected 200 when admin blocks member, got %T", resp)
	}
}

func TestBlockUser_NonMemberCannotBlock(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	outsider := testutil.CreateTestUser(t, db, "outsider@test.com", "Outsider")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, outsider.ID)
	resp, err := h.BlockUser(ctx, openapi.BlockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BlockUserJSONRequestBody{UserId: member.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BlockUser403JSONResponse); !ok {
		t.Fatalf("expected 403 for non-member, got %T", resp)
	}
}

func TestBlockUser_CannotBlockSelf(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.BlockUser(ctx, openapi.BlockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BlockUserJSONRequestBody{UserId: owner.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BlockUser400JSONResponse); !ok {
		t.Fatalf("expected 400 for self-block, got %T", resp)
	}
}

func TestUnblockUser_NoRoleRestriction(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	target := testutil.CreateTestUser(t, db, "target@test.com", "Target")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")
	addWorkspaceMember(t, db, target.ID, ws.ID, "member")

	// Create a block first
	h.moderationRepo.CreateBlock(context.Background(), ws.ID, member.ID, target.ID)

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.UnblockUser(ctx, openapi.UnblockUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.UnblockUserJSONRequestBody{UserId: target.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UnblockUser200JSONResponse); !ok {
		t.Fatalf("expected 200 on unblock, got %T", resp)
	}
}

func TestListBlocks_Success(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	target := testutil.CreateTestUser(t, db, "target@test.com", "Target")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")
	addWorkspaceMember(t, db, target.ID, ws.ID, "member")

	// Create a block
	h.moderationRepo.CreateBlock(context.Background(), ws.ID, member.ID, target.ID)

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.ListBlocks(ctx, openapi.ListBlocksRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.ListBlocks200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}
	if r.Blocks == nil || len(*r.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %v", r.Blocks)
	}
	if (*r.Blocks)[0].WorkspaceId != ws.ID {
		t.Errorf("WorkspaceId = %q, want %q", (*r.Blocks)[0].WorkspaceId, ws.ID)
	}
}

// --- Ban handler permission tests ---

func TestBanUser_MemberCannotBan(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	target := testutil.CreateTestUser(t, db, "target@test.com", "Target")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")
	addWorkspaceMember(t, db, target.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.BanUser(ctx, openapi.BanUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BanUserJSONRequestBody{UserId: target.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BanUser403JSONResponse); !ok {
		t.Fatalf("expected 403 when member tries to ban, got %T", resp)
	}
}

func TestBanUser_AdminCanBanMember(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	target := testutil.CreateTestUser(t, db, "target@test.com", "Target")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")
	addWorkspaceMember(t, db, target.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, admin.ID)
	resp, err := h.BanUser(ctx, openapi.BanUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BanUserJSONRequestBody{UserId: target.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BanUser200JSONResponse); !ok {
		t.Fatalf("expected 200 when admin bans member, got %T", resp)
	}
}

func TestBanUser_AdminCannotBanAdmin(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin1 := testutil.CreateTestUser(t, db, "admin1@test.com", "Admin1")
	admin2 := testutil.CreateTestUser(t, db, "admin2@test.com", "Admin2")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin1.ID, ws.ID, "admin")
	addWorkspaceMember(t, db, admin2.ID, ws.ID, "admin")

	ctx := ctxWithUser(t, h, admin1.ID)
	resp, err := h.BanUser(ctx, openapi.BanUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BanUserJSONRequestBody{UserId: admin2.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BanUser403JSONResponse); !ok {
		t.Fatalf("expected 403 when admin bans another admin, got %T", resp)
	}
}

func TestBanUser_AdminCannotBanOwner(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")

	ctx := ctxWithUser(t, h, admin.ID)
	resp, err := h.BanUser(ctx, openapi.BanUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BanUserJSONRequestBody{UserId: owner.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BanUser403JSONResponse); !ok {
		t.Fatalf("expected 403 when admin bans owner, got %T", resp)
	}
}

func TestBanUser_OwnerCanBanAdmin(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.BanUser(ctx, openapi.BanUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BanUserJSONRequestBody{UserId: admin.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BanUser200JSONResponse); !ok {
		t.Fatalf("expected 200 when owner bans admin, got %T", resp)
	}
}

func TestBanUser_CannotSelfBan(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.BanUser(ctx, openapi.BanUserRequestObject{
		Wid:  ws.ID,
		Body: &openapi.BanUserJSONRequestBody{UserId: owner.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.BanUser400JSONResponse); !ok {
		t.Fatalf("expected 400 for self-ban, got %T", resp)
	}
}

// --- Ban-hide message filter tests ---

func TestListMessages_BanHideFiltersMessages(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	banned := testutil.CreateTestUser(t, db, "banned@test.com", "Banned")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, banned.ID, ws.ID, "member")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", "public")
	addChannelMember(t, db, banned.ID, ch.ID, nil)

	// Create messages from both users
	testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Owner message")
	testutil.CreateTestMessage(t, db, ch.ID, banned.ID, "Banned user message")

	// Create a ban with hide_messages=true
	hideMessages := true
	adminCtx := ctxWithUser(t, h, owner.ID)
	_, err := h.BanUser(adminCtx, openapi.BanUserRequestObject{
		Wid: ws.ID,
		Body: &openapi.BanUserJSONRequestBody{
			UserId:       banned.ID,
			HideMessages: &hideMessages,
		},
	})
	if err != nil {
		t.Fatalf("ban failed: %v", err)
	}

	// List messages as owner — banned user's messages should be hidden
	resp, err := h.ListMessages(adminCtx, openapi.ListMessagesRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	r, ok := resp.(openapi.ListMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}

	for _, msg := range r.Messages {
		if msg.UserId != nil && *msg.UserId == banned.ID && msg.Content != "[deleted]" {
			t.Errorf("banned user's message should be filtered out, found: %q", msg.Content)
		}
	}

	// Should contain owner's message
	found := false
	for _, msg := range r.Messages {
		if msg.Content == "Owner message" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find owner's message in results")
	}
}

func TestListMessages_BanWithoutHideShowsMessages(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	banned := testutil.CreateTestUser(t, db, "banned@test.com", "Banned")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, banned.ID, ws.ID, "member")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", "public")
	addChannelMember(t, db, banned.ID, ch.ID, nil)

	// Create a message from the user who will be banned
	testutil.CreateTestMessage(t, db, ch.ID, banned.ID, "Still visible message")

	// Ban WITHOUT hide_messages
	adminCtx := ctxWithUser(t, h, owner.ID)
	_, err := h.BanUser(adminCtx, openapi.BanUserRequestObject{
		Wid: ws.ID,
		Body: &openapi.BanUserJSONRequestBody{
			UserId: banned.ID,
		},
	})
	if err != nil {
		t.Fatalf("ban failed: %v", err)
	}

	// Messages should still be visible
	resp, err := h.ListMessages(adminCtx, openapi.ListMessagesRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	r, ok := resp.(openapi.ListMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}

	found := false
	for _, msg := range r.Messages {
		if msg.Content == "Still visible message" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected banned user's message to still be visible when hide_messages is false")
	}
}

// --- Block message filter tests ---

func TestListMessages_BlockFiltersMessages(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	blocker := testutil.CreateTestUser(t, db, "blocker@test.com", "Blocker")
	blocked := testutil.CreateTestUser(t, db, "blocked@test.com", "Blocked")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, blocker.ID, ws.ID, "member")
	addWorkspaceMember(t, db, blocked.ID, ws.ID, "member")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", "public")
	addChannelMember(t, db, blocker.ID, ch.ID, nil)
	addChannelMember(t, db, blocked.ID, ch.ID, nil)

	// Create messages from both users
	testutil.CreateTestMessage(t, db, ch.ID, blocker.ID, "Blocker message")
	testutil.CreateTestMessage(t, db, ch.ID, blocked.ID, "Blocked user message")

	// Blocker blocks the other user
	err := h.moderationRepo.CreateBlock(context.Background(), ws.ID, blocker.ID, blocked.ID)
	if err != nil {
		t.Fatalf("CreateBlock error: %v", err)
	}

	// List messages as blocker — blocked user's messages should be filtered
	blockerCtx := ctxWithUser(t, h, blocker.ID)
	resp, err := h.ListMessages(blockerCtx, openapi.ListMessagesRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	r, ok := resp.(openapi.ListMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}

	for _, msg := range r.Messages {
		if msg.Content == "Blocked user message" {
			t.Error("blocked user's message should be filtered out from blocker's view")
		}
	}

	// Blocker's own message should still be visible
	found := false
	for _, msg := range r.Messages {
		if msg.Content == "Blocker message" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected blocker's own message to be visible")
	}

	// List as blocked user — blocker's messages should still be visible (block is one-directional for viewing)
	blockedCtx := ctxWithUser(t, h, blocked.ID)
	resp2, err := h.ListMessages(blockedCtx, openapi.ListMessagesRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("ListMessages error for blocked user: %v", err)
	}
	r2, ok := resp2.(openapi.ListMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp2)
	}

	foundBlockerMsg := false
	for _, msg := range r2.Messages {
		if msg.Content == "Blocker message" {
			foundBlockerMsg = true
			break
		}
	}
	if !foundBlockerMsg {
		t.Error("blocked user should still see blocker's messages (block is one-directional for viewing)")
	}
}

// --- DM creation block enforcement tests ---

func TestCreateDM_BlockedUserCannotCreateDM(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	user1 := testutil.CreateTestUser(t, db, "user1@test.com", "User1")
	user2 := testutil.CreateTestUser(t, db, "user2@test.com", "User2")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, user1.ID, ws.ID, "member")
	addWorkspaceMember(t, db, user2.ID, ws.ID, "member")

	// user1 blocks user2
	err := h.moderationRepo.CreateBlock(context.Background(), ws.ID, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("CreateBlock error: %v", err)
	}

	// user2 tries to create DM with user1 — should fail
	user2Ctx := ctxWithUser(t, h, user2.ID)
	resp, err := h.CreateDM(user2Ctx, openapi.CreateDMRequestObject{
		Wid:  ws.ID,
		Body: &openapi.CreateDMJSONRequestBody{UserIds: []string{user1.ID}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateDM400JSONResponse); !ok {
		t.Fatalf("expected 400 when creating DM with blocked user, got %T", resp)
	}
}

func TestCreateDM_BlockerCannotCreateDM(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	user1 := testutil.CreateTestUser(t, db, "user1@test.com", "User1")
	user2 := testutil.CreateTestUser(t, db, "user2@test.com", "User2")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, user1.ID, ws.ID, "member")
	addWorkspaceMember(t, db, user2.ID, ws.ID, "member")

	// user1 blocks user2
	err := h.moderationRepo.CreateBlock(context.Background(), ws.ID, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("CreateBlock error: %v", err)
	}

	// user1 (the blocker) tries to create DM with user2 — should also fail (bidirectional check)
	user1Ctx := ctxWithUser(t, h, user1.ID)
	resp, err := h.CreateDM(user1Ctx, openapi.CreateDMRequestObject{
		Wid:  ws.ID,
		Body: &openapi.CreateDMJSONRequestBody{UserIds: []string{user2.ID}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateDM400JSONResponse); !ok {
		t.Fatalf("expected 400 when blocker creates DM with blocked user, got %T", resp)
	}
}

func TestCreateDM_NoBlockAllowsDM(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	user1 := testutil.CreateTestUser(t, db, "user1@test.com", "User1")
	user2 := testutil.CreateTestUser(t, db, "user2@test.com", "User2")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, user1.ID, ws.ID, "member")
	addWorkspaceMember(t, db, user2.ID, ws.ID, "member")

	// No blocks — DM should succeed
	user1Ctx := ctxWithUser(t, h, user1.ID)
	resp, err := h.CreateDM(user1Ctx, openapi.CreateDMRequestObject{
		Wid:  ws.ID,
		Body: &openapi.CreateDMJSONRequestBody{UserIds: []string{user2.ID}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateDM200JSONResponse); !ok {
		t.Fatalf("expected 200 when no blocks exist, got %T", resp)
	}
}

// --- Ban-hide expired ban test ---

func TestListMessages_ExpiredBanDoesNotFilter(t *testing.T) {
	h, db := testHandlerWithModeration(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	banned := testutil.CreateTestUser(t, db, "banned@test.com", "Banned")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, banned.ID, ws.ID, "member")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", "public")
	addChannelMember(t, db, banned.ID, ch.ID, nil)

	// Create a message from the user to be banned
	testutil.CreateTestMessage(t, db, ch.ID, banned.ID, "Visible after expiry")

	// Insert an expired ban directly with hide_messages=true
	expiredAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO workspace_bans (id, workspace_id, user_id, banned_by, reason, hide_messages, expires_at, created_at)
		VALUES ('ban1', ?, ?, ?, NULL, 1, ?, ?)
	`, ws.ID, banned.ID, owner.ID, expiredAt, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert expired ban: %v", err)
	}

	// Messages should still be visible since ban is expired
	adminCtx := ctxWithUser(t, h, owner.ID)
	resp, err := h.ListMessages(adminCtx, openapi.ListMessagesRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	r, ok := resp.(openapi.ListMessages200JSONResponse)
	if !ok {
		t.Fatalf("expected 200, got %T", resp)
	}

	found := false
	for _, msg := range r.Messages {
		if msg.Content == "Visible after expiry" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expired ban should not filter messages")
	}
}
