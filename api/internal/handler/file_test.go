package handler

import (
	"context"
	"testing"

	"github.com/enzyme/api/internal/channel"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/testutil"
)

func TestUploadFile_FilesDisabled(t *testing.T) {
	h, db := testHandler(t)
	h.filesEnabled = false

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.UploadFile(ctx, openapi.UploadFileRequestObject{
		Id: openapi.ChannelId(ch.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UploadFile403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestDeleteFile_FilesDisabled_StillWorks(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)
	fileID := createFileAttachment(t, db, ch.ID, user.ID)

	// Disable files after creating the attachment
	h.filesEnabled = false

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.DeleteFile(ctx, openapi.DeleteFileRequestObject{
		Id: fileID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteFile200JSONResponse); !ok {
		t.Fatalf("expected 200 response (delete should still work), got %T", resp)
	}
}

func TestDeleteFile_Owner(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	fileID := createFileAttachment(t, db, ch.ID, user.ID)

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.DeleteFile(ctx, openapi.DeleteFileRequestObject{
		Id: fileID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteFile200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestDeleteFile_Admin(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	uploader := testutil.CreateTestUser(t, db, "uploader@test.com", "Uploader")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	addWorkspaceMember(t, db, uploader.ID, ws.ID, "member")
	fileID := createFileAttachment(t, db, ch.ID, uploader.ID)

	// Owner (admin) deletes uploader's file
	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.DeleteFile(ctx, openapi.DeleteFileRequestObject{
		Id: fileID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteFile200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestDeleteFile_NotOwner(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	uploader := testutil.CreateTestUser(t, db, "uploader@test.com", "Uploader")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", channel.TypePublic)

	addWorkspaceMember(t, db, uploader.ID, ws.ID, "member")
	addWorkspaceMember(t, db, other.ID, ws.ID, "member")
	fileID := createFileAttachment(t, db, ch.ID, uploader.ID)

	// other (regular member) tries to delete uploader's file
	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.DeleteFile(ctx, openapi.DeleteFileRequestObject{
		Id: fileID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteFile403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestDeleteFile_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	resp, err := h.DeleteFile(ctx, openapi.DeleteFileRequestObject{
		Id: "some-file-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.DeleteFile401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}
