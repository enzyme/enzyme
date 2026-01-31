package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
)

type PasswordResetRepo struct {
	db *sql.DB
}

func NewPasswordResetRepo(db *sql.DB) *PasswordResetRepo {
	return &PasswordResetRepo{db: db}
}

func (r *PasswordResetRepo) Create(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO password_resets (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, userID, token, expiresAt.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

func (r *PasswordResetRepo) GetByToken(ctx context.Context, token string) (*PasswordReset, error) {
	var reset PasswordReset
	var expiresAt, usedAt sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token, expires_at, used_at
		FROM password_resets WHERE token = ?
	`, token).Scan(&reset.ID, &reset.UserID, &reset.Token, &expiresAt, &usedAt)
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		reset.ExpiresAt = t
	}
	if usedAt.Valid {
		t, _ := time.Parse(time.RFC3339, usedAt.String)
		reset.UsedAt = &t
	}

	return &reset, nil
}

func (r *PasswordResetRepo) MarkUsed(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE password_resets SET used_at = ? WHERE id = ?
	`, now.Format(time.RFC3339), id)
	return err
}
