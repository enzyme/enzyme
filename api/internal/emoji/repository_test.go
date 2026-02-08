package emoji

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

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	e := &CustomEmoji{
		WorkspaceID: ws.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/gif",
		SizeBytes:   2048,
		StoragePath: "/data/emojis/test.gif",
	}

	err := repo.Create(ctx, e)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestRepository_Create_DuplicateName(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	e1 := &CustomEmoji{
		WorkspaceID: ws.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/gif",
		SizeBytes:   2048,
		StoragePath: "/data/emojis/test1.gif",
	}
	if err := repo.Create(ctx, e1); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	e2 := &CustomEmoji{
		WorkspaceID: ws.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/png",
		SizeBytes:   1024,
		StoragePath: "/data/emojis/test2.png",
	}
	err := repo.Create(ctx, e2)
	if !errors.Is(err, ErrEmojiNameTaken) {
		t.Errorf("expected ErrEmojiNameTaken, got %v", err)
	}
}

func TestRepository_Create_SameNameDifferentWorkspace(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws1 := testutil.CreateTestWorkspace(t, db, user.ID, "Workspace 1")
	ws2 := testutil.CreateTestWorkspace(t, db, user.ID, "Workspace 2")

	e1 := &CustomEmoji{
		WorkspaceID: ws1.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/gif",
		SizeBytes:   2048,
		StoragePath: "/data/emojis/test1.gif",
	}
	if err := repo.Create(ctx, e1); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	e2 := &CustomEmoji{
		WorkspaceID: ws2.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/gif",
		SizeBytes:   2048,
		StoragePath: "/data/emojis/test2.gif",
	}
	if err := repo.Create(ctx, e2); err != nil {
		t.Errorf("expected no error for same name in different workspace, got %v", err)
	}
}

func TestRepository_GetByID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	e := &CustomEmoji{
		WorkspaceID: ws.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/gif",
		SizeBytes:   2048,
		StoragePath: "/data/emojis/test.gif",
	}
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != e.ID {
		t.Errorf("ID = %q, want %q", got.ID, e.ID)
	}
	if got.Name != "partyparrot" {
		t.Errorf("Name = %q, want %q", got.Name, "partyparrot")
	}
	if got.WorkspaceID != ws.ID {
		t.Errorf("WorkspaceID = %q, want %q", got.WorkspaceID, ws.ID)
	}
	if got.ContentType != "image/gif" {
		t.Errorf("ContentType = %q, want %q", got.ContentType, "image/gif")
	}
	if got.SizeBytes != 2048 {
		t.Errorf("SizeBytes = %d, want %d", got.SizeBytes, 2048)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if !errors.Is(err, ErrEmojiNotFound) {
		t.Errorf("expected ErrEmojiNotFound, got %v", err)
	}
}

func TestRepository_ListByWorkspace(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	// Create emojis in non-alphabetical order
	names := []string{"zebra", "alpha", "middle"}
	for _, name := range names {
		e := &CustomEmoji{
			WorkspaceID: ws.ID,
			Name:        name,
			CreatedBy:   user.ID,
			ContentType: "image/png",
			SizeBytes:   1024,
			StoragePath: "/data/emojis/" + name + ".png",
		}
		if err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
	}

	emojis, err := repo.ListByWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("ListByWorkspace() error = %v", err)
	}

	if len(emojis) != 3 {
		t.Fatalf("expected 3 emojis, got %d", len(emojis))
	}

	// Verify sorted by name ASC
	if emojis[0].Name != "alpha" {
		t.Errorf("emojis[0].Name = %q, want %q", emojis[0].Name, "alpha")
	}
	if emojis[1].Name != "middle" {
		t.Errorf("emojis[1].Name = %q, want %q", emojis[1].Name, "middle")
	}
	if emojis[2].Name != "zebra" {
		t.Errorf("emojis[2].Name = %q, want %q", emojis[2].Name, "zebra")
	}
}

func TestRepository_ListByWorkspace_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	emojis, err := repo.ListByWorkspace(ctx, ws.ID)
	if err != nil {
		t.Fatalf("ListByWorkspace() error = %v", err)
	}

	if emojis != nil {
		t.Errorf("expected nil for empty list, got %v", emojis)
	}
}

func TestRepository_Delete(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	user := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ws := testutil.CreateTestWorkspace(t, db, user.ID, "Test Workspace")

	e := &CustomEmoji{
		WorkspaceID: ws.ID,
		Name:        "partyparrot",
		CreatedBy:   user.ID,
		ContentType: "image/gif",
		SizeBytes:   2048,
		StoragePath: "/data/emojis/test.gif",
	}
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, e.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err := repo.GetByID(ctx, e.ID)
	if !errors.Is(err, ErrEmojiNotFound) {
		t.Errorf("expected ErrEmojiNotFound after delete, got %v", err)
	}
}

func TestRepository_Delete_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if !errors.Is(err, ErrEmojiNotFound) {
		t.Errorf("expected ErrEmojiNotFound, got %v", err)
	}
}
