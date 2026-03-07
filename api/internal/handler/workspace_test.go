package handler

import (
	"context"
	"testing"

	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/testutil"
)

func TestCreateWorkspace_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.CreateWorkspace(ctx, openapi.CreateWorkspaceRequestObject{
		Body: &openapi.CreateWorkspaceJSONRequestBody{
			Name: "My Workspace",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.CreateWorkspace200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Workspace == nil {
		t.Fatal("expected workspace in response")
	}
	if r.Workspace.Name != "My Workspace" {
		t.Errorf("name = %q, want %q", r.Workspace.Name, "My Workspace")
	}
	if r.Workspace.Id == "" {
		t.Error("expected workspace ID to be set")
	}
}

func TestCreateWorkspace_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	resp, err := h.CreateWorkspace(ctx, openapi.CreateWorkspaceRequestObject{
		Body: &openapi.CreateWorkspaceJSONRequestBody{Name: "WS"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateWorkspace401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}

func TestCreateWorkspace_EmptyName(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.CreateWorkspace(ctx, openapi.CreateWorkspaceRequestObject{
		Body: &openapi.CreateWorkspaceJSONRequestBody{Name: "   "},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateWorkspace400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestGetWorkspace_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "My WS")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.GetWorkspace(ctx, openapi.GetWorkspaceRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.GetWorkspace200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Workspace.Name != "My WS" {
		t.Errorf("name = %q, want %q", r.Workspace.Name, "My WS")
	}
}

func TestUpdateWorkspace_Admin(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Original")

	ctx := ctxWithUser(t, h, user.ID)
	newName := "Updated"
	resp, err := h.UpdateWorkspace(ctx, openapi.UpdateWorkspaceRequestObject{
		Wid:  ws.ID,
		Body: &openapi.UpdateWorkspaceJSONRequestBody{Name: &newName},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.UpdateWorkspace200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Workspace.Name != "Updated" {
		t.Errorf("name = %q, want %q", r.Workspace.Name, "Updated")
	}
}

func TestUpdateWorkspace_MemberDenied(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	newName := "Hacked"
	resp, err := h.UpdateWorkspace(ctx, openapi.UpdateWorkspaceRequestObject{
		Wid:  ws.ID,
		Body: &openapi.UpdateWorkspaceJSONRequestBody{Name: &newName},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspace403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestListWorkspaceMembers_Success(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.ListWorkspaceMembers(ctx, openapi.ListWorkspaceMembersRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.ListWorkspaceMembers200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Members == nil || len(*r.Members) < 2 {
		t.Fatalf("expected at least 2 members, got %d", len(*r.Members))
	}
}

func TestRemoveWorkspaceMember_Self(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.RemoveWorkspaceMember(ctx, openapi.RemoveWorkspaceMemberRequestObject{
		Wid:  ws.ID,
		Body: &openapi.RemoveWorkspaceMemberJSONRequestBody{UserId: member.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.RemoveWorkspaceMember200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestRemoveWorkspaceMember_AdminRemovesOther(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.RemoveWorkspaceMember(ctx, openapi.RemoveWorkspaceMemberRequestObject{
		Wid:  ws.ID,
		Body: &openapi.RemoveWorkspaceMemberJSONRequestBody{UserId: member.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.RemoveWorkspaceMember200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestRemoveWorkspaceMember_MemberCannotRemoveOther(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member1 := testutil.CreateTestUser(t, db, "m1@test.com", "Member1")
	member2 := testutil.CreateTestUser(t, db, "m2@test.com", "Member2")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member1.ID, ws.ID, "member")
	addWorkspaceMember(t, db, member2.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member1.ID)
	resp, err := h.RemoveWorkspaceMember(ctx, openapi.RemoveWorkspaceMemberRequestObject{
		Wid:  ws.ID,
		Body: &openapi.RemoveWorkspaceMemberJSONRequestBody{UserId: member2.ID},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.RemoveWorkspaceMember403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateWorkspaceMemberRole_Success(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: member.ID,
			Role:   openapi.WorkspaceRole("admin"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestUpdateWorkspaceMemberRole_CannotChangeOwner(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")

	// Admin tries to change owner's role
	ctx := ctxWithUser(t, h, admin.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: owner.ID,
			Role:   openapi.WorkspaceRole("member"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateWorkspaceMemberRole_PromoteToOwner(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: member.ID,
			Role:   openapi.WorkspaceRole("owner"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}

	// Verify the member is now an owner
	m, _ := h.workspaceRepo.GetMembership(context.Background(), member.ID, ws.ID)
	if m.Role != "owner" {
		t.Errorf("role = %q, want %q", m.Role, "owner")
	}
}

func TestUpdateWorkspaceMemberRole_AdminCannotPromoteToOwner(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	admin := testutil.CreateTestUser(t, db, "admin@test.com", "Admin")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, admin.ID, ws.ID, "admin")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, admin.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: member.ID,
			Role:   openapi.WorkspaceRole("owner"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateWorkspaceMemberRole_SelfDemoteFromOwner(t *testing.T) {
	h, db := testHandler(t)

	owner1 := testutil.CreateTestUser(t, db, "owner1@test.com", "Owner1")
	owner2 := testutil.CreateTestUser(t, db, "owner2@test.com", "Owner2")
	ws := testutil.CreateTestWorkspace(t, db, owner1.ID, "WS")
	addWorkspaceMember(t, db, owner2.ID, ws.ID, "owner")

	// Owner1 demotes themselves to admin — should succeed because owner2 exists
	ctx := ctxWithUser(t, h, owner1.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: owner1.ID,
			Role:   openapi.WorkspaceRole("admin"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}

	m, _ := h.workspaceRepo.GetMembership(context.Background(), owner1.ID, ws.ID)
	if m.Role != "admin" {
		t.Errorf("role = %q, want %q", m.Role, "admin")
	}
}

func TestUpdateWorkspaceMemberRole_LastOwnerCannotDemoteSelf(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: owner.ID,
			Role:   openapi.WorkspaceRole("admin"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestUpdateWorkspaceMemberRole_CannotDemoteAnotherOwner(t *testing.T) {
	h, db := testHandler(t)

	owner1 := testutil.CreateTestUser(t, db, "owner1@test.com", "Owner1")
	owner2 := testutil.CreateTestUser(t, db, "owner2@test.com", "Owner2")
	ws := testutil.CreateTestWorkspace(t, db, owner1.ID, "WS")
	addWorkspaceMember(t, db, owner2.ID, ws.ID, "owner")

	// Owner1 tries to demote owner2 — should be forbidden
	ctx := ctxWithUser(t, h, owner1.ID)
	resp, err := h.UpdateWorkspaceMemberRole(ctx, openapi.UpdateWorkspaceMemberRoleRequestObject{
		Wid: ws.ID,
		Body: &openapi.UpdateWorkspaceMemberRoleJSONRequestBody{
			UserId: owner2.ID,
			Role:   openapi.WorkspaceRole("admin"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.UpdateWorkspaceMemberRole403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestCreateWorkspaceInvite_Success(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.CreateWorkspaceInvite(ctx, openapi.CreateWorkspaceInviteRequestObject{
		Wid: ws.ID,
		Body: &openapi.CreateWorkspaceInviteJSONRequestBody{
			Role: openapi.WorkspaceRole("member"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.CreateWorkspaceInvite200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if r.Invite == nil {
		t.Fatal("expected invite in response")
	}
	if r.Invite.Code == "" {
		t.Error("expected invite code to be set")
	}
}

func TestGetWorkspace_NotMember(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	outsider := testutil.CreateTestUser(t, db, "outsider@test.com", "Outsider")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, outsider.ID)
	resp, err := h.GetWorkspace(ctx, openapi.GetWorkspaceRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.GetWorkspace404JSONResponse); !ok {
		t.Fatalf("expected 404 response, got %T", resp)
	}
}

func TestGetWorkspaceNotifications_Success(t *testing.T) {
	h, db := testHandler(t)

	user := testutil.CreateTestUser(t, db, "user@test.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "WS")
	ch := testutil.CreateTestChannel(t, db, ws.ID, user.ID, "general", "public")

	// Add an unread message from another user
	other := testutil.CreateTestUser(t, db, "other@test.com", "Other")
	testutil.CreateTestMessage(t, db, ch.ID, other.ID, "Hello")

	ctx := ctxWithUser(t, h, user.ID)
	resp, err := h.GetWorkspaceNotifications(ctx, openapi.GetWorkspaceNotificationsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := resp.(openapi.GetWorkspaceNotifications200JSONResponse)
	if !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
	if len(r.Workspaces) == 0 {
		t.Fatal("expected at least one workspace summary")
	}

	// Find our workspace
	var found bool
	for _, summary := range r.Workspaces {
		if summary.WorkspaceId == ws.ID {
			found = true
			if summary.UnreadCount != 1 {
				t.Errorf("unread_count = %d, want 1", summary.UnreadCount)
			}
			// No mentions, default notify level = mentions, so notification_count = 0
			if summary.NotificationCount != 0 {
				t.Errorf("notification_count = %d, want 0", summary.NotificationCount)
			}
		}
	}
	if !found {
		t.Error("workspace not found in summaries")
	}
}

func TestGetWorkspaceNotifications_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	resp, err := h.GetWorkspaceNotifications(ctx, openapi.GetWorkspaceNotificationsRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.GetWorkspaceNotifications401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}

func TestLeaveWorkspace_MemberSuccess(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	// Also add member to a channel to verify channel membership cleanup
	ch := testutil.CreateTestChannel(t, db, ws.ID, owner.ID, "general", "public")
	addChannelMember(t, db, member.ID, ch.ID, nil)

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.LeaveWorkspace(ctx, openapi.LeaveWorkspaceRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.LeaveWorkspace200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}

	// Verify workspace membership is gone
	_, err = h.workspaceRepo.GetMembership(context.Background(), member.ID, ws.ID)
	if err == nil {
		t.Error("expected workspace membership to be removed")
	}

	// Verify channel membership is gone
	_, err = h.channelRepo.GetMembership(context.Background(), member.ID, ch.ID)
	if err == nil {
		t.Error("expected channel membership to be removed")
	}
}

func TestLeaveWorkspace_LastOwnerForbidden(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, owner.ID)
	resp, err := h.LeaveWorkspace(ctx, openapi.LeaveWorkspaceRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.LeaveWorkspace403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}

func TestLeaveWorkspace_OwnerAllowedWhenMultipleOwners(t *testing.T) {
	h, db := testHandler(t)

	owner1 := testutil.CreateTestUser(t, db, "owner1@test.com", "Owner1")
	owner2 := testutil.CreateTestUser(t, db, "owner2@test.com", "Owner2")
	ws := testutil.CreateTestWorkspace(t, db, owner1.ID, "WS")
	addWorkspaceMember(t, db, owner2.ID, ws.ID, "owner")

	ctx := ctxWithUser(t, h, owner1.ID)
	resp, err := h.LeaveWorkspace(ctx, openapi.LeaveWorkspaceRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.LeaveWorkspace200JSONResponse); !ok {
		t.Fatalf("expected 200 response, got %T", resp)
	}
}

func TestLeaveWorkspace_NotMember(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	outsider := testutil.CreateTestUser(t, db, "outsider@test.com", "Outsider")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")

	ctx := ctxWithUser(t, h, outsider.ID)
	resp, err := h.LeaveWorkspace(ctx, openapi.LeaveWorkspaceRequestObject{
		Wid: ws.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.LeaveWorkspace404JSONResponse); !ok {
		t.Fatalf("expected 404 response, got %T", resp)
	}
}

func TestLeaveWorkspace_Unauthenticated(t *testing.T) {
	h, _ := testHandler(t)

	ctx := context.Background()
	resp, err := h.LeaveWorkspace(ctx, openapi.LeaveWorkspaceRequestObject{
		Wid: "any-workspace-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.LeaveWorkspace401JSONResponse); !ok {
		t.Fatalf("expected 401 response, got %T", resp)
	}
}

func TestCreateWorkspaceInvite_NotAdmin(t *testing.T) {
	h, db := testHandler(t)

	owner := testutil.CreateTestUser(t, db, "owner@test.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@test.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "WS")
	addWorkspaceMember(t, db, member.ID, ws.ID, "member")

	ctx := ctxWithUser(t, h, member.ID)
	resp, err := h.CreateWorkspaceInvite(ctx, openapi.CreateWorkspaceInviteRequestObject{
		Wid: ws.ID,
		Body: &openapi.CreateWorkspaceInviteJSONRequestBody{
			Role: openapi.WorkspaceRole("member"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.CreateWorkspaceInvite403JSONResponse); !ok {
		t.Fatalf("expected 403 response, got %T", resp)
	}
}
