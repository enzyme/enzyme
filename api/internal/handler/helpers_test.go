package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/emoji"
	"github.com/feather/api/internal/file"
	"github.com/feather/api/internal/message"
	"github.com/feather/api/internal/notification"
	"github.com/feather/api/internal/signing"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/testutil"
	"github.com/feather/api/internal/thread"
	"github.com/feather/api/internal/user"
	"github.com/feather/api/internal/workspace"
	"github.com/oklog/ulid/v2"
)

// testHandler creates a fully-wired Handler backed by an in-memory SQLite database.
func testHandler(t *testing.T) (*Handler, *sql.DB) {
	t.Helper()

	db := testutil.TestDB(t)

	userRepo := user.NewRepository(db)
	workspaceRepo := workspace.NewRepository(db)
	channelRepo := channel.NewRepository(db)
	messageRepo := message.NewRepository(db)
	fileRepo := file.NewRepository(db)
	threadRepo := thread.NewRepository(db)
	emojiRepo := emoji.NewRepository(db)
	hub := sse.NewHub(db, 24*time.Hour, time.Hour)

	passwordResets := auth.NewPasswordResetRepo(db)
	authService := auth.NewService(userRepo, passwordResets, 4)

	sessionStore := auth.NewSessionStore(db, 24*time.Hour)

	notifPrefsRepo := notification.NewPreferencesRepository(db)
	notifPendingRepo := notification.NewPendingRepository(db)
	notifService := notification.NewService(notifPrefsRepo, notifPendingRepo, channelRepo, hub)

	h := New(Dependencies{
		AuthService:         authService,
		SessionStore:        sessionStore,
		UserRepo:            userRepo,
		WorkspaceRepo:       workspaceRepo,
		ChannelRepo:         channelRepo,
		MessageRepo:         messageRepo,
		FileRepo:            fileRepo,
		ThreadRepo:          threadRepo,
		EmojiRepo:           emojiRepo,
		NotificationService: notifService,
		Hub:                 hub,
		Signer:              signing.NewSigner("test-signing-secret"),
		StoragePath:         t.TempDir(),
		MaxUploadSize:       10 * 1024 * 1024,
		PublicURL:           "http://localhost:8080",
	})

	return h, db
}

// ctxWithUser creates a context that simulates an authenticated user.
// Sets the user ID and token directly in context (as TokenMiddleware would)
// and attaches a request for handlers that need it.
func ctxWithUser(t *testing.T, h *Handler, userID string) context.Context {
	t.Helper()

	// Create a token for this user
	token, err := h.sessionStore.Create(userID)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	// Build context with user ID, token, and request (as middleware would)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.WithUserID(r.Context(), userID)
	ctx = auth.WithToken(ctx, token)
	return WithRequest(ctx, r.WithContext(ctx))
}

// addWorkspaceMember adds a user to a workspace with the given role directly in the database.
func addWorkspaceMember(t *testing.T, db *sql.DB, userID, workspaceID, role string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO workspace_memberships (id, user_id, workspace_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, ulid.Make().String(), userID, workspaceID, role,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("adding workspace member: %v", err)
	}
}

// addChannelMember adds a user to a channel with the given role directly in the database.
func addChannelMember(t *testing.T, db *sql.DB, userID, channelID string, role *string) {
	t.Helper()

	var roleVal any
	if role != nil {
		roleVal = *role
	}

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, ulid.Make().String(), userID, channelID, roleVal,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("adding channel member: %v", err)
	}
}

// createFileAttachment creates a file attachment directly in the database.
// It also creates a real temp file so that DeleteFile's os.Remove succeeds safely.
func createFileAttachment(t *testing.T, db *sql.DB, channelID, userID string) string {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("creating temp file: %v", err)
	}

	id := ulid.Make().String()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO attachments (id, channel_id, user_id, filename, content_type, size_bytes, storage_path, created_at)
		VALUES (?, ?, ?, 'test.txt', 'text/plain', 100, ?, ?)
	`, id, channelID, userID, tmpFile, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("creating file attachment: %v", err)
	}
	return id
}
