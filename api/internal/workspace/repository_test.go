package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/feather/api/internal/testutil"
)

func TestRepository_Create(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	ws := &Workspace{
		Name:     "Test Workspace",
		Settings: "{}",
	}

	err := repo.Create(ctx, ws, owner.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if ws.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ws.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestRepository_Create_AddsOwnerMembership(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	ws := &Workspace{
		Name:     "Test Workspace",
		Settings: "{}",
	}

	err := repo.Create(ctx, ws, owner.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify owner membership was created
	membership, err := repo.GetMembership(ctx, owner.ID, ws.ID)
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}

	if membership.Role != RoleOwner {
		t.Errorf("Role = %q, want %q", membership.Role, RoleOwner)
	}
}

func TestRepository_GetByID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	created := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, created, owner.ID)

	ws, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if ws.ID != created.ID {
		t.Errorf("ID = %q, want %q", ws.ID, created.ID)
	}
	if ws.Name != "Test WS" {
		t.Errorf("Name = %q, want %q", ws.Name, "Test WS")
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrWorkspaceNotFound)
	}
}

func TestRepository_Update(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	ws := &Workspace{Name: "Update Me", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	ws.Name = "Updated Name"
	iconURL := "https://example.com/icon.png"
	ws.IconURL = &iconURL

	if err := repo.Update(ctx, ws); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, ws.ID)
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated Name")
	}
	if updated.IconURL == nil || *updated.IconURL != iconURL {
		t.Errorf("IconURL = %v, want %q", updated.IconURL, iconURL)
	}
}

func TestRepository_AddMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	m, err := repo.AddMember(ctx, member.ID, ws.ID, RoleMember)
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}

	if m.UserID != member.ID {
		t.Errorf("UserID = %q, want %q", m.UserID, member.ID)
	}
	if m.Role != RoleMember {
		t.Errorf("Role = %q, want %q", m.Role, RoleMember)
	}
}

func TestRepository_AddMember_AlreadyMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	// Owner is already a member
	_, err := repo.AddMember(ctx, owner.ID, ws.ID, RoleMember)
	if !errors.Is(err, ErrMembershipExists) {
		t.Errorf("AddMember() error = %v, want %v", err, ErrMembershipExists)
	}
}

func TestRepository_RemoveMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	repo.AddMember(ctx, member.ID, ws.ID, RoleMember)

	err := repo.RemoveMember(ctx, member.ID, ws.ID)
	if err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}

	_, err = repo.GetMembership(ctx, member.ID, ws.ID)
	if !errors.Is(err, ErrNotAMember) {
		t.Errorf("GetMembership() after removal error = %v, want %v", err, ErrNotAMember)
	}
}

func TestRepository_RemoveMember_CannotRemoveOwner(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	err := repo.RemoveMember(ctx, owner.ID, ws.ID)
	if !errors.Is(err, ErrCannotRemoveOwner) {
		t.Errorf("RemoveMember() error = %v, want %v", err, ErrCannotRemoveOwner)
	}
}

func TestRepository_UpdateMemberRole(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	repo.AddMember(ctx, member.ID, ws.ID, RoleMember)

	err := repo.UpdateMemberRole(ctx, member.ID, ws.ID, RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateMemberRole() error = %v", err)
	}

	m, _ := repo.GetMembership(ctx, member.ID, ws.ID)
	if m.Role != RoleAdmin {
		t.Errorf("Role = %q, want %q", m.Role, RoleAdmin)
	}
}

func TestRepository_ListMembers(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	repo.AddMember(ctx, member.ID, ws.ID, RoleMember)

	members, err := repo.ListMembers(ctx, ws.ID)
	if err != nil {
		t.Fatalf("ListMembers() error = %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("len(members) = %d, want 2", len(members))
	}
}

func TestRepository_CreateInvite(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	invite := &Invite{
		WorkspaceID: ws.ID,
		Role:        RoleMember,
		CreatedBy:   &owner.ID,
	}

	err := repo.CreateInvite(ctx, invite)
	if err != nil {
		t.Fatalf("CreateInvite() error = %v", err)
	}

	if invite.ID == "" {
		t.Error("expected non-empty ID")
	}
	if invite.Code == "" {
		t.Error("expected non-empty Code")
	}
}

func TestRepository_AcceptInvite(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	newMember := testutil.CreateTestUser(t, db, "new@example.com", "New Member")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	invite := &Invite{
		WorkspaceID: ws.ID,
		Role:        RoleMember,
	}
	repo.CreateInvite(ctx, invite)

	acceptedWS, err := repo.AcceptInvite(ctx, invite.Code, newMember.ID)
	if err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}

	if acceptedWS.ID != ws.ID {
		t.Errorf("workspace ID = %q, want %q", acceptedWS.ID, ws.ID)
	}

	// Verify membership was created
	m, err := repo.GetMembership(ctx, newMember.ID, ws.ID)
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}
	if m.Role != RoleMember {
		t.Errorf("Role = %q, want %q", m.Role, RoleMember)
	}
}

func TestRepository_AcceptInvite_Expired(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	newMember := testutil.CreateTestUser(t, db, "new@example.com", "New Member")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	expiredTime := time.Now().Add(-1 * time.Hour)
	invite := &Invite{
		WorkspaceID: ws.ID,
		Role:        RoleMember,
		ExpiresAt:   &expiredTime,
	}
	repo.CreateInvite(ctx, invite)

	_, err := repo.AcceptInvite(ctx, invite.Code, newMember.ID)
	if !errors.Is(err, ErrInviteExpired) {
		t.Errorf("AcceptInvite() error = %v, want %v", err, ErrInviteExpired)
	}
}

func TestRepository_AcceptInvite_MaxUsed(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member1 := testutil.CreateTestUser(t, db, "member1@example.com", "Member 1")
	member2 := testutil.CreateTestUser(t, db, "member2@example.com", "Member 2")

	ws := &Workspace{Name: "Test WS", Settings: "{}"}
	repo.Create(ctx, ws, owner.ID)

	maxUses := 1
	invite := &Invite{
		WorkspaceID: ws.ID,
		Role:        RoleMember,
		MaxUses:     &maxUses,
	}
	repo.CreateInvite(ctx, invite)

	// First use should succeed
	_, err := repo.AcceptInvite(ctx, invite.Code, member1.ID)
	if err != nil {
		t.Fatalf("first AcceptInvite() error = %v", err)
	}

	// Second use should fail
	_, err = repo.AcceptInvite(ctx, invite.Code, member2.ID)
	if !errors.Is(err, ErrInviteMaxUsed) {
		t.Errorf("second AcceptInvite() error = %v, want %v", err, ErrInviteMaxUsed)
	}
}
