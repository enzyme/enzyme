package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type EmailVerificationRepository interface {
	Create(ctx context.Context, userID string, token string, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*EmailVerification, error)
	DeleteForUser(ctx context.Context, userID string) error
}

type EmailVerification struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type EmailVerificationRepo struct {
	db *sql.DB
}

func NewEmailVerificationRepo(db *sql.DB) *EmailVerificationRepo {
	return &EmailVerificationRepo{db: db}
}

func (r *EmailVerificationRepo) Create(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Delete existing tokens for this user (only one active at a time)
	if _, err := tx.ExecContext(ctx, `DELETE FROM email_verifications WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("deleting existing tokens: %w", err)
	}

	now := time.Now().UTC()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO email_verifications (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, token, userID, expiresAt.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("inserting verification token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing verification token: %w", err)
	}
	return nil
}

func (r *EmailVerificationRepo) GetByToken(ctx context.Context, token string) (*EmailVerification, error) {
	var ev EmailVerification
	var expiresAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT user_id, token, expires_at
		FROM email_verifications WHERE token = ?
	`, token).Scan(&ev.UserID, &ev.Token, &expiresAt)
	if err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}
	ev.ExpiresAt = t

	return &ev, nil
}

func (r *EmailVerificationRepo) DeleteForUser(ctx context.Context, userID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM email_verifications WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("deleting verification tokens for user: %w", err)
	}
	return nil
}

func (r *EmailVerificationRepo) DeleteExpired(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM email_verifications WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("deleting expired verification tokens: %w", err)
	}
	return nil
}
