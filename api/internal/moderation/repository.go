package moderation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrBanNotFound   = errors.New("ban not found")
	ErrAlreadyBanned = errors.New("user is already banned")
	ErrBlockNotFound = errors.New("block not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// --- Bans ---

// CreateBan creates a new workspace ban. Uses the provided transaction if non-nil.
func (r *Repository) CreateBan(ctx context.Context, tx *sql.Tx, ban *Ban) error {
	ban.ID = ulid.Make().String()
	now := time.Now().UTC()
	ban.CreatedAt = now

	hideMessages := 0
	if ban.HideMessages {
		hideMessages = 1
	}

	var expiresAt *string
	if ban.ExpiresAt != nil {
		s := ban.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}

	var execer interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	}
	if tx != nil {
		execer = tx
	} else {
		execer = r.db
	}

	_, err := execer.ExecContext(ctx, `
		INSERT INTO workspace_bans (id, workspace_id, user_id, banned_by, reason, hide_messages, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ban.ID, ban.WorkspaceID, ban.UserID, ban.BannedBy, ban.Reason, hideMessages, expiresAt, now.Format(time.RFC3339))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyBanned
		}
		return err
	}
	return nil
}

// DeleteBan removes a ban record
func (r *Repository) DeleteBan(ctx context.Context, workspaceID, userID string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM workspace_bans WHERE workspace_id = ? AND user_id = ?
	`, workspaceID, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBanNotFound
	}
	return nil
}

// GetActiveBan returns an active (non-expired) ban for a user in a workspace
func (r *Repository) GetActiveBan(ctx context.Context, workspaceID, userID string) (*Ban, error) {
	var ban Ban
	var hideMessages int
	var expiresAt, createdAt sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, user_id, banned_by, reason, hide_messages, expires_at, created_at
		FROM workspace_bans
		WHERE workspace_id = ? AND user_id = ?
		AND (expires_at IS NULL OR expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	`, workspaceID, userID).Scan(
		&ban.ID, &ban.WorkspaceID, &ban.UserID, &ban.BannedBy,
		&ban.Reason, &hideMessages, &expiresAt, &createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	ban.HideMessages = hideMessages == 1
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		ban.ExpiresAt = &t
	}
	if createdAt.Valid {
		ban.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}

	return &ban, nil
}

// ListActiveBans returns active bans for a workspace with user display info
func (r *Repository) ListActiveBans(ctx context.Context, workspaceID string, cursor string, limit int) ([]BanWithUser, bool, string, error) {
	if limit <= 0 {
		limit = 50
	}

	args := []interface{}{workspaceID}
	cursorClause := ""
	if cursor != "" {
		cursorClause = "AND wb.id < ?"
		args = append(args, cursor)
	}
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, `
		SELECT wb.id, wb.workspace_id, wb.user_id, wb.banned_by, wb.reason,
			   wb.hide_messages, wb.expires_at, wb.created_at,
			   u.display_name, u.email, u.avatar_url,
			   b.display_name
		FROM workspace_bans wb
		JOIN users u ON u.id = wb.user_id
		JOIN users b ON b.id = wb.banned_by
		WHERE wb.workspace_id = ?
		AND (wb.expires_at IS NULL OR wb.expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		`+cursorClause+`
		ORDER BY wb.id DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()

	var bans []BanWithUser
	for rows.Next() {
		var b BanWithUser
		var hideMessages int
		var expiresAt, createdAt sql.NullString

		err := rows.Scan(
			&b.ID, &b.WorkspaceID, &b.UserID, &b.BannedBy, &b.Reason,
			&hideMessages, &expiresAt, &createdAt,
			&b.UserDisplayName, &b.UserEmail, &b.UserAvatarURL,
			&b.BannedByName,
		)
		if err != nil {
			return nil, false, "", err
		}

		b.HideMessages = hideMessages == 1
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			b.ExpiresAt = &t
		}
		if createdAt.Valid {
			b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}

		bans = append(bans, b)
	}

	hasMore := len(bans) > limit
	nextCursor := ""
	if hasMore {
		bans = bans[:limit]
		nextCursor = bans[len(bans)-1].ID
	}

	return bans, hasMore, nextCursor, nil
}

// --- Blocks ---

// CreateBlock creates a user block
func (r *Repository) CreateBlock(ctx context.Context, blockerID, blockedID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_blocks (blocker_id, blocked_id, created_at)
		VALUES (?, ?, ?)
	`, blockerID, blockedID, now.Format(time.RFC3339))
	if err != nil {
		if isUniqueViolation(err) {
			return nil // Idempotent — already blocked
		}
		return err
	}
	return nil
}

// DeleteBlock removes a user block
func (r *Repository) DeleteBlock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?
	`, blockerID, blockedID)
	return err // Idempotent — no error if not found
}

