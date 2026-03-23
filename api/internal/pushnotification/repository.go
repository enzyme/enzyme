package pushnotification

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

var ErrTokenNotFound = errors.New("device token not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Upsert inserts a new device token or updates the existing one on (user_id, token) conflict.
// If the user already has MaxTokensPerUser tokens, the least-recently-updated one is evicted.
func (r *Repository) Upsert(ctx context.Context, token *DeviceToken) error {
	now := time.Now().UTC()
	if token.ID == "" {
		token.ID = ulid.Make().String()
	}
	token.CreatedAt = now
	token.UpdatedAt = now

	// Evict oldest token if at limit (only matters for new inserts, not upsert updates)
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM device_tokens WHERE id IN (
			SELECT id FROM device_tokens WHERE user_id = ?
			ORDER BY updated_at DESC
			LIMIT -1 OFFSET ?
		)
	`, token.UserID, MaxTokensPerUser-1)
	if err != nil {
		return fmt.Errorf("evicting oldest token: %w", err)
	}

	return r.db.QueryRowContext(ctx, `
		INSERT INTO device_tokens (id, user_id, token, platform, device_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, token) DO UPDATE SET
			platform = excluded.platform,
			device_id = excluded.device_id,
			updated_at = excluded.updated_at
		RETURNING id
	`, token.ID, token.UserID, token.Token, token.Platform, token.DeviceID,
		now.Format(time.RFC3339), now.Format(time.RFC3339)).Scan(&token.ID)
}

// Delete removes a specific token for a user by token value.
func (r *Repository) Delete(ctx context.Context, userID, tokenValue string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE user_id = ? AND token = ?`, userID, tokenValue)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// DeleteByID removes a device token by its record ID, scoped to a user.
func (r *Repository) DeleteByID(ctx context.Context, userID, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// ListByUserID returns all device tokens for a user.
func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]*DeviceToken, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, token, platform, device_id, created_at, updated_at
		FROM device_tokens WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*DeviceToken
	for rows.Next() {
		t, err := scanDeviceToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// CleanupStale removes device tokens that haven't been updated since the given time.
func (r *Repository) CleanupStale(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE updated_at < ?`, olderThan.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDeviceToken(row scanner) (*DeviceToken, error) {
	var t DeviceToken
	var createdAt, updatedAt string

	err := row.Scan(&t.ID, &t.UserID, &t.Token, &t.Platform, &t.DeviceID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	t.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	t.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}
	return &t, nil
}
