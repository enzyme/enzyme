package file

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrAttachmentNotFound = errors.New("attachment not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, attachment *Attachment) error {
	attachment.ID = ulid.Make().String()
	attachment.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO attachments (id, message_id, channel_id, user_id, filename, content_type, size_bytes, storage_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, attachment.ID, attachment.MessageID, attachment.ChannelID, attachment.UserID, attachment.Filename, attachment.ContentType, attachment.SizeBytes, attachment.StoragePath, attachment.CreatedAt.Format(time.RFC3339))
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Attachment, error) {
	var a Attachment
	var messageID, userID sql.NullString
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, message_id, channel_id, user_id, filename, content_type, size_bytes, storage_path, created_at
		FROM attachments WHERE id = ?
	`, id).Scan(&a.ID, &messageID, &a.ChannelID, &userID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}

	if messageID.Valid {
		a.MessageID = &messageID.String
	}
	if userID.Valid {
		a.UserID = &userID.String
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &a, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM attachments WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}

func (r *Repository) ListForMessage(ctx context.Context, messageID string) ([]Attachment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, message_id, channel_id, user_id, filename, content_type, size_bytes, storage_path, created_at
		FROM attachments WHERE message_id = ?
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []Attachment
	for rows.Next() {
		var a Attachment
		var msgID, userID sql.NullString
		var createdAt string

		err := rows.Scan(&a.ID, &msgID, &a.ChannelID, &userID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath, &createdAt)
		if err != nil {
			return nil, err
		}

		if msgID.Valid {
			a.MessageID = &msgID.String
		}
		if userID.Valid {
			a.UserID = &userID.String
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		attachments = append(attachments, a)
	}

	return attachments, rows.Err()
}

func (r *Repository) UpdateMessageID(ctx context.Context, attachmentID, messageID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE attachments SET message_id = ? WHERE id = ?
	`, messageID, attachmentID)
	return err
}

// ListForMessages returns attachments for multiple messages, keyed by message ID
func (r *Repository) ListForMessages(ctx context.Context, messageIDs []string) (map[string][]Attachment, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT id, message_id, channel_id, user_id, filename, content_type, size_bytes, storage_path, created_at
		FROM attachments
		WHERE message_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make(map[string][]Attachment)
	for rows.Next() {
		var a Attachment
		var messageID, userID sql.NullString
		var createdAt string

		err := rows.Scan(&a.ID, &messageID, &a.ChannelID, &userID, &a.Filename, &a.ContentType, &a.SizeBytes, &a.StoragePath, &createdAt)
		if err != nil {
			return nil, err
		}

		if messageID.Valid {
			a.MessageID = &messageID.String
		}
		if userID.Valid {
			a.UserID = &userID.String
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		if messageID.Valid {
			attachments[messageID.String] = append(attachments[messageID.String], a)
		}
	}

	return attachments, rows.Err()
}
