package emoji

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrEmojiNotFound  = errors.New("custom emoji not found")
	ErrEmojiNameTaken = errors.New("emoji name already taken in this workspace")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, e *CustomEmoji) error {
	e.ID = ulid.Make().String()
	e.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO custom_emojis (id, workspace_id, name, created_by, content_type, size_bytes, storage_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.WorkspaceID, e.Name, e.CreatedBy, e.ContentType, e.SizeBytes, e.StoragePath, e.CreatedAt.Format(time.RFC3339))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrEmojiNameTaken
		}
		return err
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*CustomEmoji, error) {
	var e CustomEmoji
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, created_by, content_type, size_bytes, storage_path, created_at
		FROM custom_emojis WHERE id = ?
	`, id).Scan(&e.ID, &e.WorkspaceID, &e.Name, &e.CreatedBy, &e.ContentType, &e.SizeBytes, &e.StoragePath, &createdAt)
	if err == sql.ErrNoRows {
		return nil, ErrEmojiNotFound
	}
	if err != nil {
		return nil, err
	}

	e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &e, nil
}

func (r *Repository) ListByWorkspace(ctx context.Context, workspaceID string) ([]CustomEmoji, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, created_by, content_type, size_bytes, storage_path, created_at
		FROM custom_emojis WHERE workspace_id = ? ORDER BY name ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emojis []CustomEmoji
	for rows.Next() {
		var e CustomEmoji
		var createdAt string

		err := rows.Scan(&e.ID, &e.WorkspaceID, &e.Name, &e.CreatedBy, &e.ContentType, &e.SizeBytes, &e.StoragePath, &createdAt)
		if err != nil {
			return nil, err
		}

		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		emojis = append(emojis, e)
	}

	return emojis, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM custom_emojis WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrEmojiNotFound
	}
	return nil
}
