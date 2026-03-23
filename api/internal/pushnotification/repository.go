package pushnotification

import (
	"context"
	"database/sql"
	"errors"
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
func (r *Repository) Upsert(ctx context.Context, token *DeviceToken) error {
	now := time.Now().UTC()
	if token.ID == "" {
		token.ID = ulid.Make().String()
	}
	token.CreatedAt = now
	token.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO device_tokens (id, user_id, token, platform, device_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, token) DO UPDATE SET
			platform = excluded.platform,
			device_id = excluded.device_id,
			updated_at = excluded.updated_at
	`, token.ID, token.UserID, token.Token, token.Platform, token.DeviceID,
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// On conflict, the ID remains the original row's ID. Fetch it.
	row := r.db.QueryRowContext(ctx, `SELECT id FROM device_tokens WHERE user_id = ? AND token = ?`, token.UserID, token.Token)
	return row.Scan(&token.ID)
}

// Delete removes a specific token for a user.
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

// DeleteByDeviceID removes all tokens for a specific device belonging to a user.
func (r *Repository) DeleteByDeviceID(ctx context.Context, userID, deviceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE user_id = ? AND device_id = ?`, userID, deviceID)
	return err
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

// HasTokens returns true if the user has at least one registered device token.
func (r *Repository) HasTokens(ctx context.Context, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM device_tokens WHERE user_id = ? LIMIT 1`, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteToken removes a token by its value (regardless of user). Used for relay invalid_token cleanup.
func (r *Repository) DeleteToken(ctx context.Context, tokenValue string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE token = ?`, tokenValue)
	return err
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
	Scan(dest ...interface{}) error
}

func scanDeviceToken(row scanner) (*DeviceToken, error) {
	var t DeviceToken
	var createdAt, updatedAt string

	err := row.Scan(&t.ID, &t.UserID, &t.Token, &t.Platform, &t.DeviceID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &t, nil
}
