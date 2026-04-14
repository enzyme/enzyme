package notification

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// Notification levels
const (
	NotifyAll      = "all"      // Notify on all messages
	NotifyMentions = "mentions" // Only notify on @mentions
	NotifyNone     = "none"     // No notifications
)

// NotificationPreference represents a user's notification settings for a channel
type NotificationPreference struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ChannelID    string    `json:"channel_id"`
	NotifyLevel  string    `json:"notify_level"`
	EmailEnabled bool      `json:"email_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

var ErrPreferenceNotFound = errors.New("notification preference not found")

// PreferencesRepository handles notification preference persistence
type PreferencesRepository struct {
	db *sql.DB
}

// NewPreferencesRepository creates a new preferences repository
func NewPreferencesRepository(db *sql.DB) *PreferencesRepository {
	return &PreferencesRepository{db: db}
}

// Get retrieves notification preferences for a user and channel
func (r *PreferencesRepository) Get(ctx context.Context, userID, channelID string) (*NotificationPreference, error) {
	var pref NotificationPreference
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, channel_id, notify_level, email_enabled, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = ? AND channel_id = ?
	`, userID, channelID).Scan(
		&pref.ID, &pref.UserID, &pref.ChannelID, &pref.NotifyLevel, &pref.EmailEnabled,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPreferenceNotFound
	}
	if err != nil {
		return nil, err
	}

	if pref.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	if pref.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}

	return &pref, nil
}

// GetOrDefault retrieves preferences or returns defaults based on channel type
func (r *PreferencesRepository) GetOrDefault(ctx context.Context, userID, channelID, channelType string) (*NotificationPreference, error) {
	pref, err := r.Get(ctx, userID, channelID)
	if err == nil {
		return pref, nil
	}
	if !errors.Is(err, ErrPreferenceNotFound) {
		return nil, err
	}

	// Return default preferences based on channel type
	defaultLevel := NotifyMentions
	if channelType == "dm" || channelType == "group_dm" {
		defaultLevel = NotifyAll
	}

	return &NotificationPreference{
		UserID:       userID,
		ChannelID:    channelID,
		NotifyLevel:  defaultLevel,
		EmailEnabled: true,
	}, nil
}

// Upsert creates or updates notification preferences
func (r *PreferencesRepository) Upsert(ctx context.Context, pref *NotificationPreference) error {
	now := time.Now().UTC().Format(time.RFC3339)
	id := ulid.Make().String()

	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO notification_preferences (id, user_id, channel_id, notify_level, email_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, channel_id) DO UPDATE SET
			notify_level = excluded.notify_level,
			email_enabled = excluded.email_enabled,
			updated_at = excluded.updated_at
		RETURNING id, user_id, channel_id, notify_level, email_enabled, created_at, updated_at
	`, id, pref.UserID, pref.ChannelID, pref.NotifyLevel, pref.EmailEnabled, now, now).Scan(
		&pref.ID, &pref.UserID, &pref.ChannelID, &pref.NotifyLevel, &pref.EmailEnabled,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return err
	}

	if pref.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return fmt.Errorf("parsing created_at: %w", err)
	}
	if pref.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return fmt.Errorf("parsing updated_at: %w", err)
	}

	return nil
}

// Delete removes notification preferences
func (r *PreferencesRepository) Delete(ctx context.Context, userID, channelID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM notification_preferences WHERE user_id = ? AND channel_id = ?
	`, userID, channelID)
	return err
}

// ListForUser returns all notification preferences for a user
func (r *PreferencesRepository) ListForUser(ctx context.Context, userID string) ([]NotificationPreference, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, channel_id, notify_level, email_enabled, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []NotificationPreference
	for rows.Next() {
		var pref NotificationPreference
		var createdAt, updatedAt string

		err := rows.Scan(&pref.ID, &pref.UserID, &pref.ChannelID, &pref.NotifyLevel, &pref.EmailEnabled,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		if pref.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("parsing created_at: %w", err)
		}
		if pref.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
			return nil, fmt.Errorf("parsing updated_at: %w", err)
		}
		prefs = append(prefs, pref)
	}

	return prefs, rows.Err()
}
