package moderation

import (
	"context"
	"testing"
	"time"

	"github.com/enzyme/api/internal/testutil"
)

func TestCreateBan(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ban := &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
	}

	err := repo.CreateBan(ctx, ban)
	if err != nil {
		t.Fatalf("CreateBan() error = %v", err)
	}

	if ban.ID == "" {
		t.Error("expected non-empty ID")
	}
	if ban.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCreateBan_WithReasonAndExpiry(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	reason := "spamming"
	expires := time.Now().Add(24 * time.Hour)
	ban := &Ban{
		WorkspaceID:  ws.ID,
		UserID:       user.ID,
		BannedBy:     &owner.ID,
		Reason:       &reason,
		HideMessages: true,
		ExpiresAt:    &expires,
	}

	err := repo.CreateBan(ctx, ban)
	if err != nil {
		t.Fatalf("CreateBan() error = %v", err)
	}

	// Verify the ban can be retrieved with all fields
	got, err := repo.GetActiveBan(ctx, ws.ID, user.ID)
	if err != nil {
		t.Fatalf("GetActiveBan() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil ban")
	}
	if got.Reason == nil || *got.Reason != reason {
		t.Errorf("Reason = %v, want %q", got.Reason, reason)
	}
	if !got.HideMessages {
		t.Error("expected HideMessages = true")
	}
	if got.ExpiresAt == nil {
		t.Error("expected non-nil ExpiresAt")
	}
}

func TestCreateBan_Duplicate(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ban := &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
	}

	repo.CreateBan(ctx, ban)

	// Second ban for same user+workspace should fail
	ban2 := &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
	}
	err := repo.CreateBan(ctx, ban2)
	if err != ErrAlreadyBanned {
		t.Errorf("CreateBan() error = %v, want %v", err, ErrAlreadyBanned)
	}
}

func TestDeleteBan(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	ban := &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
	}
	repo.CreateBan(ctx, ban)

	err := repo.DeleteBan(ctx, ws.ID, user.ID)
	if err != nil {
		t.Fatalf("DeleteBan() error = %v", err)
	}

	// Verify ban is gone
	got, err := repo.GetActiveBan(ctx, ws.ID, user.ID)
	if err != nil {
		t.Fatalf("GetActiveBan() error = %v", err)
	}
	if got != nil {
		t.Error("expected nil ban after deletion")
	}
}

func TestDeleteBan_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.DeleteBan(ctx, "nonexistent-ws", "nonexistent-user")
	if err != ErrBanNotFound {
		t.Errorf("DeleteBan() error = %v, want %v", err, ErrBanNotFound)
	}
}

func TestGetActiveBan_Expired(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Create an expired ban by inserting directly
	expired := time.Now().Add(-1 * time.Hour)
	ban := &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
		ExpiresAt:   &expired,
	}
	repo.CreateBan(ctx, ban)

	// GetActiveBan should not return expired bans
	got, err := repo.GetActiveBan(ctx, ws.ID, user.ID)
	if err != nil {
		t.Fatalf("GetActiveBan() error = %v", err)
	}
	if got != nil {
		t.Error("expected nil for expired ban")
	}
}

func TestCreateBan_AfterExpired(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Create an expired ban
	expired := time.Now().Add(-1 * time.Hour)
	repo.CreateBan(ctx, &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
		ExpiresAt:   &expired,
	})

	// Should be able to ban again after expiry
	reason := "second ban"
	err := repo.CreateBan(ctx, &Ban{
		WorkspaceID: ws.ID,
		UserID:      user.ID,
		BannedBy:    &owner.ID,
		Reason:      &reason,
	})
	if err != nil {
		t.Fatalf("CreateBan() after expired ban should succeed, got error: %v", err)
	}

	// The new ban should be active
	got, err := repo.GetActiveBan(ctx, ws.ID, user.ID)
	if err != nil {
		t.Fatalf("GetActiveBan() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected active ban after re-ban")
	}
	if got.Reason == nil || *got.Reason != "second ban" {
		t.Errorf("expected reason 'second ban', got %v", got.Reason)
	}
}

func TestGetActiveBan_NoBan(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	got, err := repo.GetActiveBan(ctx, "ws-id", "user-id")
	if err != nil {
		t.Fatalf("GetActiveBan() error = %v", err)
	}
	if got != nil {
		t.Error("expected nil when no ban exists")
	}
}

