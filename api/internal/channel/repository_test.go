package channel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/feather/api/internal/testutil"
	"github.com/oklog/ulid/v2"
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

func TestRepository_ConvertToChannel(t *testing.T) {
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

	// Convert to channel
	desc := "A project channel"
	converted, err := repo.ConvertToChannel(ctx, dm.ID, "project-x", &desc, user1.ID, TypePrivate)
	if err != nil {
		t.Fatalf("ConvertToChannel() error = %v", err)
	}

	// Verify type changed to private
	if converted.Type != TypePrivate {
		t.Errorf("Type = %q, want %q", converted.Type, TypePrivate)
	}

	// Verify name was set
	if converted.Name != "project-x" {
		t.Errorf("Name = %q, want %q", converted.Name, "project-x")
	}

	// Verify description was set
	if converted.Description == nil || *converted.Description != desc {
		t.Errorf("Description = %v, want %q", converted.Description, desc)
	}

	// Verify DM hash was cleared
	if converted.DMParticipantHash != nil {
		t.Errorf("DMParticipantHash = %v, want nil", converted.DMParticipantHash)
	}

	// Verify created_by was set
	if converted.CreatedBy == nil || *converted.CreatedBy != user1.ID {
		t.Errorf("CreatedBy = %v, want %q", converted.CreatedBy, user1.ID)
	}

	// Verify converter was promoted to admin
	membership, err := repo.GetMembership(ctx, user1.ID, dm.ID)
	if err != nil {
		t.Fatalf("GetMembership() error = %v", err)
	}
	if membership.ChannelRole == nil || *membership.ChannelRole != ChannelRoleAdmin {
		t.Errorf("ChannelRole = %v, want %q", membership.ChannelRole, ChannelRoleAdmin)
	}

	// Verify other members are still present
	_, err = repo.GetMembership(ctx, user2.ID, dm.ID)
	if err != nil {
		t.Errorf("GetMembership() for user2 error = %v", err)
	}
	_, err = repo.GetMembership(ctx, user3.ID, dm.ID)
	if err != nil {
		t.Errorf("GetMembership() for user3 error = %v", err)
	}
}

func TestRepository_ConvertToChannel_NoDescription(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User 3")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	dm, _ := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID, user3.ID})

	// Convert without description
	converted, err := repo.ConvertToChannel(ctx, dm.ID, "my-channel", nil, user1.ID, TypePublic)
	if err != nil {
		t.Fatalf("ConvertToChannel() error = %v", err)
	}

	if converted.Type != TypePublic {
		t.Errorf("Type = %q, want %q", converted.Type, TypePublic)
	}
	if converted.Name != "my-channel" {
		t.Errorf("Name = %q, want %q", converted.Name, "my-channel")
	}
}

