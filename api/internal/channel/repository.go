package channel

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrChannelNotFound     = errors.New("channel not found")
	ErrNotChannelMember    = errors.New("not a member of this channel")
	ErrAlreadyMember       = errors.New("already a member of this channel")
	ErrChannelArchived     = errors.New("channel is archived")
	ErrCannotLeaveChannel  = errors.New("cannot leave this channel")
	ErrDMAlreadyExists     = errors.New("DM channel already exists")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, channel *Channel, creatorID string) error {
	channel.ID = ulid.Make().String()
	now := time.Now().UTC()
	channel.CreatedAt = now
	channel.UpdatedAt = now
	channel.CreatedBy = &creatorID

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO channels (id, workspace_id, name, description, type, dm_participant_hash, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, channel.ID, channel.WorkspaceID, channel.Name, channel.Description, channel.Type, channel.DMParticipantHash, channel.CreatedBy, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Add creator as member
	membershipID := ulid.Make().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO channel_memberships (id, user_id, channel_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, membershipID, creatorID, channel.ID, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) CreateDM(ctx context.Context, workspaceID string, userIDs []string) (*Channel, error) {
	hash := computeDMHash(userIDs)

	// Check if DM already exists
	var existingID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM channels
		WHERE workspace_id = ? AND dm_participant_hash = ?
	`, workspaceID, hash).Scan(&existingID)
	if err == nil {
		return r.GetByID(ctx, existingID)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	channelType := TypeDM
	if len(userIDs) > 2 {
		channelType = TypeGroupDM
	}

	channel := &Channel{
		ID:                ulid.Make().String(),
		WorkspaceID:       workspaceID,
		Name:              "Direct Message",
		Type:              channelType,
		DMParticipantHash: &hash,
	}
	now := time.Now().UTC()
	channel.CreatedAt = now
	channel.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO channels (id, workspace_id, name, type, dm_participant_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, channel.ID, channel.WorkspaceID, channel.Name, channel.Type, channel.DMParticipantHash, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	// Add all participants as members
	for _, userID := range userIDs {
		membershipID := ulid.Make().String()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO channel_memberships (id, user_id, channel_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, membershipID, userID, channel.ID, now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return channel, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Channel, error) {
	return r.scanChannel(r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, type, dm_participant_hash, archived_at, created_by, created_at, updated_at
		FROM channels WHERE id = ?
	`, id))
}

func (r *Repository) Update(ctx context.Context, channel *Channel) error {
	channel.UpdatedAt = time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE channels SET name = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, channel.Name, channel.Description, channel.UpdatedAt.Format(time.RFC3339), channel.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrChannelNotFound
	}
	return nil
}

func (r *Repository) Archive(ctx context.Context, channelID string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE channels SET archived_at = ?, updated_at = ?
		WHERE id = ? AND archived_at IS NULL
	`, now.Format(time.RFC3339), now.Format(time.RFC3339), channelID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrChannelNotFound
	}
	return nil
}

func (r *Repository) ListForWorkspace(ctx context.Context, workspaceID, userID string) ([]ChannelWithMembership, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.workspace_id, c.name, c.description, c.type, c.dm_participant_hash, c.archived_at, c.created_by, c.created_at, c.updated_at,
		       cm.channel_role, cm.last_read_message_id,
		       COALESCE((
		           SELECT COUNT(*) FROM messages m
		           WHERE m.channel_id = c.id
		             AND m.thread_parent_id IS NULL
		             AND m.deleted_at IS NULL
		             AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
		       ), 0) as unread_count
		FROM channels c
		LEFT JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
		WHERE c.workspace_id = ? AND c.archived_at IS NULL
		  AND (c.type = 'public' OR cm.id IS NOT NULL)
		ORDER BY c.name
	`, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []ChannelWithMembership
	for rows.Next() {
		var c ChannelWithMembership
		var description, dmHash, archivedAt, createdBy, channelRole, lastReadID sql.NullString
		var createdAt, updatedAt string
		var unreadCount int

		err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &description, &c.Type, &dmHash, &archivedAt, &createdBy, &createdAt, &updatedAt,
			&channelRole, &lastReadID, &unreadCount)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			c.Description = &description.String
		}
		if dmHash.Valid {
			c.DMParticipantHash = &dmHash.String
		}
		if archivedAt.Valid {
			t, _ := time.Parse(time.RFC3339, archivedAt.String)
			c.ArchivedAt = &t
		}
		if createdBy.Valid {
			c.CreatedBy = &createdBy.String
		}
		if channelRole.Valid {
			c.ChannelRole = &channelRole.String
		}
		if lastReadID.Valid {
			c.LastReadMessageID = &lastReadID.String
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		c.UnreadCount = unreadCount

		channels = append(channels, c)
	}

	return channels, rows.Err()
}

