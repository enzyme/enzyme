package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/enzyme/api/internal/database"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"
)

// TestDB creates an in-memory SQLite database with migrations applied.
// The database is automatically closed when the test completes.
func TestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(":memory:", database.Options{
		MaxOpenConns: 1, // single connection for test determinism
		BusyTimeout:  5000,
		CacheSize:    -2000,
	})
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		_ = db.Close()
		t.Fatalf("running migrations: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db.DB
}

// hashPassword creates a bcrypt hash with low cost for tests
func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 4)
	return string(hash)
}

// TestUser represents a test user
type TestUser struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateTestUser creates a user directly in the database without using the user package
func CreateTestUser(t *testing.T, db *sql.DB, email, displayName string) *TestUser {
	t.Helper()

	id := ulid.Make().String()
	hash := hashPassword("password123")
	now := time.Now().UTC()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO users (id, email, password_hash, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?)
	`, id, email, hash, displayName, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	return &TestUser{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// TestWorkspace represents a test workspace
type TestWorkspace struct {
	ID        string
	Name      string
	Settings  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateTestWorkspace creates a workspace directly in the database
func CreateTestWorkspace(t *testing.T, db *sql.DB, ownerID, name string) *TestWorkspace {
	t.Helper()

	id := ulid.Make().String()
	now := time.Now().UTC()

	// Create workspace
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO workspaces (id, name, settings, created_at, updated_at)
		VALUES (?, ?, '{}', ?, ?)
	`, id, name, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating test workspace: %v", err)
	}

	// Add owner membership
	membershipID := ulid.Make().String()
	_, err = db.ExecContext(context.Background(), `
		INSERT INTO workspace_memberships (id, user_id, workspace_id, role, created_at, updated_at)
		VALUES (?, ?, ?, 'owner', ?, ?)
	`, membershipID, ownerID, id, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating workspace membership: %v", err)
	}

	return &TestWorkspace{
		ID:        id,
		Name:      name,
		Settings:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestChannel represents a test channel
type TestChannel struct {
	ID          string
	WorkspaceID string
	Name        string
	Type        string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateTestChannel creates a channel directly in the database
func CreateTestChannel(t *testing.T, db *sql.DB, workspaceID, creatorID, name, channelType string) *TestChannel {
	t.Helper()

	id := ulid.Make().String()
	now := time.Now().UTC()

	// Create channel
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO channels (id, workspace_id, name, type, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, workspaceID, name, channelType, creatorID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating test channel: %v", err)
	}

	// Add creator as admin member
	membershipID := ulid.Make().String()
	_, err = db.ExecContext(context.Background(), `
		INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
		VALUES (?, ?, ?, 'admin', ?, ?)
	`, membershipID, creatorID, id, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating channel membership: %v", err)
	}

	return &TestChannel{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        name,
		Type:        channelType,
		CreatedBy:   creatorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// TestMessage represents a test message
type TestMessage struct {
	ID        string
	ChannelID string
	UserID    string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TestEmoji represents a test custom emoji
type TestEmoji struct {
	ID          string
	WorkspaceID string
	Name        string
	CreatedBy   string
	ContentType string
	SizeBytes   int64
	StoragePath string
	CreatedAt   time.Time
}

// CreateTestEmoji creates a custom emoji directly in the database
func CreateTestEmoji(t *testing.T, db *sql.DB, workspaceID, createdBy, name string) *TestEmoji {
	t.Helper()

	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO custom_emojis (id, workspace_id, name, created_by, content_type, size_bytes, storage_path, created_at)
		VALUES (?, ?, ?, ?, 'image/png', 1024, '/emojis/test.png', ?)
	`, id, workspaceID, name, createdBy, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating test emoji: %v", err)
	}

	return &TestEmoji{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        name,
		CreatedBy:   createdBy,
		ContentType: "image/png",
		SizeBytes:   1024,
		StoragePath: "/emojis/test.png",
		CreatedAt:   now,
	}
}

// CreateTestMessage creates a message directly in the database
func CreateTestMessage(t *testing.T, db *sql.DB, channelID, userID, content string) *TestMessage {
	t.Helper()

	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO messages (id, channel_id, user_id, content, reply_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)
	`, id, channelID, userID, content, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating test message: %v", err)
	}

	return &TestMessage{
		ID:        id,
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
