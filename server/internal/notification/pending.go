package notification

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
)

// Notification types
const (
	TypeMention     = "mention"
	TypeDM          = "dm"
	TypeChannel     = "channel"
	TypeHere        = "here"
	TypeEveryone    = "everyone"
	TypeThreadReply = "thread_reply"
)

// PendingNotification represents a notification queued for email delivery
type PendingNotification struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	WorkspaceID      string    `json:"workspace_id"`
	ChannelID        string    `json:"channel_id"`
	MessageID        string    `json:"message_id"`
	NotificationType string    `json:"notification_type"`
	CreatedAt        time.Time `json:"created_at"`
	SendAfter        time.Time `json:"send_after"`
}

// PendingRepository handles pending notification persistence
type PendingRepository struct {
	db *sql.DB
}

// NewPendingRepository creates a new pending notifications repository
func NewPendingRepository(db *sql.DB) *PendingRepository {
	return &PendingRepository{db: db}
}

// Create adds a new pending notification
func (r *PendingRepository) Create(ctx context.Context, notification *PendingNotification) error {
	notification.ID = ulid.Make().String()
	notification.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO pending_notifications (id, user_id, workspace_id, channel_id, message_id, notification_type, created_at, send_after)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, message_id) DO NOTHING
	`, notification.ID, notification.UserID, notification.WorkspaceID, notification.ChannelID,
		notification.MessageID, notification.NotificationType,
		notification.CreatedAt.Format(time.RFC3339), notification.SendAfter.Format(time.RFC3339))

	return err
}

// DeleteForUser removes all pending notifications for a user
func (r *PendingRepository) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM pending_notifications WHERE user_id = ?
	`, userID)
	return err
}

// DeleteForUserInWorkspace removes pending notifications for a user in a specific workspace
func (r *PendingRepository) DeleteForUserInWorkspace(ctx context.Context, userID, workspaceID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM pending_notifications WHERE user_id = ? AND workspace_id = ?
	`, userID, workspaceID)
	return err
}

// GetReadyToSend returns all notifications that are ready to be sent (send_after <= now)
func (r *PendingRepository) GetReadyToSend(ctx context.Context) ([]PendingNotification, error) {
	now := time.Now().UTC()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, workspace_id, channel_id, message_id, notification_type, created_at, send_after
		FROM pending_notifications
		WHERE send_after <= ?
		ORDER BY created_at
	`, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []PendingNotification
	for rows.Next() {
		var n PendingNotification
		var createdAt, sendAfter string

		err := rows.Scan(&n.ID, &n.UserID, &n.WorkspaceID, &n.ChannelID, &n.MessageID,
			&n.NotificationType, &createdAt, &sendAfter)
		if err != nil {
			return nil, err
		}

		n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		n.SendAfter, _ = time.Parse(time.RFC3339, sendAfter)
		notifications = append(notifications, n)
	}

	return notifications, rows.Err()
}

// GetGroupedByUser returns pending notifications grouped by user ID
func (r *PendingRepository) GetGroupedByUser(ctx context.Context) (map[string][]PendingNotification, error) {
	now := time.Now().UTC()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, workspace_id, channel_id, message_id, notification_type, created_at, send_after
		FROM pending_notifications
		WHERE send_after <= ?
		ORDER BY user_id, created_at
	`, now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]PendingNotification)
	for rows.Next() {
		var n PendingNotification
		var createdAt, sendAfter string

		err := rows.Scan(&n.ID, &n.UserID, &n.WorkspaceID, &n.ChannelID, &n.MessageID,
			&n.NotificationType, &createdAt, &sendAfter)
		if err != nil {
			return nil, err
		}

		n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		n.SendAfter, _ = time.Parse(time.RFC3339, sendAfter)
		result[n.UserID] = append(result[n.UserID], n)
	}

	return result, rows.Err()
}

// DeleteByIDs removes pending notifications by their IDs
func (r *PendingRepository) DeleteByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build query with placeholders
	query := "DELETE FROM pending_notifications WHERE id IN (?"
	args := make([]interface{}, len(ids))
	args[0] = ids[0]
	for i := 1; i < len(ids); i++ {
		query += ",?"
		args[i] = ids[i]
	}
	query += ")"

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// CountForUser returns the count of pending notifications for a user
func (r *PendingRepository) CountForUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pending_notifications WHERE user_id = ?
	`, userID).Scan(&count)
	return count, err
}