// ListBlocks returns all users blocked by the given user
func (r *Repository) ListBlocks(ctx context.Context, blockerID string) ([]BlockWithUser, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ub.blocker_id, ub.blocked_id, ub.created_at,
			   u.display_name, u.email, u.avatar_url
		FROM user_blocks ub
		JOIN users u ON u.id = ub.blocked_id
		WHERE ub.blocker_id = ?
		ORDER BY ub.created_at DESC
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []BlockWithUser
	for rows.Next() {
		var b BlockWithUser
		var createdAt string
		err := rows.Scan(
			&b.BlockerID, &b.BlockedID, &createdAt,
			&b.DisplayName, &b.Email, &b.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// GetBlockedUserIDs returns the set of user IDs blocked by the given user
func (r *Repository) GetBlockedUserIDs(ctx context.Context, blockerID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT blocked_id FROM user_blocks WHERE blocker_id = ?
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocked := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		blocked[id] = true
	}
	return blocked, nil
}

// IsBlocked checks if blockerID has blocked blockedID
func (r *Repository) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?
	`, blockerID, blockedID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsBlockedEitherDirection checks if either user has blocked the other
func (r *Repository) IsBlockedEitherDirection(ctx context.Context, userA, userB string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_blocks
		WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)
	`, userA, userB, userB, userA).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// --- Audit Log ---

// CreateAuditLogEntry creates an audit log entry
func (r *Repository) CreateAuditLogEntry(ctx context.Context, entry *AuditLogEntry) error {
	entry.ID = ulid.Make().String()
	now := time.Now().UTC()
	entry.CreatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO moderation_log (id, workspace_id, actor_id, action, target_type, target_id, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.WorkspaceID, entry.ActorID, entry.Action, entry.TargetType, entry.TargetID, entry.Metadata, now.Format(time.RFC3339))
	return err
}

// CreateAuditLogEntryWithMetadata creates an audit log entry with a metadata map
func (r *Repository) CreateAuditLogEntryWithMetadata(ctx context.Context, workspaceID, actorID, action, targetType, targetID string, metadata map[string]interface{}) error {
	var metadataJSON *string
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err == nil {
			s := string(data)
			metadataJSON = &s
		}
	}

	entry := &AuditLogEntry{
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Metadata:    metadataJSON,
	}
	return r.CreateAuditLogEntry(ctx, entry)
}

// ListAuditLog returns audit log entries for a workspace with cursor-based pagination
func (r *Repository) ListAuditLog(ctx context.Context, workspaceID string, cursor string, limit int) ([]AuditLogEntryWithActor, bool, string, error) {
	if limit <= 0 {
		limit = 50
	}

	args := []interface{}{workspaceID}
	cursorClause := ""
	if cursor != "" {
		cursorClause = "AND ml.id < ?"
		args = append(args, cursor)
	}
	args = append(args, limit+1)

	rows, err := r.db.QueryContext(ctx, `
		SELECT ml.id, ml.workspace_id, ml.actor_id, ml.action,
			   ml.target_type, ml.target_id, ml.metadata, ml.created_at,
			   u.display_name, u.avatar_url
		FROM moderation_log ml
		JOIN users u ON u.id = ml.actor_id
		WHERE ml.workspace_id = ?
		`+cursorClause+`
		ORDER BY ml.id DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, false, "", err
	}
	defer rows.Close()

	var entries []AuditLogEntryWithActor
	for rows.Next() {
		var e AuditLogEntryWithActor
		var createdAt string
		err := rows.Scan(
			&e.ID, &e.WorkspaceID, &e.ActorID, &e.Action,
			&e.TargetType, &e.TargetID, &e.Metadata, &createdAt,
			&e.ActorDisplayName, &e.ActorAvatarURL,
		)
		if err != nil {
			return nil, false, "", err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entries = append(entries, e)
	}

	hasMore := len(entries) > limit
	nextCursor := ""
	if hasMore {
		entries = entries[:limit]
		nextCursor = entries[len(entries)-1].ID
	}

	return entries, hasMore, nextCursor, nil
}

// isUniqueViolation checks if the error is a SQLite unique constraint violation
func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key"))
}
