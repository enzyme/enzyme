package scheduled

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, msg *ScheduledMessage) error {
	msg.ID = ulid.Make().String()
	now := time.Now().UTC()
	msg.CreatedAt = now
	msg.UpdatedAt = now
	msg.Status = StatusPending

	attachmentIDsJSON := "[]"
	if len(msg.AttachmentIDs) > 0 {
		data, err := json.Marshal(msg.AttachmentIDs)
		if err == nil {
			attachmentIDsJSON = string(data)
		}
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO scheduled_messages (id, channel_id, user_id, content, thread_parent_id, also_send_to_channel, attachment_ids, scheduled_for, status, retry_count, last_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.ChannelID, msg.UserID, msg.Content, msg.ThreadParentID, msg.AlsoSendToChannel,
		attachmentIDsJSON, msg.ScheduledFor.UTC().Format(time.RFC3339), msg.Status, msg.RetryCount, nil,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*ScheduledMessage, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, channel_id, user_id, content, thread_parent_id, also_send_to_channel, attachment_ids, scheduled_for, status, retry_count, last_error, created_at, updated_at
		FROM scheduled_messages WHERE id = ?
	`, id)
	return scanScheduledMessage(row)
}

func (r *Repository) Update(ctx context.Context, msg *ScheduledMessage) error {
	now := time.Now().UTC()
	msg.UpdatedAt = now

	attachmentIDsJSON := "[]"
	if len(msg.AttachmentIDs) > 0 {
		data, err := json.Marshal(msg.AttachmentIDs)
		if err == nil {
			attachmentIDsJSON = string(data)
		}
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_messages
		SET content = ?, scheduled_for = ?, attachment_ids = ?, status = ?, retry_count = 0, last_error = NULL, updated_at = ?
		WHERE id = ?
	`, msg.Content, msg.ScheduledFor.UTC().Format(time.RFC3339), attachmentIDsJSON, StatusPending, now.Format(time.RFC3339), msg.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM scheduled_messages WHERE id = ?`, id)
	return err
}

func (r *Repository) ListByUser(ctx context.Context, userID, workspaceID string) ([]ScheduledMessageWithChannel, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sm.id, sm.channel_id, sm.user_id, sm.content, sm.thread_parent_id,
		       sm.also_send_to_channel, sm.attachment_ids, sm.scheduled_for,
		       sm.status, sm.retry_count, sm.last_error,
		       sm.created_at, sm.updated_at,
		       c.name, c.workspace_id
		FROM scheduled_messages sm
		JOIN channels c ON sm.channel_id = c.id
		WHERE sm.user_id = ? AND c.workspace_id = ?
		ORDER BY sm.scheduled_for ASC
	`, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ScheduledMessageWithChannel
	for rows.Next() {
		msg, err := scanScheduledMessageWithChannel(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}
	return messages, rows.Err()
}

func (r *Repository) ListDue(ctx context.Context) ([]ScheduledMessage, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, channel_id, user_id, content, thread_parent_id, also_send_to_channel, attachment_ids, scheduled_for, status, retry_count, last_error, created_at, updated_at
		FROM scheduled_messages
		WHERE scheduled_for <= ? AND status = ?
		ORDER BY scheduled_for ASC
	`, now, StatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ScheduledMessage
	for rows.Next() {
		msg, err := scanScheduledMessageFromRows(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}
	return messages, rows.Err()
}

func (r *Repository) CountByWorkspace(ctx context.Context, userID, workspaceID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM scheduled_messages sm
		JOIN channels c ON sm.channel_id = c.id
		WHERE sm.user_id = ? AND c.workspace_id = ?
	`, userID, workspaceID).Scan(&count)
	return count, err
}

// MarkSending atomically claims a message for processing.
// Returns true if the row was updated (claimed), false if already claimed.
func (r *Repository) MarkSending(ctx context.Context, id string) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_messages SET status = ?, updated_at = ?
		WHERE id = ? AND status = ?
	`, StatusSending, now, id, StatusPending)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// MarkFailed marks a message as permanently failed.
func (r *Repository) MarkFailed(ctx context.Context, id, lastError string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_messages SET status = ?, last_error = ?, updated_at = ?
		WHERE id = ?
	`, StatusFailed, lastError, now, id)
	return err
}

