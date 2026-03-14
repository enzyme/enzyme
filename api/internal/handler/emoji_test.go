package handler

import (
	"testing"

	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/testutil"
)

func TestListCustomEmojis_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	testutil.CreateTestEmoji(t, db, ws.ID, user.ID, "thumbsup")
	testutil.CreateTestEmoji(t, db, ws.ID, user.ID, "wave")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.ListCustomEmojis(ctx, openapi.ListCustomEmojisRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.ListCustomEmojis200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if len(r.Emojis) != 2 {
		t.Errorf("expected 2 emojis, got %d", len(r.Emojis))
	}
}

func TestListCustomEmojis_NotMember(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	outsider := testutil.CreateTestUser(t, db, "outsider@test.com", "Outsider")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, outsider.ID)
	resp, err := h.ListCustomEmojis(ctx, openapi.ListCustomEmojisRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ListCustomEmojis403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestDeleteCustomEmoji_Creator(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	em := testutil.CreateTestEmoji(t, db, ws.ID, user.ID, "myemoji")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.DeleteCustomEmoji(ctx, openapi.DeleteCustomEmojiRequestObject{
		Id: em.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteCustomEmoji200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestDeleteCustomEmoji_Admin(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	creator := testutil.CreateTestUser(t, db, "creator@test.com", "Creator")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, creator.ID, ws.ID, "member")
	em := testutil.CreateTestEmoji(t, db, ws.ID, creator.ID, "their-emoji")

	// Owner (admin) can delete creator's emoji
	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.DeleteCustomEmoji(ctx, openapi.DeleteCustomEmojiRequestObject{
		Id: em.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteCustomEmoji200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestDeleteCustomEmoji_MemberCannotDeleteOthers(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	creator := testutil.CreateTestUser(t, db, "creator@test.com", "Creator")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, creator.ID, ws.ID, "member")
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")
	em := testutil.CreateTestEmoji(t, db, ws.ID, creator.ID, "not-yours")

	// Members cannot delete other members' emoji — only creator or admin/owner can
	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.DeleteCustomEmoji(ctx, openapi.DeleteCustomEmojiRequestObject{
		Id: em.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteCustomEmoji403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestDeleteCustomEmoji_GuestDenied(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	creator := testutil.CreateTestUser(t, db, "creator@test.com", "Creator")
	guest := testutil.CreateTestUser(t, db, "guest@test.com", "Guest")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, creator.ID, ws.ID, "member")
	addWorkspaceMember(t, db, guest.ID, ws.ID, "guest")
	em := testutil.CreateTestEmoji(t, db, ws.ID, creator.ID, "not-yours")

	// Default who_can_manage_custom_emoji is "members", so guest is denied
	ctx := ctxWithUser(t, h, guest.ID)
	resp, err := h.DeleteCustomEmoji(ctx, openapi.DeleteCustomEmojiRequestObject{
		Id: em.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteCustomEmoji403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestDeleteCustomEmoji_NotFound(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	_ = testutil.CreateTestWorkspace(t, db, user.ID, "WS")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.DeleteCustomEmoji(ctx, openapi.DeleteCustomEmojiRequestObject{
		Id: "nonexistent-emoji-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteCustomEmoji404JSONResponse); !ok {
		t.Fatalf("expected 404 response, got %T", resp)
	}
}
