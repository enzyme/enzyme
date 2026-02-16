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
	ErrChannelNotFound      = errors.New("channel not found")
	ErrNotChannelMember     = errors.New("not a member of this channel")
	ErrAlreadyMember        = errors.New("already a member of this channel")
	ErrChannelArchived      = errors.New("channel is archived")
	ErrCannotLeaveChannel   = errors.New("cannot leave this channel")
	ErrDMAlreadyExists      = errors.New("DM channel already exists")
	ErrCannotLeaveDefault   = errors.New("cannot leave the default channel")
	ErrCannotArchiveDefault = errors.New("cannot archive the default channel")
	ErrChannelNameTaken     = errors.New("channel name already taken")
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

	isDefault := 0
	if channel.IsDefault {
		isDefault = 1
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO channels (id, workspace_id, name, description, type, dm_participant_hash, is_default, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, channel.ID, channel.WorkspaceID, channel.Name, channel.Description, channel.Type, channel.DMParticipantHash, isDefault, channel.CreatedBy, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Add creator as admin member
	membershipID := ulid.Make().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, membershipID, creatorID, channel.ID, "admin", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) CreateDM(ctx context.Context, workspaceID string, userIDs []string) (*Channel, error) {
	hash := ComputeDMHash(userIDs)

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
			INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, membershipID, userID, channel.ID, "poster", now.Format(time.RFC3339), now.Format(time.RFC3339))
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
		SELECT id, workspace_id, name, description, type, dm_participant_hash, is_default, archived_at, created_by, created_at, updated_at
		FROM channels WHERE id = ?
	`, id))
}

func (r *Repository) GetByWorkspaceAndName(ctx context.Context, workspaceID, name string) (*Channel, error) {
	ch, err := r.scanChannel(r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, type, dm_participant_hash, is_default, archived_at, created_by, created_at, updated_at
		FROM channels WHERE workspace_id = ? AND name = ? AND type IN ('public', 'private')
	`, workspaceID, name))
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return ch, nil
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
	// Check if channel is default
	channel, err := r.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel.IsDefault {
		return ErrCannotArchiveDefault
	}

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
		SELECT c.id, c.workspace_id, c.name, c.description, c.type, c.dm_participant_hash, c.is_default, c.archived_at, c.created_by, c.created_at, c.updated_at,
		       cm.channel_role, cm.last_read_message_id, COALESCE(cm.is_starred, 0) as is_starred,
		       COALESCE((
		           SELECT COUNT(*) FROM messages m
		           WHERE m.channel_id = c.id
		             AND m.thread_parent_id IS NULL
		             AND m.deleted_at IS NULL
		             AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
		       ), 0) as unread_count,
		       COALESCE((
		           SELECT COUNT(*) FROM messages m
		           WHERE m.channel_id = c.id
		             AND m.thread_parent_id IS NULL
		             AND m.deleted_at IS NULL
		             AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
		             AND CASE
		               WHEN c.type IN ('dm', 'group_dm') THEN 1
		               WHEN np.notify_level = 'none' THEN 0
		               WHEN np.notify_level = 'all' THEN 1
		               WHEN np.notify_level = 'mentions' OR np.notify_level IS NULL THEN
		                 EXISTS (
		                   SELECT 1 FROM json_each(m.mentions) je
		                   WHERE je.value = ? OR je.value IN ('@channel', '@here', '@everyone')
		                 )
		               ELSE 0
		             END = 1
		       ), 0) as notification_count
		FROM channels c
		LEFT JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
		LEFT JOIN notification_preferences np ON np.channel_id = c.id AND np.user_id = ?
		WHERE c.workspace_id = ? AND c.archived_at IS NULL
		  AND (c.type = 'public' OR cm.id IS NOT NULL)
		ORDER BY c.name
	`, userID, userID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []ChannelWithMembership
	var dmChannelIDs []string
	channelIndex := make(map[string]int)

	for rows.Next() {
		var c ChannelWithMembership
		var description, dmHash, archivedAt, createdBy, channelRole, lastReadID sql.NullString
		var createdAt, updatedAt string
		var isDefault int
		var isStarred int
		var unreadCount int
		var notificationCount int

		err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &description, &c.Type, &dmHash, &isDefault, &archivedAt, &createdBy, &createdAt, &updatedAt,
			&channelRole, &lastReadID, &isStarred, &unreadCount, &notificationCount)
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
		c.NotificationCount = notificationCount
		c.IsStarred = isStarred != 0
		c.IsDefault = isDefault != 0

		// Track DM channels for participant lookup
		if c.Type == TypeDM || c.Type == TypeGroupDM {
			channelIndex[c.ID] = len(channels)
			dmChannelIDs = append(dmChannelIDs, c.ID)
		}

		channels = append(channels, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch DM participants for DM channels
	if len(dmChannelIDs) > 0 {
		if err := r.fetchDMParticipants(ctx, channels, dmChannelIDs, channelIndex, userID); err != nil {
			return nil, err
		}
	}

	return channels, nil
}

// fetchDMParticipants loads participant info for DM channels, excluding the current user
func (r *Repository) fetchDMParticipants(ctx context.Context, channels []ChannelWithMembership, dmChannelIDs []string, channelIndex map[string]int, currentUserID string) error {
	if len(dmChannelIDs) == 0 {
		return nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(dmChannelIDs))
	args := make([]interface{}, 0, len(dmChannelIDs)+1)
	for i, id := range dmChannelIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	args = append(args, currentUserID)

	query := `
		SELECT cm.channel_id, u.id, u.email, u.display_name, u.avatar_url, cm.channel_role
		FROM channel_memberships cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.channel_id IN (` + strings.Join(placeholders, ",") + `)
		  AND cm.user_id != ?
		ORDER BY u.display_name
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var channelID string
		var member MemberInfo
		var avatarURL, channelRole sql.NullString

		if err := rows.Scan(&channelID, &member.UserID, &member.Email, &member.DisplayName, &avatarURL, &channelRole); err != nil {
			return err
		}

		if avatarURL.Valid {
			member.AvatarURL = &avatarURL.String
		}
		if channelRole.Valid {
			member.ChannelRole = &channelRole.String
		}

		if idx, ok := channelIndex[channelID]; ok {
			channels[idx].DMParticipants = append(channels[idx].DMParticipants, member)
		}
	}

	return rows.Err()
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

