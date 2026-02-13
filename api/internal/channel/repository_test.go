package channel

import (
	"context"
	"errors"
	"testing"

	"github.com/feather/api/internal/testutil"
)

func TestRepository_Create(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{
		WorkspaceID: ws.ID,
		Name:        "general",
		Type:        TypePublic,
	}

	err := repo.Create(ctx, ch, owner.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if ch.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ch.CreatedBy == nil || *ch.CreatedBy != owner.ID {
		t.Error("expected CreatedBy to be set to creator ID")
	}
}

func TestRepository_Create_AddsCreatorAsMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{
		WorkspaceID: ws.ID,
		Name:        "general",
		Type:        TypePublic,
	}

	repo.Create(ctx, ch, owner.ID)

	// Verify creator was added as admin member
	membership, err := repo.GetMembership(ctx, owner.ID, ch.ID)
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}

	if membership.ChannelRole == nil || *membership.ChannelRole != ChannelRoleAdmin {
		t.Errorf("ChannelRole = %v, want %q", membership.ChannelRole, ChannelRoleAdmin)
	}
}

func TestRepository_GetByID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	created := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, created, owner.ID)

	ch, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if ch.ID != created.ID {
		t.Errorf("ID = %q, want %q", ch.ID, created.ID)
	}
	if ch.Name != "general" {
		t.Errorf("Name = %q, want %q", ch.Name, "general")
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, ErrChannelNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrChannelNotFound)
	}
}

func TestRepository_Update(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	ch.Name = "updated-name"
	description := "Updated description"
	ch.Description = &description

	err := repo.Update(ctx, ch)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, ch.ID)
	if updated.Name != "updated-name" {
		t.Errorf("Name = %q, want %q", updated.Name, "updated-name")
	}
	if updated.Description == nil || *updated.Description != description {
		t.Errorf("Description = %v, want %q", updated.Description, description)
	}
}

func TestRepository_Archive(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	err := repo.Archive(ctx, ch.ID)
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	archived, _ := repo.GetByID(ctx, ch.ID)
	if archived.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be set")
	}
}

func TestRepository_CreateDM(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	ch, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("CreateDM() error = %v", err)
	}

	if ch.Type != TypeDM {
		t.Errorf("Type = %q, want %q", ch.Type, TypeDM)
	}
	if ch.DMParticipantHash == nil {
		t.Error("expected DMParticipantHash to be set")
	}
}

func TestRepository_CreateDM_GroupDM(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User 3")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	ch, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID, user3.ID})
	if err != nil {
		t.Fatalf("CreateDM() error = %v", err)
	}

	if ch.Type != TypeGroupDM {
		t.Errorf("Type = %q, want %q", ch.Type, TypeGroupDM)
	}
}

func TestRepository_CreateDM_ReturnsExisting(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create first DM
	dm1, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("first CreateDM() error = %v", err)
	}

	// Creating again should return the same DM
	dm2, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("second CreateDM() error = %v", err)
	}

	if dm1.ID != dm2.ID {
		t.Errorf("expected same DM ID, got %q and %q", dm1.ID, dm2.ID)
	}
}

func TestRepository_CreateDM_OrderIndependent(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create with user1 first
	dm1, _ := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})

	// Create with user2 first - should return same DM
	dm2, _ := repo.CreateDM(ctx, ws.ID, []string{user2.ID, user1.ID})

	if dm1.ID != dm2.ID {
		t.Errorf("expected same DM regardless of order, got %q and %q", dm1.ID, dm2.ID)
	}
}

func TestRepository_AddMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	role := ChannelRolePoster
	m, err := repo.AddMember(ctx, member.ID, ch.ID, &role)
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}

	if m.UserID != member.ID {
		t.Errorf("UserID = %q, want %q", m.UserID, member.ID)
	}
	if m.ChannelRole == nil || *m.ChannelRole != ChannelRolePoster {
		t.Errorf("ChannelRole = %v, want %q", m.ChannelRole, ChannelRolePoster)
	}
}