func TestListActiveBans(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User One")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User Two")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	repo.CreateBan(ctx, &Ban{WorkspaceID: ws.ID, UserID: user1.ID, BannedBy: &owner.ID})
	repo.CreateBan(ctx, &Ban{WorkspaceID: ws.ID, UserID: user2.ID, BannedBy: &owner.ID})

	bans, hasMore, _, err := repo.ListActiveBans(ctx, ws.ID, "", 50)
	if err != nil {
		t.Fatalf("ListActiveBans() error = %v", err)
	}
	if len(bans) != 2 {
		t.Fatalf("len(bans) = %d, want 2", len(bans))
	}
	if hasMore {
		t.Error("expected hasMore = false")
	}

	// Verify user display info is populated
	for _, b := range bans {
		if b.UserDisplayName == "" {
			t.Error("expected non-empty UserDisplayName")
		}
		if b.BannedByName == nil || *b.BannedByName == "" {
			t.Error("expected non-empty BannedByName")
		}
	}
}

func TestListActiveBans_Pagination(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Create 3 banned users
	for i := 0; i < 3; i++ {
		u := testutil.CreateTestUser(t, db, "user"+string(rune('a'+i))+"@example.com", "User")
		repo.CreateBan(ctx, &Ban{WorkspaceID: ws.ID, UserID: u.ID, BannedBy: &owner.ID})
	}

	// Fetch page 1 with limit 2
	bans, hasMore, cursor, err := repo.ListActiveBans(ctx, ws.ID, "", 2)
	if err != nil {
		t.Fatalf("ListActiveBans() page 1 error = %v", err)
	}
	if len(bans) != 2 {
		t.Fatalf("page 1 len(bans) = %d, want 2", len(bans))
	}
	if !hasMore {
		t.Error("expected hasMore = true for page 1")
	}
	if cursor == "" {
		t.Error("expected non-empty cursor")
	}

	// Fetch page 2
	bans2, hasMore2, _, err := repo.ListActiveBans(ctx, ws.ID, cursor, 2)
	if err != nil {
		t.Fatalf("ListActiveBans() page 2 error = %v", err)
	}
	if len(bans2) != 1 {
		t.Fatalf("page 2 len(bans) = %d, want 1", len(bans2))
	}
	if hasMore2 {
		t.Error("expected hasMore = false for page 2")
	}
}

func TestListActiveBans_ExcludesExpired(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user1 := testutil.CreateTestUser(t, db, "user1@example.com", "User One")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User Two")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Active ban
	repo.CreateBan(ctx, &Ban{WorkspaceID: ws.ID, UserID: user1.ID, BannedBy: &owner.ID})

	// Expired ban
	expired := time.Now().Add(-1 * time.Hour)
	repo.CreateBan(ctx, &Ban{WorkspaceID: ws.ID, UserID: user2.ID, BannedBy: &owner.ID, ExpiresAt: &expired})

	bans, _, _, err := repo.ListActiveBans(ctx, ws.ID, "", 50)
	if err != nil {
		t.Fatalf("ListActiveBans() error = %v", err)
	}
	if len(bans) != 1 {
		t.Fatalf("len(bans) = %d, want 1 (only active)", len(bans))
	}
	if bans[0].UserID != user1.ID {
		t.Errorf("expected active ban user %q, got %q", user1.ID, bans[0].UserID)
	}
}

// --- Block Tests (workspace-scoped) ---

func TestCreateBlock(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	err := repo.CreateBlock(ctx, ws.ID, owner.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateBlock() error = %v", err)
	}

	// Verify block exists via GetBlockedUserIDs
	blocked, err := repo.GetBlockedUserIDs(ctx, ws.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetBlockedUserIDs() error = %v", err)
	}
	if !blocked[user.ID] {
		t.Error("expected user to be in blocked set")
	}
}

func TestCreateBlock_Idempotent(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	repo.CreateBlock(ctx, ws.ID, owner.ID, user.ID)

	// Second block should succeed (idempotent)
	err := repo.CreateBlock(ctx, ws.ID, owner.ID, user.ID)
	if err != nil {
		t.Errorf("CreateBlock() duplicate error = %v, want nil (idempotent)", err)
	}
}

func TestCreateBlock_WorkspaceScoped(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws1 := testutil.CreateTestWorkspace(t, db, owner.ID, "Workspace 1")
	ws2 := testutil.CreateTestWorkspace(t, db, owner.ID, "Workspace 2")

	// Block in workspace 1
	repo.CreateBlock(ctx, ws1.ID, owner.ID, user.ID)

	// Block should exist in workspace 1
	blocked, _ := repo.GetBlockedUserIDs(ctx, ws1.ID, owner.ID)
	if !blocked[user.ID] {
		t.Error("expected block in workspace 1")
	}

	// Block should NOT exist in workspace 2
	blocked2, _ := repo.GetBlockedUserIDs(ctx, ws2.ID, owner.ID)
	if blocked2[user.ID] {
		t.Error("expected no block in workspace 2")
	}
}

