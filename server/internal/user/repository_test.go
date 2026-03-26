package user

import (
	"context"
	"errors"
	"testing"

	"github.com/enzyme/server/internal/testutil"
)

func TestRepository_Create(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	u, err := repo.Create(ctx, CreateUserInput{
		Email:        "test@example.com",
		DisplayName:  "Test User",
		PasswordHash: "$2a$04$fakehash",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if u.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "test@example.com")
	}
	if u.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", u.DisplayName, "Test User")
	}
	if u.Status != "active" {
		t.Errorf("Status = %q, want %q", u.Status, "active")
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
}

func TestRepository_Create_DuplicateEmail(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	input := CreateUserInput{
		Email:        "duplicate@example.com",
		DisplayName:  "Test User",
		PasswordHash: "$2a$04$fakehash",
	}

	// First creation should succeed
	_, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	// Second creation with same email should fail
	_, err = repo.Create(ctx, input)
	if !errors.Is(err, ErrEmailAlreadyInUse) {
		t.Errorf("second Create() error = %v, want %v", err, ErrEmailAlreadyInUse)
	}
}

func TestRepository_GetByID(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, CreateUserInput{
		Email:        "test@example.com",
		DisplayName:  "Test User",
		PasswordHash: "$2a$04$fakehash",
	})

	u, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if u.ID != created.ID {
		t.Errorf("ID = %q, want %q", u.ID, created.ID)
	}
	if u.Email != created.Email {
		t.Errorf("Email = %q, want %q", u.Email, created.Email)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetByID() error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestRepository_GetByEmail(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, CreateUserInput{
		Email:        "findme@example.com",
		DisplayName:  "Find Me",
		PasswordHash: "$2a$04$fakehash",
	})

	u, err := repo.GetByEmail(ctx, "findme@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if u.ID != created.ID {
		t.Errorf("ID = %q, want %q", u.ID, created.ID)
	}
}

func TestRepository_GetByEmail_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("GetByEmail() error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestRepository_Update(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, CreateUserInput{
		Email:        "update@example.com",
		DisplayName:  "Original Name",
		PasswordHash: "$2a$04$fakehash",
	})

	created.DisplayName = "Updated Name"
	avatarURL := "https://example.com/avatar.png"
	created.AvatarURL = &avatarURL

	err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if updated.DisplayName != "Updated Name" {
		t.Errorf("DisplayName = %q, want %q", updated.DisplayName, "Updated Name")
	}
	if updated.AvatarURL == nil || *updated.AvatarURL != avatarURL {
		t.Errorf("AvatarURL = %v, want %q", updated.AvatarURL, avatarURL)
	}
}

func TestRepository_UpdatePassword(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, CreateUserInput{
		Email:        "password@example.com",
		DisplayName:  "Password User",
		PasswordHash: "$2a$04$originalhash",
	})

	newHash := "$2a$04$newhash"
	err := repo.UpdatePassword(ctx, created.ID, newHash)
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, created.ID)
	if updated.PasswordHash != newHash {
		t.Errorf("PasswordHash = %q, want %q", updated.PasswordHash, newHash)
	}
}

func TestRepository_VerifyEmail(t *testing.T) {
	db := testutil.TestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, CreateUserInput{
		Email:        "verify@example.com",
		DisplayName:  "Verify User",
		PasswordHash: "$2a$04$fakehash",
	})

	if created.EmailVerifiedAt != nil {
		t.Error("email should not be verified initially")
	}

	err := repo.VerifyEmail(ctx, created.ID)
	if err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}

	updated, _ := repo.GetByID(ctx, created.ID)
	if updated.EmailVerifiedAt == nil {
		t.Error("email should be verified after VerifyEmail()")
	}
}
