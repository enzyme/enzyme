package handler

import (
	"context"
	"testing"

	"github.com/enzyme/server/internal/channel"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/testutil"
)

func TestCreateChannel_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.CreateChannel(ctx, openapi.CreateChannelRequestObject{
		Wid: ws.ID,
		Body: &openapi.CreateChannelJSONRequestBody{
			Name: "new-channel",
			Type: openapi.ChannelType("public"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.CreateChannel200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Channel.Name != "new-channel" {
		t.Errorf("name = %q, want %q", r.Channel.Name, "new-channel")
	}
}

func TestCreateChannel_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	resp, err := h.CreateChannel(ctx, openapi.CreateChannelRequestObject{
		Wid: "ws-id",
		Body: &openapi.CreateChannelJSONRequestBody{
			Name: "ch",
			Type: openapi.ChannelType("public"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateChannel401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}

func TestCreateChannel_InvalidName(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")

	ctx := ctxWithUser(t, h, user.ID)

	tests := []struct {
		name     string
		chanName string
	}{
		{"empty", ""},
		{"has spaces", "has spaces"},
		{"uppercase", "UPPER"},
		{"special chars", "chan!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := h.CreateChannel(ctx, openapi.CreateChannelRequestObject{
				Wid: ws.ID,
				Body: &openapi.CreateChannelJSONRequestBody{
					Name: tt.chanName,
					Type: openapi.ChannelType("public"),
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := resp.(openapi.CreateChannel400JSONResponse); !ok {
				t.Fatalf("expected 400 response for name %q, got %T", tt.chanName, resp)
			}
		})
	}
}

func TestCreateChannel_GuestCannotCreate(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	guest := testutil.CreateTestUser(t, db, "guest@test.com", "Guest")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, guest.ID, ws.ID, "guest")

	ctx := ctxWithUser(t, h, guest.ID)
	resp, err := h.CreateChannel(ctx, openapi.CreateChannelRequestObject{
		Wid: ws.ID,
		Body: &openapi.CreateChannelJSONRequestBody{
			Name: "new-ch",
			Type: openapi.ChannelType("public"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateChannel403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateChannel_Admin(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "original", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	newName := "renamed"
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id: ch.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{
			Name: &newName,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateChannel200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Channel.Name != "renamed" {
		t.Errorf("name = %q, want %q", r.Channel.Name, "renamed")
	}
}

func TestUpdateChannel_NotAdmin(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "locked", channel.TypePublic)

	addWorkspaceMember(t, db, member.ID, ws.ID, "member")
	posterRole := "poster"
	addChannelMember(t, db, member.ID, ch.ID, &posterRole)

	ctx := ctxWithUser(t, h, member.ID)
	newName := "hacked"
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id:   ch.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{Name: &newName},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateChannel403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateChannel_ChangeType(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "my-channel", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	newType := openapi.ChannelType("private")
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id:   ch.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{Type: &newType},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateChannel200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if string(r.Channel.Type) != "private" {
		t.Errorf("type = %q, want %q", r.Channel.Type, "private")
	}
}

func TestUpdateChannel_CannotMakeDefaultPrivate(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Mark as default channel
	_, err := db.ExecContext(context.Background(),
		`UPDATE channels SET is_default = 1 WHERE id = ?`, ch.ID)
	if err != nil {
		t.Fatalf("marking as default: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	newType := openapi.ChannelType("private")
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id:   ch.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{Type: &newType},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestUpdateChannel_CannotChangeTypeDM(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "dm-chan", channel.TypeDM)

	ctx := ctxWithUser(t, h, user.ID)
	newType := openapi.ChannelType("public")
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id:   ch.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{Type: &newType},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestUpdateChannel_DuplicateName(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	testutil.CreateTestChannel(t, db, ws.ID, user.ID, "taken-name", channel.TypePublic)
	ch2 := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "my-channel", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	dupName := "taken-name"
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id:   ch2.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{Name: &dupName},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestUpdateChannel_InvalidType(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "my-channel", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	badType := openapi.ChannelType("dm")
	resp, err := h.UpdateChannel(ctx, openapi.UpdateChannelRequestObject{
		Id:   ch.ID,
		Body: &openapi.UpdateChannelJSONRequestBody{Type: &badType},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestArchiveChannel_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "archive-me", channel.TypePublic)

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.ArchiveChannel(ctx, openapi.ArchiveChannelRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ArchiveChannel200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestArchiveChannel_CannotArchiveDM(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "dm", channel.TypeDM)

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.ArchiveChannel(ctx, openapi.ArchiveChannelRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ArchiveChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestArchiveChannel_CannotArchiveDefault(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", channel.TypePublic)

	// Mark as default channel
	_, err := db.ExecContext(context.Background(),
		`UPDATE channels SET is_default = 1 WHERE id = ?`, ch.ID)
	if err != nil {
		t.Fatalf("marking as default: %v", err)
	}

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.ArchiveChannel(ctx, openapi.ArchiveChannelRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ArchiveChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestJoinChannel_Public(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	joiner := testutil.CreateTestUser(t, db, "joiner@test.com", "Joiner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "public-ch", channel.TypePublic)

	addWorkspaceMember(t, db, joiner.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, joiner.ID)
	resp, err := h.JoinChannel(ctx, openapi.JoinChannelRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.JoinChannel200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestJoinChannel_Private(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	joiner := testutil.CreateTestUser(t, db, "joiner@test.com", "Joiner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "private-ch", channel.TypePrivate)

	addWorkspaceMember(t, db, joiner.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, joiner.ID)
	resp, err := h.JoinChannel(ctx, openapi.JoinChannelRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.JoinChannel400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestLeaveChannel_Success(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "leaveable", channel.TypePublic)

	addWorkspaceMember(t, db, member.ID, ws.ID, "member")
	posterRole := "poster"
	addChannelMember(t, db, member.ID, ch.ID, &posterRole)

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.LeaveChannel(ctx, openapi.LeaveChannelRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.LeaveChannel200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestAddChannelMember_Success(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	newMember := testutil.CreateTestUser(t, db, "new@test.com", "New")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "team", channel.TypePrivate)

	addWorkspaceMember(t, db, newMember.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.AddChannelMember(ctx, openapi.AddChannelMemberRequestObject{
		Id: ch.ID,
		Body: &openapi.AddChannelMemberJSONRequestBody{
			UserId: newMember.ID,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.AddChannelMember200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestListChannelMembers_PrivateNonMember(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "secret", channel.TypePrivate)

	addWorkspaceMember(t, db, other.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, other.ID)
	resp, err := h.ListChannelMembers(ctx, openapi.ListChannelMembersRequestObject{
		Id: ch.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ListChannelMembers404JSONResponse); !ok {
		t.Fatalf("expected 404 response, got %T", resp)
	}
}