func TestDeleteBlock(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	repo.CreateBlock(ctx, ws.ID, owner.ID, user.ID)

	err := repo.DeleteBlock(ctx, ws.ID, owner.ID, user.ID)
	if err != nil {
		t.Fatalf("DeleteBlock() error = %v", err)
	}

	// Verify block is gone
	blocked, _ := repo.GetBlockedUserIDs(ctx, ws.ID, owner.ID)
	if blocked[user.ID] {
		t.Error("expected user not in blocked set after deletion")
	}
}

func TestDeleteBlock_Idempotent(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Deleting a non-existent block should not error
	err := repo.DeleteBlock(ctx, ws.ID, "user-a", "user-b")
	if err != nil {
		t.Errorf("DeleteBlock() non-existent error = %v, want nil", err)
	}
}

func TestListBlocks(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User Two")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User Three")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	repo.CreateBlock(ctx, ws.ID, owner.ID, user2.ID)
	repo.CreateBlock(ctx, ws.ID, owner.ID, user3.ID)

	blocks, err := repo.ListBlocks(ctx, ws.ID, owner.ID)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}

	// Verify display info populated
	for _, b := range blocks {
		if b.DisplayName == "" {
			t.Error("expected non-empty DisplayName")
		}
		if b.WorkspaceID != ws.ID {
			t.Errorf("WorkspaceID = %q, want %q", b.WorkspaceID, ws.ID)
		}
	}
}

func TestListBlocks_ScopedToWorkspace(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws1 := testutil.CreateTestWorkspace(t, db, owner.ID, "Workspace 1")
	ws2 := testutil.CreateTestWorkspace(t, db, owner.ID, "Workspace 2")

	repo.CreateBlock(ctx, ws1.ID, owner.ID, user.ID)
	repo.CreateBlock(ctx, ws2.ID, owner.ID, user.ID)

	// List blocks in workspace 1 only
	blocks, err := repo.ListBlocks(ctx, ws1.ID, owner.ID)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1 (workspace-scoped)", len(blocks))
	}
}

func TestListBlocks_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	blocks, err := repo.ListBlocks(ctx, ws.ID, owner.ID)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("len(blocks) = %d, want 0", len(blocks))
	}
}

func TestGetBlockedUserIDs(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User Two")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User Three")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	repo.CreateBlock(ctx, ws.ID, owner.ID, user2.ID)
	repo.CreateBlock(ctx, ws.ID, owner.ID, user3.ID)

	blocked, err := repo.GetBlockedUserIDs(ctx, ws.ID, owner.ID)
	if err != nil {
		t.Fatalf("GetBlockedUserIDs() error = %v", err)
	}
	if len(blocked) != 2 {
		t.Fatalf("len(blocked) = %d, want 2", len(blocked))
	}
	if !blocked[user2.ID] {
		t.Errorf("expected user2 (%s) to be blocked", user2.ID)
	}
	if !blocked[user3.ID] {
		t.Errorf("expected user3 (%s) to be blocked", user3.ID)
	}
}

func TestGetBlockedUserIDs_DirectionMatters(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	repo.CreateBlock(ctx, ws.ID, owner.ID, user.ID)

	// owner blocked user — user should appear in owner's blocked set
	ownerBlocked, _ := repo.GetBlockedUserIDs(ctx, ws.ID, owner.ID)
	if !ownerBlocked[user.ID] {
		t.Error("expected owner->user blocked = true")
	}

	// user did NOT block owner — owner should NOT appear in user's blocked set
	userBlocked, _ := repo.GetBlockedUserIDs(ctx, ws.ID, user.ID)
	if userBlocked[owner.ID] {
		t.Error("expected user->owner blocked = false")
	}
}

func TestIsBlockedEitherDirection(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user2 := testutil.CreateTestUser(t, db, "user2@example.com", "User Two")
	user3 := testutil.CreateTestUser(t, db, "user3@example.com", "User Three")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// owner blocks user2
	repo.CreateBlock(ctx, ws.ID, owner.ID, user2.ID)

	// Either direction should return true
	blocked, _ := repo.IsBlockedEitherDirection(ctx, ws.ID, owner.ID, user2.ID)
	if !blocked {
		t.Error("expected blocked in owner->user2 direction")
	}
	blocked, _ = repo.IsBlockedEitherDirection(ctx, ws.ID, user2.ID, owner.ID)
	if !blocked {
		t.Error("expected blocked in user2->owner direction")
	}

	// No block between owner and user3
	blocked, _ = repo.IsBlockedEitherDirection(ctx, ws.ID, owner.ID, user3.ID)
	if blocked {
		t.Error("expected no block between owner and user3")
	}
}

