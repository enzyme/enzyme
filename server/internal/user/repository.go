package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyInUse = errors.New("email already in use")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?)
	`, id, input.Email, input.PasswordHash, input.DisplayName, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrEmailAlreadyInUse
		}
		return nil, err
	}

	return &User{
		ID:          id,
		Email:       input.Email,
		DisplayName: input.DisplayName,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified_at, password_hash, display_name, avatar_url, status, created_at, updated_at
		FROM users WHERE id = ?
	`, id))
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		SELECT id, email, email_verified_at, password_hash, display_name, avatar_url, status, created_at, updated_at
		FROM users WHERE email = ?
	`, email))
}

func (r *Repository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET
			email = ?, email_verified_at = ?, display_name = ?, avatar_url = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, user.Email, formatNullableTime(user.EmailVerifiedAt), user.DisplayName, user.AvatarURL, user.Status, user.UpdatedAt.Format(time.RFC3339), user.ID)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?
	`, passwordHash, time.Now().UTC().Format(time.RFC3339), userID)
	return err
}

func (r *Repository) VerifyEmail(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET email_verified_at = ?, updated_at = ? WHERE id = ?
	`, now.Format(time.RFC3339), now.Format(time.RFC3339), userID)
	return err
}

func (r *Repository) scanUser(row *sql.Row) (*User, error) {
	var user User
	var emailVerifiedAt, avatarURL sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(
		&user.ID,
		&user.Email,
		&emailVerifiedAt,
		&user.PasswordHash,
		&user.DisplayName,
		&avatarURL,
		&user.Status,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	if emailVerifiedAt.Valid {
		t, _ := time.Parse(time.RFC3339, emailVerifiedAt.String)
		user.EmailVerifiedAt = &t
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &user, nil
}

func formatNullableTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// ResolveDisplayNames resolves display names to user IDs within a workspace.
// Returns a map of display_name (lowercase) -> user_id for all matched users.
func (r *Repository) ResolveDisplayNames(ctx context.Context, workspaceID string, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	// Build query with LOWER for case-insensitive matching
	query := `
		SELECT u.id, LOWER(u.display_name)
		FROM users u
		JOIN workspace_memberships wm ON wm.user_id = u.id
		WHERE wm.workspace_id = ? AND LOWER(u.display_name) IN (?`

	args := make([]interface{}, 0, len(names)+1)
	args = append(args, workspaceID)
	args = append(args, strings.ToLower(names[0]))

	for i := 1; i < len(names); i++ {
		query += ",?"
		args = append(args, strings.ToLower(names[i]))
	}
	query += ")"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var userID, lowerName string
		if err := rows.Scan(&userID, &lowerName); err != nil {
			return nil, err
		}
		result[lowerName] = userID
	}

	return result, rows.Err()
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
