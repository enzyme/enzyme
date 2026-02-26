package handler

import (
	"context"
	"database/sql"
	"testing"

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
