package voice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// ErrChannelFull is returned when a voice channel has reached its participant limit.
var ErrChannelFull = errors.New("voice channel is full")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AddParticipant(ctx context.Context, channelID, userID string) (*Participant, error) {
	p := &Participant{
		ID:        ulid.Make().String(),
		ChannelID: channelID,
		UserID:    userID,
		JoinedAt:  time.Now().UTC(),
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO voice_participants (id, channel_id, user_id, joined_at) VALUES (?, ?, ?, ?)`,
		p.ID, p.ChannelID, p.UserID, p.JoinedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("inserting voice participant: %w", err)
	}
	return p, nil
}

// AddParticipantIfUnderLimit atomically checks the participant count and inserts
// a new participant within a single transaction, preventing TOCTOU races.
func (r *Repository) AddParticipantIfUnderLimit(ctx context.Context, channelID, userID string, maxPerChannel int) (*Participant, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	var count int
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM voice_participants WHERE channel_id = ?`, channelID,
	).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("counting voice participants: %w", err)
	}
	if maxPerChannel > 0 && count >= maxPerChannel {
		return nil, ErrChannelFull
	}

	p := &Participant{
		ID:        ulid.Make().String(),
		ChannelID: channelID,
		UserID:    userID,
		JoinedAt:  time.Now().UTC(),
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO voice_participants (id, channel_id, user_id, joined_at) VALUES (?, ?, ?, ?)`,
		p.ID, p.ChannelID, p.UserID, p.JoinedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("inserting voice participant: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing voice participant: %w", err)
	}
	return p, nil
}

func (r *Repository) RemoveParticipant(ctx context.Context, channelID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM voice_participants WHERE channel_id = ? AND user_id = ?`,
		channelID, userID,
	)
	if err != nil {
		return fmt.Errorf("removing voice participant: %w", err)
	}
	return nil
}

func (r *Repository) GetParticipants(ctx context.Context, channelID string) ([]Participant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT vp.id, vp.channel_id, vp.user_id, vp.is_muted, vp.is_deafened, vp.is_server_muted, vp.joined_at,
		        u.display_name, u.avatar_url
		 FROM voice_participants vp
		 JOIN users u ON u.id = vp.user_id
		 WHERE vp.channel_id = ?
		 ORDER BY vp.joined_at ASC`,
		channelID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying voice participants: %w", err)
	}
	defer rows.Close()

	var participants []Participant
	for rows.Next() {
		var p Participant
		var joinedAt string
		if err := rows.Scan(&p.ID, &p.ChannelID, &p.UserID, &p.IsMuted, &p.IsDeafened, &p.IsServerMuted, &joinedAt, &p.DisplayName, &p.AvatarURL); err != nil {
			return nil, fmt.Errorf("scanning voice participant: %w", err)
		}
		var parseErr error
		p.JoinedAt, parseErr = time.Parse(time.RFC3339, joinedAt)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing joined_at: %w", parseErr)
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

func (r *Repository) UpdateMuteState(ctx context.Context, channelID, userID string, muted, deafened, serverMuted bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE voice_participants SET is_muted = ?, is_deafened = ?, is_server_muted = ? WHERE channel_id = ? AND user_id = ?`,
		muted, deafened, serverMuted, channelID, userID,
	)
	if err != nil {
		return fmt.Errorf("updating voice mute state: %w", err)
	}
	return nil
}

func (r *Repository) RemoveAllParticipants(ctx context.Context, channelID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM voice_participants WHERE channel_id = ?`,
		channelID,
	)
	if err != nil {
		return fmt.Errorf("removing all voice participants: %w", err)
	}
	return nil
}

func (r *Repository) IsParticipant(ctx context.Context, channelID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM voice_participants WHERE channel_id = ? AND user_id = ?`,
		channelID, userID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) GetParticipantCount(ctx context.Context, channelID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM voice_participants WHERE channel_id = ?`,
		channelID,
	).Scan(&count)
	return count, err
}

func (r *Repository) GetParticipant(ctx context.Context, channelID, userID string) (*Participant, error) {
	var p Participant
	var joinedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, channel_id, user_id, is_muted, is_deafened, is_server_muted, joined_at
		 FROM voice_participants WHERE channel_id = ? AND user_id = ?`,
		channelID, userID,
	).Scan(&p.ID, &p.ChannelID, &p.UserID, &p.IsMuted, &p.IsDeafened, &p.IsServerMuted, &joinedAt)
	if err != nil {
		return nil, err
	}
	var parseErr error
	p.JoinedAt, parseErr = time.Parse(time.RFC3339, joinedAt)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing joined_at: %w", parseErr)
	}
	return &p, nil
}

// RemoveStaleParticipants removes participants that have been connected for longer than maxAge.
// This is used as a cleanup mechanism for participants whose disconnect was not properly handled.
func (r *Repository) RemoveStaleParticipants(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM voice_participants WHERE joined_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("removing stale voice participants: %w", err)
	}
	return result.RowsAffected()
}

// FlushAll removes all voice participants. Used at startup to clean up stale
// state from a previous server process (no peer connections survive a restart).
func (r *Repository) FlushAll(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM voice_participants`)
	if err != nil {
		return fmt.Errorf("flushing voice participants: %w", err)
	}
	return nil
}