// IncrementRetry increments retry count and resets status to pending.
func (r *Repository) IncrementRetry(ctx context.Context, id, lastError string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_messages SET status = ?, retry_count = retry_count + 1, last_error = ?, updated_at = ?
		WHERE id = ?
	`, StatusPending, lastError, now, id)
	return err
}

// ResetStuckSending recovers messages stuck in "sending" state (crash recovery).
// Returns the number of messages reset.
func (r *Repository) ResetStuckSending(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	now := time.Now().UTC()
	threshold := now.Add(-staleThreshold).Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_messages SET status = ?, retry_count = retry_count + 1, updated_at = ?
		WHERE status = ? AND updated_at < ?
	`, StatusPending, now.Format(time.RFC3339), StatusSending, threshold)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// RemoveAttachmentID removes an attachment reference from scheduled messages when a file is deleted.
// Returns the affected messages so the caller can notify users.
func (r *Repository) RemoveAttachmentID(ctx context.Context, attachmentID string) ([]ScheduledMessage, error) {
	// Find all pending/sending scheduled messages that reference this attachment
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, channel_id, user_id, content, thread_parent_id, also_send_to_channel, attachment_ids, scheduled_for, status, retry_count, last_error, created_at, updated_at
		FROM scheduled_messages
		WHERE status IN (?, ?) AND attachment_ids LIKE ?
	`, StatusPending, StatusSending, "%"+attachmentID+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var affected []ScheduledMessage
	for rows.Next() {
		msg, err := scanScheduledMessageFromRows(rows)
		if err != nil {
			return nil, err
		}
		// Check if this message actually contains the attachment ID
		newIDs := make([]string, 0, len(msg.AttachmentIDs))
		found := false
		for _, id := range msg.AttachmentIDs {
			if id == attachmentID {
				found = true
				continue
			}
			newIDs = append(newIDs, id)
		}
		if !found {
			continue
		}
		msg.AttachmentIDs = newIDs
		affected = append(affected, *msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Update each affected message
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range affected {
		idsJSON := "[]"
		if len(affected[i].AttachmentIDs) > 0 {
			data, _ := json.Marshal(affected[i].AttachmentIDs)
			idsJSON = string(data)
		}
		_, err := r.db.ExecContext(ctx, `
			UPDATE scheduled_messages SET attachment_ids = ?, updated_at = ? WHERE id = ?
		`, idsJSON, now, affected[i].ID)
		if err != nil {
			return nil, err
		}
	}

	return affected, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanBase scans the base ScheduledMessage columns from any scanner.
func scanBase(s scanner, extra ...any) (*ScheduledMessage, error) {
	var msg ScheduledMessage
	var threadParentID sql.NullString
	var attachmentIDsJSON string
	var lastError sql.NullString
	var scheduledFor, createdAt, updatedAt string

	dest := []any{
		&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Content,
		&threadParentID, &msg.AlsoSendToChannel, &attachmentIDsJSON,
		&scheduledFor, &msg.Status, &msg.RetryCount, &lastError,
		&createdAt, &updatedAt,
	}
	dest = append(dest, extra...)

	if err := s.Scan(dest...); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrScheduledMessageNotFound
		}
		return nil, err
	}

	if threadParentID.Valid {
		msg.ThreadParentID = &threadParentID.String
	}

	if lastError.Valid {
		msg.LastError = lastError.String
	}

	if err := json.Unmarshal([]byte(attachmentIDsJSON), &msg.AttachmentIDs); err != nil {
		msg.AttachmentIDs = []string{}
	}

	msg.ScheduledFor, _ = time.Parse(time.RFC3339, scheduledFor)
	msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	msg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &msg, nil
}

func scanScheduledMessage(row *sql.Row) (*ScheduledMessage, error) {
	return scanBase(row)
}

func scanScheduledMessageFromRows(rows *sql.Rows) (*ScheduledMessage, error) {
	return scanBase(rows)
}

func scanScheduledMessageWithChannel(rows *sql.Rows) (*ScheduledMessageWithChannel, error) {
	var msg ScheduledMessageWithChannel
	base, err := scanBase(rows, &msg.ChannelName, &msg.WorkspaceID)
	if err != nil {
		return nil, err
	}
	msg.ScheduledMessage = *base
	return &msg, nil
}