func (r *Repository) UpdateMemberRole(ctx context.Context, userID, channelID string, role *string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE channel_memberships SET channel_role = ?, updated_at = ?
		WHERE user_id = ? AND channel_id = ?
	`, role, now.Format(time.RFC3339), userID, channelID)
	return err
}

func (r *Repository) RemoveMember(ctx context.Context, userID, channelID string) error {
	// Check channel type - can't leave 1:1 DMs or default channels
	ch, err := r.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.Type == TypeDM {
		return ErrCannotLeaveChannel
	}
	if ch.IsDefault {
		return ErrCannotLeaveDefault
	}

	// Group DMs need special handling (hash update, possible type conversion)
	if ch.Type == TypeGroupDM {
		return r.LeaveGroupDM(ctx, userID, channelID)
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

// AddMemberToDM adds a member to a DM or group DM, updating the hash and converting dm -> group_dm if needed.
func (r *Repository) AddMemberToDM(ctx context.Context, channelID, userID string, currentMemberIDs []string) (*Channel, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Insert membership
	membershipID := ulid.Make().String()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO channel_memberships (id, user_id, channel_id, channel_role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, membershipID, userID, channelID, "poster", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrAlreadyMember
		}
		return nil, err
	}

	// Compute new hash with all members including the new one
	allMemberIDs := append(currentMemberIDs, userID)
	newHash := ComputeDMHash(allMemberIDs)

	// Determine new type
	newType := TypeDM
	if len(allMemberIDs) >= 3 {
		newType = TypeGroupDM
	}

	// Update channel hash and type
	_, err = tx.ExecContext(ctx, `
		UPDATE channels SET type = ?, dm_participant_hash = ?, updated_at = ?
		WHERE id = ?
	`, newType, newHash, now.Format(time.RFC3339), channelID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, channelID)
}

// LeaveGroupDM removes a member from a group DM, updating the hash and converting group_dm -> dm if only 2 remain.
func (r *Repository) LeaveGroupDM(ctx context.Context, userID, channelID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Remove membership
	result, err := tx.ExecContext(ctx, `
		DELETE FROM channel_memberships WHERE user_id = ? AND channel_id = ?
	`, userID, channelID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotChannelMember
	}

	// Get remaining member IDs
	memberRows, err := tx.QueryContext(ctx, `
		SELECT user_id FROM channel_memberships WHERE channel_id = ?
	`, channelID)
	if err != nil {
		return err
	}
	defer memberRows.Close()

	var remainingIDs []string
	for memberRows.Next() {
		var id string
		if err := memberRows.Scan(&id); err != nil {
			return err
		}
		remainingIDs = append(remainingIDs, id)
	}
	if err := memberRows.Err(); err != nil {
		return err
	}

	// Recompute hash
	newHash := ComputeDMHash(remainingIDs)

	// Convert to dm if exactly 2 remain
	newType := TypeGroupDM
	if len(remainingIDs) == 2 {
		newType = TypeDM
	}

	// Update channel
	_, err = tx.ExecContext(ctx, `
		UPDATE channels SET type = ?, dm_participant_hash = ?, updated_at = ?
		WHERE id = ?
	`, newType, newHash, now.Format(time.RFC3339), channelID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ConvertToChannel converts a group DM to a channel, clearing the DM hash,
// updating name/description/type, and promoting the converting user to admin.
func (r *Repository) ConvertToChannel(ctx context.Context, channelID, name string, description *string, createdBy string, channelType string) (*Channel, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Update channel: set type, name, description, clear hash, set created_by
	_, err = tx.ExecContext(ctx, `
		UPDATE channels SET type = ?, name = ?, description = ?, dm_participant_hash = NULL, created_by = ?, updated_at = ?
		WHERE id = ?
	`, channelType, name, description, createdBy, now.Format(time.RFC3339), channelID)
	if err != nil {
		return nil, err
	}

	// Promote the converting user to admin
	_, err = tx.ExecContext(ctx, `
		UPDATE channel_memberships SET channel_role = ?, updated_at = ?
		WHERE user_id = ? AND channel_id = ?
	`, ChannelRoleAdmin, now.Format(time.RFC3339), createdBy, channelID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, channelID)
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

func (r *Repository) StarChannel(ctx context.Context, userID, channelID string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE channel_memberships SET is_starred = 1, updated_at = ?
		WHERE user_id = ? AND channel_id = ?
	`, now.Format(time.RFC3339), userID, channelID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotChannelMember
	}
	return nil
}