func TestIsBlockedEitherDirection_WorkspaceScoped(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	user := testutil.CreateTestUser(t, db, "user@example.com", "User")
	ws1 := testutil.CreateTestWorkspace(t, db, owner.ID, "Workspace 1")
	ws2 := testutil.CreateTestWorkspace(t, db, owner.ID, "Workspace 2")

	// Block in workspace 1 only
	repo.CreateBlock(ctx, ws1.ID, owner.ID, user.ID)

	// Should be blocked in workspace 1
	blocked, _ := repo.IsBlockedEitherDirection(ctx, ws1.ID, owner.ID, user.ID)
	if !blocked {
		t.Error("expected blocked in workspace 1")
	}

	// Should NOT be blocked in workspace 2
	blocked, _ = repo.IsBlockedEitherDirection(ctx, ws2.ID, owner.ID, user.ID)
	if blocked {
		t.Error("expected no block in workspace 2")
	}
}

// --- Audit Log Tests ---

func TestCreateAuditLogEntry(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	entry := &AuditLogEntry{
		WorkspaceID: ws.ID,
		ActorID:     owner.ID,
		Action:      ActionUserBanned,
		TargetType:  TargetTypeUser,
		TargetID:    "some-user-id",
	}

	err := repo.CreateAuditLogEntry(ctx, entry)
	if err != nil {
		t.Fatalf("CreateAuditLogEntry() error = %v", err)
	}
	if entry.ID == "" {
		t.Error("expected non-empty ID")
	}
	if entry.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestCreateAuditLogEntryWithMetadata(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	metadata := map[string]interface{}{
		"reason":       "spamming",
		"channel_name": "general",
	}

	err := repo.CreateAuditLogEntryWithMetadata(ctx, ws.ID, owner.ID, ActionUserBanned, TargetTypeUser, "target-id", metadata)
	if err != nil {
		t.Fatalf("CreateAuditLogEntryWithMetadata() error = %v", err)
	}

	// Verify via ListAuditLog
	entries, _, _, err := repo.ListAuditLog(ctx, ws.ID, "", 50)
	if err != nil {
		t.Fatalf("ListAuditLog() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Metadata == nil {
		t.Fatal("expected non-nil Metadata")
	}
	if entries[0].Action != ActionUserBanned {
		t.Errorf("Action = %q, want %q", entries[0].Action, ActionUserBanned)
	}
}

func TestListAuditLog(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Create multiple audit log entries
	for _, action := range []string{ActionUserBanned, ActionUserUnbanned, ActionMessageDeleted} {
		repo.CreateAuditLogEntry(ctx, &AuditLogEntry{
			WorkspaceID: ws.ID,
			ActorID:     owner.ID,
			Action:      action,
			TargetType:  TargetTypeUser,
			TargetID:    "target-id",
		})
	}

	entries, hasMore, _, err := repo.ListAuditLog(ctx, ws.ID, "", 50)
	if err != nil {
		t.Fatalf("ListAuditLog() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	if hasMore {
		t.Error("expected hasMore = false")
	}

	// Verify actor display info
	for _, e := range entries {
		if e.ActorDisplayName == "" {
			t.Error("expected non-empty ActorDisplayName")
		}
	}

	// Verify ordering (newest first via ULID DESC)
	if entries[0].Action != ActionMessageDeleted {
		t.Errorf("first entry action = %q, want %q (newest first)", entries[0].Action, ActionMessageDeleted)
	}
}

func TestListAuditLog_Pagination(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	// Create 5 entries
	for i := 0; i < 5; i++ {
		repo.CreateAuditLogEntry(ctx, &AuditLogEntry{
			WorkspaceID: ws.ID,
			ActorID:     owner.ID,
			Action:      ActionUserBanned,
			TargetType:  TargetTypeUser,
			TargetID:    "target-id",
		})
	}

	// Page 1
	entries, hasMore, cursor, err := repo.ListAuditLog(ctx, ws.ID, "", 3)
	if err != nil {
		t.Fatalf("ListAuditLog() page 1 error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("page 1 len = %d, want 3", len(entries))
	}
	if !hasMore {
		t.Error("expected hasMore = true for page 1")
	}

	// Page 2
	entries2, hasMore2, _, err := repo.ListAuditLog(ctx, ws.ID, cursor, 3)
	if err != nil {
		t.Fatalf("ListAuditLog() page 2 error = %v", err)
	}
	if len(entries2) != 2 {
		t.Fatalf("page 2 len = %d, want 2", len(entries2))
	}
	if hasMore2 {
		t.Error("expected hasMore = false for page 2")
	}
}

func TestListAuditLog_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	owner := testutil.CreateTestUser(t, db, "owner@example.com", "Owner")
	ws := testutil.CreateTestWorkspace(t, db, owner.ID, "Test WS")

	entries, hasMore, cursor, err := repo.ListAuditLog(ctx, ws.ID, "", 50)
	if err != nil {
		t.Fatalf("ListAuditLog() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
	if hasMore {
		t.Error("expected hasMore = false")
	}
	if cursor != "" {
		t.Error("expected empty cursor")
	}
}
