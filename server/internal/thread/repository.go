package thread

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
)

// Repository handles thread subscription database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new thread repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetSubscription returns a user's subscription to a thread, or nil if none exists
func (r *Repository) GetSubscription(ctx context.Context, threadParentID, userID string) (*Subscription, error) {
	query := `
		SELECT id, thread_parent_id, user_id, status, last_read_reply_id, created_at, updated_at
		FROM thread_subscriptions
		WHERE thread_parent_id = ? AND user_id = ?
	`

	var sub Subscription
	var lastReadReplyID sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, query, threadParentID, userID).Scan(
		&sub.ID,
		&sub.ThreadParentID,
		&sub.UserID,
		&sub.Status,
		&lastReadReplyID,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastReadReplyID.Valid {
		sub.LastReadReplyID = &lastReadReplyID.String
	}
	sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sub.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &sub, nil
}

// GetSubscribedUserIDs returns all user IDs subscribed to a thread
func (r *Repository) GetSubscribedUserIDs(ctx context.Context, threadParentID string) ([]string, error) {
	query := `
		SELECT user_id FROM thread_subscriptions
		WHERE thread_parent_id = ? AND status = 'subscribed'
	`

	rows, err := r.db.QueryContext(ctx, query, threadParentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

// Subscribe creates or updates a subscription to "subscribed" status
func (r *Repository) Subscribe(ctx context.Context, threadParentID, userID string) (*Subscription, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := ulid.Make().String()

	// Use INSERT OR REPLACE to handle both new and existing subscriptions
	query := `
		INSERT INTO thread_subscriptions (id, thread_parent_id, user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'subscribed', ?, ?)
		ON CONFLICT(thread_parent_id, user_id) DO UPDATE SET
			status = 'subscribed',
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query, id, threadParentID, userID, now, now)
	if err != nil {
		return nil, err
	}

	// Fetch the updated/created subscription
	return r.GetSubscription(ctx, threadParentID, userID)
}

// Unsubscribe creates or updates a subscription to "unsubscribed" status
func (r *Repository) Unsubscribe(ctx context.Context, threadParentID, userID string) (*Subscription, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := ulid.Make().String()

	// Use INSERT OR REPLACE to handle both new and existing subscriptions
	query := `
		INSERT INTO thread_subscriptions (id, thread_parent_id, user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'unsubscribed', ?, ?)
		ON CONFLICT(thread_parent_id, user_id) DO UPDATE SET
			status = 'unsubscribed',
			updated_at = excluded.updated_at
	`

	_, err := r.db.ExecContext(ctx, query, id, threadParentID, userID, now, now)
	if err != nil {
		return nil, err
	}

	// Fetch the updated/created subscription
	return r.GetSubscription(ctx, threadParentID, userID)
}

// AutoSubscribe subscribes a user to a thread ONLY if they have no existing subscription row.
// This respects explicit unsubscribes - if a user has previously unsubscribed, they won't be
// re-subscribed automatically.
func (r *Repository) AutoSubscribe(ctx context.Context, threadParentID, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id := ulid.Make().String()

	// INSERT OR IGNORE - only creates row if no subscription exists
	query := `
		INSERT OR IGNORE INTO thread_subscriptions (id, thread_parent_id, user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'subscribed', ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, id, threadParentID, userID, now, now)
	return err
}

// UpdateLastReadReplyID updates the last read reply ID for a thread subscription
func (r *Repository) UpdateLastReadReplyID(ctx context.Context, threadParentID, userID, replyID string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		UPDATE thread_subscriptions
		SET last_read_reply_id = ?, updated_at = ?
		WHERE thread_parent_id = ? AND user_id = ? AND status = 'subscribed'
	`

	_, err := r.db.ExecContext(ctx, query, replyID, now, threadParentID, userID)
	return err
}

// CountUnreadThreads counts how many subscribed threads have new replies for a user in a workspace
func (r *Repository) CountUnreadThreads(ctx context.Context, workspaceID, userID string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT ts.thread_parent_id)
		FROM thread_subscriptions ts
		JOIN messages m ON m.id = ts.thread_parent_id
		JOIN channels c ON c.id = m.channel_id
		WHERE ts.user_id = ?
		  AND ts.status = 'subscribed'
		  AND c.workspace_id = ?
		  AND m.deleted_at IS NULL
		  AND (
		    ts.last_read_reply_id IS NULL
		    OR EXISTS (
		      SELECT 1 FROM messages r
		      WHERE r.thread_parent_id = m.id
		        AND r.id > ts.last_read_reply_id
		        AND r.deleted_at IS NULL
		      LIMIT 1
		    )
		  )
		  AND m.reply_count > 0
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, workspaceID).Scan(&count)
	return count, err
}

// GetLatestReplyID returns the ID of the latest reply in a thread
func (r *Repository) GetLatestReplyID(ctx context.Context, threadParentID string) (string, error) {
	query := `
		SELECT id FROM messages
		WHERE thread_parent_id = ? AND deleted_at IS NULL
		ORDER BY id DESC
		LIMIT 1
	`

	var replyID string
	err := r.db.QueryRowContext(ctx, query, threadParentID).Scan(&replyID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return replyID, err
}