func TestRepository_GetByWorkspaceAndName(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create a public channel
	ch := testutil.CreateTestChannel(t, db, ws.ID, user1.ID, "general", TypePublic)

	// Should find it by name
	found, err := repo.GetByWorkspaceAndName(ctx, ws.ID, "general")
	if err != nil {
		t.Fatalf("GetByWorkspaceAndName() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByWorkspaceAndName() returned nil, want channel")
	}
	if found.ID != ch.ID {
		t.Errorf("ID = %q, want %q", found.ID, ch.ID)
	}

	// Should return nil for non-existent name
	notFound, err := repo.GetByWorkspaceAndName(ctx, ws.ID, "nonexistent")
	if err != nil {
		t.Fatalf("GetByWorkspaceAndName() error = %v", err)
	}
	if notFound != nil {
		t.Errorf("GetByWorkspaceAndName() = %v, want nil", notFound)
	}

	// Should not find DM channels by name
	repo.CreateDM(ctx, ws.ID, []string{user1.ID, testutil.CreateTestUser(t, db, "user2@example.com", "User 2").ID})
	dmFound, err := repo.GetByWorkspaceAndName(ctx, ws.ID, "Direct Message")
	if err != nil {
		t.Fatalf("GetByWorkspaceAndName() error = %v", err)
	}
	if dmFound != nil {
		t.Errorf("GetByWorkspaceAndName() should not find DM channels, got %v", dmFound)
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

// createMessageWithMentions creates a message with specified mentions JSON array
func createMessageWithMentions(t *testing.T, db *sql.DB, channelID, userID, content string, mentions []string) string {
	t.Helper()

	id := ulid.Make().String()
	now := time.Now().UTC()

	mentionsJSON, err := json.Marshal(mentions)
	if err != nil {
		t.Fatalf("marshaling mentions: %v", err)
	}

	_, err = db.ExecContext(context.Background(), `
		INSERT INTO messages (id, channel_id, user_id, content, mentions, reply_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`, id, channelID, userID, content, string(mentionsJSON), now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating test message with mentions: %v", err)
	}

	return id
}

// setNotificationPreference sets a notification preference for a user/channel
func setNotificationPreference(t *testing.T, db *sql.DB, userID, channelID, notifyLevel string) {
	t.Helper()

	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO notification_preferences (id, user_id, channel_id, notify_level, email_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
	`, id, userID, channelID, notifyLevel, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("setting notification preference: %v", err)
	}
}

func TestRepository_ListForWorkspace_NotificationCount_DMsAlwaysNotify(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create DM
	dm, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("CreateDM() error = %v", err)
	}

	// Add messages (no mentions needed for DMs)
	testutil.CreateTestMessage(t, db, dm.ID, user2.ID, "Hello")
	testutil.CreateTestMessage(t, db, dm.ID, user2.ID, "How are you?")

	channels, err := repo.ListForWorkspace(ctx, ws.ID, user1.ID)
	if err != nil {
		t.Fatalf("ListForWorkspace() error = %v", err)
	}

	var dmChannel *ChannelWithMembership
	for i, c := range channels {
		if c.ID == dm.ID {
			dmChannel = &channels[i]
			break
		}
	}

	if dmChannel == nil {
		t.Fatal("DM channel not found in results")
	}

	// DMs should always have notification_count == unread_count
	if dmChannel.NotificationCount != dmChannel.UnreadCount {
		t.Errorf("DM NotificationCount = %d, want %d (same as UnreadCount)", dmChannel.NotificationCount, dmChannel.UnreadCount)
	}
	if dmChannel.NotificationCount != 2 {
		t.Errorf("DM NotificationCount = %d, want 2", dmChannel.NotificationCount)
	}
}

func TestRepository_ListForWorkspace_NotificationCount_MentionsDefault(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	ch := testutil.CreateTestChannel(t, db, ws.ID, user1.ID, "general", "public")

	// Message 1: mentions user1
	createMessageWithMentions(t, db, ch.ID, user2.ID, "Hey @User 1", []string{user1.ID})
	// Message 2: no mentions
	createMessageWithMentions(t, db, ch.ID, user2.ID, "Just chatting", []string{})
	// Message 3: mentions @channel
	createMessageWithMentions(t, db, ch.ID, user2.ID, "Hey @channel", []string{"@channel"})

	channels, err := repo.ListForWorkspace(ctx, ws.ID, user1.ID)
	if err != nil {
		t.Fatalf("ListForWorkspace() error = %v", err)
	}

	var found *ChannelWithMembership
	for i, c := range channels {
		if c.ID == ch.ID {
			found = &channels[i]
			break
		}
	}

	if found == nil {
		t.Fatal("channel not found in results")
	}

	// Default (no preference row) = 'mentions': should count messages with user mention or @channel
	if found.UnreadCount != 3 {
		t.Errorf("UnreadCount = %d, want 3", found.UnreadCount)
	}
	if found.NotificationCount != 2 {
		t.Errorf("NotificationCount = %d, want 2 (1 direct mention + 1 @channel)", found.NotificationCount)
	}
}

func TestRepository_ListForWorkspace_NotificationCount_NotifyAll(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	ch := testutil.CreateTestChannel(t, db, ws.ID, user1.ID, "general", "public")

	// Set notify_level = 'all'
	setNotificationPreference(t, db, user1.ID, ch.ID, "all")

	// Messages without mentions
	testutil.CreateTestMessage(t, db, ch.ID, user2.ID, "Hello")
	testutil.CreateTestMessage(t, db, ch.ID, user2.ID, "World")

	channels, err := repo.ListForWorkspace(ctx, ws.ID, user1.ID)
	if err != nil {
		t.Fatalf("ListForWorkspace() error = %v", err)
	}

	var found *ChannelWithMembership
	for i, c := range channels {
		if c.ID == ch.ID {
			found = &channels[i]
			break
		}
	}

	if found == nil {
		t.Fatal("channel not found in results")
	}

	// notify_level='all': notification_count should match unread_count
	if found.NotificationCount != found.UnreadCount {
		t.Errorf("NotificationCount = %d, want %d (same as UnreadCount)", found.NotificationCount, found.UnreadCount)
	}
	if found.NotificationCount != 2 {
		t.Errorf("NotificationCount = %d, want 2", found.NotificationCount)
	}
}

func TestRepository_ListForWorkspace_NotificationCount_NotifyNone(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	ch := testutil.CreateTestChannel(t, db, ws.ID, user1.ID, "general", "public")

	// Set notify_level = 'none'
	setNotificationPreference(t, db, user1.ID, ch.ID, "none")

	// Messages with mentions
	createMessageWithMentions(t, db, ch.ID, user2.ID, "Hey @User 1", []string{user1.ID})
	testutil.CreateTestMessage(t, db, ch.ID, user2.ID, "Hello")

	channels, err := repo.ListForWorkspace(ctx, ws.ID, user1.ID)
	if err != nil {
		t.Fatalf("ListForWorkspace() error = %v", err)
	}

	var found *ChannelWithMembership
	for i, c := range channels {
		if c.ID == ch.ID {
			found = &channels[i]
			break
		}
	}

	if found == nil {
		t.Fatal("channel not found in results")
	}

	// notify_level='none': notification_count should be 0
	if found.UnreadCount != 2 {
		t.Errorf("UnreadCount = %d, want 2", found.UnreadCount)
	}
	if found.NotificationCount != 0 {
		t.Errorf("NotificationCount = %d, want 0", found.NotificationCount)
	}
}

func TestRepository_ListForWorkspace_NotificationCount_MentionsExplicit(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	ch := testutil.CreateTestChannel(t, db, ws.ID, user1.ID, "general", "public")

	// Explicit mentions preference (same as default for channels)
	setNotificationPreference(t, db, user1.ID, ch.ID, "mentions")

	// Messages: 1 with direct mention, 1 with @here, 1 with @everyone, 1 without
	createMessageWithMentions(t, db, ch.ID, user2.ID, "Hey @User 1", []string{user1.ID})
	createMessageWithMentions(t, db, ch.ID, user2.ID, "@here check this", []string{"@here"})
	createMessageWithMentions(t, db, ch.ID, user2.ID, "@everyone important", []string{"@everyone"})
	createMessageWithMentions(t, db, ch.ID, user2.ID, "just chatting", []string{})

	channels, err := repo.ListForWorkspace(ctx, ws.ID, user1.ID)
	if err != nil {
		t.Fatalf("ListForWorkspace() error = %v", err)
	}

	var found *ChannelWithMembership
	for i, c := range channels {
		if c.ID == ch.ID {
			found = &channels[i]
			break
		}
	}

	if found == nil {
		t.Fatal("channel not found in results")
	}

	if found.UnreadCount != 4 {
		t.Errorf("UnreadCount = %d, want 4", found.UnreadCount)
	}
	// Should count: direct mention + @here + @everyone = 3
	if found.NotificationCount != 3 {
		t.Errorf("NotificationCount = %d, want 3 (1 direct + 1 @here + 1 @everyone)", found.NotificationCount)
	}
}

func TestRepository_ListRecentDMs_NotificationCount(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User 1")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User 2")
	ws := testutil.CreateTestWorkspace(t, db, user1.ID, "Test WS")

	// Create DM
	dm, err := repo.CreateDM(ctx, ws.ID, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("CreateDM() error = %v", err)
	}

	// Add messages (DMs always notify regardless of mentions)
	testutil.CreateTestMessage(t, db, dm.ID, user2.ID, "Hello")
	testutil.CreateTestMessage(t, db, dm.ID, user2.ID, "How are you?")
	testutil.CreateTestMessage(t, db, dm.ID, user2.ID, "Are you there?")

	channels, err := repo.ListRecentDMs(ctx, ws.ID, user1.ID, time.Now().Add(-24*time.Hour), 50)
	if err != nil {
		t.Fatalf("ListRecentDMs() error = %v", err)
	}

	var dmChannel *ChannelWithMembership
	for i, c := range channels {
		if c.ID == dm.ID {
			dmChannel = &channels[i]
			break
		}
	}

	if dmChannel == nil {
		t.Fatal("DM channel not found in results")
	}

	if dmChannel.UnreadCount != 3 {
		t.Errorf("UnreadCount = %d, want 3", dmChannel.UnreadCount)
	}
	// DMs should always have notification_count == unread_count
	if dmChannel.NotificationCount != dmChannel.UnreadCount {
		t.Errorf("NotificationCount = %d, want %d (same as UnreadCount)", dmChannel.NotificationCount, dmChannel.UnreadCount)
	}
}