func TestRepository_AddMember_AlreadyMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	// Owner is already a member from channel creation
	role := ChannelRolePoster
	_, err := repo.AddMember(ctx, owner.ID, ch.ID, &role)
	if !errors.Is(err, ErrAlreadyMember) {
		t.Errorf("AddMember() error = %v, want %v", err, ErrAlreadyMember)
	}
}

func TestRepository_AddMember_ArchivedChannel(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	repo.Archive(ctx, ch.ID)

	role := ChannelRolePoster
	_, err := repo.AddMember(ctx, member.ID, ch.ID, &role)
	if !errors.Is(err, ErrChannelArchived) {
		t.Errorf("AddMember() error = %v, want %v", err, ErrChannelArchived)
	}
}

func TestRepository_RemoveMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	role := ChannelRolePoster
	repo.AddMember(ctx, member.ID, ch.ID, &role)

	err := repo.RemoveMember(ctx, member.ID, ch.ID)
	if err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}

	_, err = repo.GetMembership(ctx, member.ID, ch.ID)
	if !errors.Is(err, ErrNotChannelMember) {
		t.Errorf("GetMembership() after removal error = %v, want %v", err, ErrNotChannelMember)
	}
}

func TestRepository_RemoveMember_CannotLeaveDM(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	dm, _ := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})

	err := repo.RemoveMember(ctx, user1.ID, dm.ID)
	if !errors.Is(err, ErrCannotLeaveChannel) {
		t.Errorf("RemoveMember() error = %v, want %v", err, ErrCannotLeaveChannel)
	}
}

func TestRepository_ListMembers(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	member := testutil.CreateTestUser(t, db, "member@example.com", "Member")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	role := ChannelRolePoster
	repo.AddMember(ctx, member.ID, ch.ID, &role)

	members, err := repo.ListMembers(ctx, ch.ID)
	if err != nil {
		t.Fatalf("ListMembers() error = %v", err)
	}

	if len(members) != 2 {
		t.Fatalf("len(members) = %d, want 2", len(members))
	}
}

func TestRepository_UpdateLastRead(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch, owner.ID)

	// Create a message to mark as read
	msg := testutil.CreateTestMessage(t, db, ch.ID, owner.ID, "Hello")

	err := repo.UpdateLastRead(ctx, owner.ID, ch.ID, msg.ID)
	if err != nil {
		t.Fatalf("UpdateLastRead() error = %v", err)
	}

	membership, _ := repo.GetMembership(ctx, owner.ID, ch.ID)
	if membership.LastReadMessageID == nil || *membership.LastReadMessageID != msg.ID {
		t.Errorf("LastReadMessageID = %v, want %q", membership.LastReadMessageID, msg.ID)
	}
}

func TestRepository_ListMemberChannelIDs(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ch1 := &Channel{WorkspaceID: ws.ID, Name: "general", Type: TypePublic}
	repo.Create(ctx, ch1, owner.ID)

	ch2 := &Channel{WorkspaceID: ws.ID, Name: "random", Type: TypePublic}
	repo.Create(ctx, ch2, owner.ID)

	channelIDs, err := repo.ListMemberChannelIDs(ctx, ws.ID, owner.ID)
	if err != nil {
		t.Fatalf("ListMemberChannelIDs() error = %v", err)
	}

	if len(channelIDs) != 2 {
		t.Fatalf("len(channelIDs) = %d, want 2", len(channelIDs))
	}

	// Verify both channels are in the list
	found := make(map[string]bool)
	for _, id := range channelIDs {
		found[id] = true
	}
	if !found[ch1.ID] || !found[ch2.ID] {
		t.Errorf("expected both channel IDs in list")
	}
}

func TestComputeDMHash_Consistent(t *testing.T) {
	hash1 := ComputeDMHash([]string{"a", "b", "c"})
	hash2 := ComputeDMHash([]string{"c", "a", "b"})
	hash3 := ComputeDMHash([]string{"b", "c", "a"})

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("ComputeDMHash should be order-independent: %q, %q, %q", hash1, hash2, hash3)
	}
}