func (r *Repository) GetMembership(ctx context.Context, userID, channelID string) (*ChannelMembership, error) {
	var m ChannelMembership
	var channelRole, lastReadID sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, channel_id, channel_role, last_read_message_id, created_at, updated_at
		FROM channel_memberships WHERE user_id = ? AND channel_id = ?
	`, userID, channelID).Scan(&m.ID, &m.UserID, &m.ChannelID, &channelRole, &lastReadID, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotChannelMember
	}
	if err != nil {
		return nil, err
	}

	if channelRole.Valid {
		m.ChannelRole = &channelRole.String
	}
	if lastReadID.Valid {
		m.LastReadMessageID = &lastReadID.String
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &m, nil
}

func (r *Repository) AddMember(ctx context.Context, userID, channelID string, role *string) (*ChannelMembership, error) {
	// Check if channel exists and is not archived
	channel, err := r.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel.ArchivedAt != nil {
		return nil, ErrChannelArchived
	}

	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, userID, channelID, role, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrAlreadyMember
		}
		return nil, err
	}

	return &ChannelMembership{
		ID:          id,
		UserID:      userID,
		ChannelID:   channelID,
		ChannelRole: role,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *Repository) RemoveMember(ctx context.Context, userID, channelID string) error {
	// Check channel type - can't leave DMs
	channel, err := r.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel.Type == TypeDM {
		return ErrCannotLeaveChannel
	}

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM channel_memberships WHERE user_id = ? AND channel_id = ?
	`, userID, channelID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotChannelMember
	}
	return nil
}

func (r *Repository) ListMembers(ctx context.Context, channelID string) ([]MemberInfo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.email, u.display_name, u.avatar_url, cm.channel_role
		FROM channel_memberships cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.channel_id = ?
		ORDER BY u.display_name
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []MemberInfo
	for rows.Next() {
		var m MemberInfo
		var avatarURL, channelRole sql.NullString

		err := rows.Scan(&m.UserID, &m.Email, &m.DisplayName, &avatarURL, &channelRole)
		if err != nil {
			return nil, err
		}

		if avatarURL.Valid {
			m.AvatarURL = &avatarURL.String
		}
		if channelRole.Valid {
			m.ChannelRole = &channelRole.String
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

func (r *Repository) UpdateLastRead(ctx context.Context, userID, channelID, messageID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE channel_memberships SET last_read_message_id = ?, updated_at = ?
		WHERE user_id = ? AND channel_id = ?
	`, messageID, now.Format(time.RFC3339), userID, channelID)
	return err
}

func (r *Repository) GetLatestMessageID(ctx context.Context, channelID string) (string, error) {
	var messageID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM messages
		WHERE channel_id = ? AND thread_parent_id IS NULL AND deleted_at IS NULL
		ORDER BY id DESC LIMIT 1
	`, channelID).Scan(&messageID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return messageID, err
}

func (r *Repository) GetPreviousMessageID(ctx context.Context, messageID string) (string, error) {
	// First get the channel ID for the message
	var channelID string
	err := r.db.QueryRowContext(ctx, `
		SELECT channel_id FROM messages WHERE id = ?
	`, messageID).Scan(&channelID)
	if err != nil {
		return "", err
	}

	// Get the message before the given one (comparing by ULID which are lexicographically sortable)
	var prevID string
	err = r.db.QueryRowContext(ctx, `
		SELECT id FROM messages
		WHERE channel_id = ? AND thread_parent_id IS NULL AND deleted_at IS NULL AND id < ?
		ORDER BY id DESC LIMIT 1
	`, channelID, messageID).Scan(&prevID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return prevID, err
}

func (r *Repository) ListMemberChannelIDs(ctx context.Context, workspaceID, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id FROM channels c
		JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
		WHERE c.workspace_id = ? AND c.archived_at IS NULL
	`, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channelIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		channelIDs = append(channelIDs, id)
	}
	return channelIDs, rows.Err()
}

func (r *Repository) GetMemberUserIDs(ctx context.Context, channelID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id FROM channel_memberships WHERE channel_id = ?
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}

func (r *Repository) scanChannel(row *sql.Row) (*Channel, error) {
	var c Channel
	var description, dmHash, archivedAt, createdBy sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&c.ID, &c.WorkspaceID, &c.Name, &description, &c.Type, &dmHash, &archivedAt, &createdBy, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrChannelNotFound
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		c.Description = &description.String
	}
	if dmHash.Valid {
		c.DMParticipantHash = &dmHash.String
	}
	if archivedAt.Valid {
		t, _ := time.Parse(time.RFC3339, archivedAt.String)
		c.ArchivedAt = &t
	}
	if createdBy.Valid {
		c.CreatedBy = &createdBy.String
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &c, nil
}

func computeDMHash(userIDs []string) string {
	sorted := make([]string, len(userIDs))
	copy(sorted, userIDs)
	sort.Strings(sorted)
	combined := strings.Join(sorted, ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key"))
}