func (r *Repository) UnstarChannel(ctx context.Context, userID, channelID string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE channel_memberships SET is_starred = 0, updated_at = ?
		WHERE user_id = ? AND channel_id = ?
	`, now.Format(time.RFC3339), userID, channelID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotChannelMember
	}
	return nil
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

// GetDefaultChannel returns the default channel for a workspace
func (r *Repository) GetDefaultChannel(ctx context.Context, workspaceID string) (*Channel, error) {
	return r.scanChannel(r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, description, type, dm_participant_hash, is_default, archived_at, created_by, created_at, updated_at
		FROM channels WHERE workspace_id = ? AND is_default = 1
	`, workspaceID))
}

// ListRecentDMs returns DM channels that have messages since the given time
func (r *Repository) ListRecentDMs(ctx context.Context, workspaceID, userID string, since time.Time, limit int) ([]ChannelWithMembership, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.workspace_id, c.name, c.description, c.type, c.dm_participant_hash, c.is_default, c.archived_at, c.created_by, c.created_at, c.updated_at,
		       cm.channel_role, cm.last_read_message_id, COALESCE(cm.is_starred, 0) as is_starred,
		       COALESCE((
		           SELECT COUNT(*) FROM messages m
		           WHERE m.channel_id = c.id
		             AND m.thread_parent_id IS NULL
		             AND m.deleted_at IS NULL
		             AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
		       ), 0) as unread_count,
		       (SELECT MAX(m.created_at) FROM messages m WHERE m.channel_id = c.id AND m.deleted_at IS NULL) as last_message_at
		FROM channels c
		JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
		WHERE c.workspace_id = ? AND c.archived_at IS NULL
		  AND c.type IN ('dm', 'group_dm')
		  AND EXISTS (
		      SELECT 1 FROM messages m
		      WHERE m.channel_id = c.id
		        AND m.deleted_at IS NULL
		        AND m.created_at >= ?
		  )
		ORDER BY last_message_at DESC
		LIMIT ?
	`, userID, workspaceID, since.Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []ChannelWithMembership
	var dmChannelIDs []string
	channelIndex := make(map[string]int)

	for rows.Next() {
		var c ChannelWithMembership
		var description, dmHash, archivedAt, createdBy, channelRole, lastReadID, lastMsgAt sql.NullString
		var createdAt, updatedAt string
		var isDefault int
		var isStarred int
		var unreadCount int

		err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Name, &description, &c.Type, &dmHash, &isDefault, &archivedAt, &createdBy, &createdAt, &updatedAt,
			&channelRole, &lastReadID, &isStarred, &unreadCount, &lastMsgAt)
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
		c.NotificationCount = unreadCount // DMs always notify
		c.IsStarred = isStarred != 0
		c.IsDefault = isDefault != 0

		channelIndex[c.ID] = len(channels)
		dmChannelIDs = append(dmChannelIDs, c.ID)
		channels = append(channels, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch DM participants
	if len(dmChannelIDs) > 0 {
		if err := r.fetchDMParticipants(ctx, channels, dmChannelIDs, channelIndex, userID); err != nil {
			return nil, err
		}
	}

	return channels, nil
}

// CreateDefaultChannel creates the default #general channel for a workspace
func (r *Repository) CreateDefaultChannel(ctx context.Context, workspaceID, creatorID string) (*Channel, error) {
	ch := &Channel{
		WorkspaceID: workspaceID,
		Name:        DefaultChannelName,
		Type:        TypePublic,
		IsDefault:   true,
	}
	if err := r.Create(ctx, ch, creatorID); err != nil {
		return nil, err
	}
	return ch, nil
}

func (r *Repository) scanChannel(row *sql.Row) (*Channel, error) {
	var c Channel
	var description, dmHash, archivedAt, createdBy sql.NullString
	var createdAt, updatedAt string
	var isDefault int

	err := row.Scan(&c.ID, &c.WorkspaceID, &c.Name, &description, &c.Type, &dmHash, &isDefault, &archivedAt, &createdBy, &createdAt, &updatedAt)
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
	c.IsDefault = isDefault != 0

	return &c, nil
}

func ComputeDMHash(userIDs []string) string {
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