func TestRepository_AddMemberToDM(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User 3")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create a 1:1 DM
	dm, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("CreateDM() error = %v", err)
	}
	if dm.Type != TypeDM {
		t.Fatalf("Type = %q, want %q", dm.Type, TypeDM)
	}

	// Add a third member — should convert to group_dm
	currentIDs := []string{user1.ID, user2.ID}
	updatedCh, err := repo.AddMemberToDM(ctx, dm.ID, user3.ID, currentIDs)
	if err != nil {
		t.Fatalf("AddMemberToDM() error = %v", err)
	}

	if updatedCh.Type != TypeGroupDM {
		t.Errorf("Type = %q, want %q", updatedCh.Type, TypeGroupDM)
	}

	// Verify hash was updated
	expectedHash := ComputeDMHash([]string{user1.ID, user2.ID, user3.ID})
	if updatedCh.DMParticipantHash == nil || *updatedCh.DMParticipantHash != expectedHash {
		t.Errorf("DMParticipantHash = %v, want %q", updatedCh.DMParticipantHash, expectedHash)
	}

	// Verify user3 is now a member
	_, err = repo.GetMembership(ctx, user3.ID, dm.ID)
	if err != nil {
		t.Errorf("GetMembership() for new member error = %v", err)
	}
}

func TestRepository_AddMemberToDM_AlreadyMember(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	dm, _ := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})

	_, err := repo.AddMemberToDM(ctx, dm.ID, user1.ID, []string{user1.ID, user2.ID})
	if !errors.Is(err, ErrAlreadyMember) {
		t.Errorf("AddMemberToDM() error = %v, want %v", err, ErrAlreadyMember)
	}
}

func TestRepository_LeaveGroupDM(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User 3")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create a group DM
	dm, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID, user3.ID})
	if err != nil {
		t.Fatalf("CreateDM() error = %v", err)
	}
	if dm.Type != TypeGroupDM {
		t.Fatalf("Type = %q, want %q", dm.Type, TypeGroupDM)
	}

	// User3 leaves — should convert back to dm
	err = repo.LeaveGroupDM(ctx, user3.ID, dm.ID)
	if err != nil {
		t.Fatalf("LeaveGroupDM() error = %v", err)
	}

	updatedCh, err := repo.GetByID(ctx, dm.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if updatedCh.Type != TypeDM {
		t.Errorf("Type = %q, want %q after leaving", updatedCh.Type, TypeDM)
	}

	// Verify hash matches remaining two users
	expectedHash := ComputeDMHash([]string{user1.ID, user2.ID})
	if updatedCh.DMParticipantHash == nil || *updatedCh.DMParticipantHash != expectedHash {
		t.Errorf("DMParticipantHash = %v, want %q", updatedCh.DMParticipantHash, expectedHash)
	}

	// Verify user3 is no longer a member
	_, err = repo.GetMembership(ctx, user3.ID, dm.ID)
	if !errors.Is(err, ErrNotChannelMember) {
		t.Errorf("GetMembership() for removed member error = %v, want %v", err, ErrNotChannelMember)
	}
}

func TestRepository_LeaveGroupDM_StaysGroupDMWith3Plus(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User 3")
	user4 := testutil.CreateTestUser(t, db, "user4@example.com", "User 4")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	dm, _ := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID, user3.ID, user4.ID})

	// User4 leaves — still 3 members, should stay group_dm
	err := repo.LeaveGroupDM(ctx, user4.ID, dm.ID)
	if err != nil {
		t.Fatalf("LeaveGroupDM() error = %v", err)
	}

	updatedCh, _ := repo.GetByID(ctx, dm.ID)
	if updatedCh.Type != TypeGroupDM {
		t.Errorf("Type = %q, want %q (still 3 members)", updatedCh.Type, TypeGroupDM)
	}
}

func TestRepository_RemoveMember_AllowsGroupDM(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User 3")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	dm, _ := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID, user3.ID})

	// RemoveMember should work for group_dm (delegates to LeaveGroupDM)
	err := repo.RemoveMember(ctx, user3.ID, dm.ID)
	if err != nil {
		t.Fatalf("RemoveMember() for group_dm error = %v", err)
	}

	// Verify user3 is gone
	_, err = repo.GetMembership(ctx, user3.ID, dm.ID)
	if !errors.Is(err, ErrNotChannelMember) {
		t.Errorf("expected ErrNotChannelMember, got %v", err)
	}
}
