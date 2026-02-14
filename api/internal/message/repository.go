package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrMessageNotFound       = errors.New("message not found")
	ErrReactionExists        = errors.New("reaction already exists")
	ErrReactionNotFound      = errors.New("reaction not found")
	ErrCannotEditMessage     = errors.New("cannot edit this message")
	ErrCannotEditSystemMsg   = errors.New("cannot edit system messages")
	ErrCannotDeleteSystemMsg = errors.New("cannot delete system messages")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, msg *Message) error {
	msg.ID = ulid.Make().String()
	now := time.Now().UTC()
	msg.CreatedAt = now
	msg.UpdatedAt = now

	// Default type to user
	if msg.Type == "" {
		msg.Type = MessageTypeUser
	}

	// Serialize mentions to JSON
	mentionsJSON := "[]"
	if len(msg.Mentions) > 0 {
		data, err := json.Marshal(msg.Mentions)
		if err == nil {
			mentionsJSON = string(data)
		}
	}

	// Serialize system_event to JSON
	var systemEventJSON *string
	if msg.SystemEvent != nil {
		data, err := json.Marshal(msg.SystemEvent)
		if err == nil {
			s := string(data)
			systemEventJSON = &s
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (id, channel_id, user_id, content, type, system_event, mentions, thread_parent_id, also_send_to_channel, reply_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
	`, msg.ID, msg.ChannelID, msg.UserID, msg.Content, msg.Type, systemEventJSON, mentionsJSON, msg.ThreadParentID, msg.AlsoSendToChannel, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// Update parent's reply_count and last_reply_at if this is a thread reply
	if msg.ThreadParentID != nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE messages SET reply_count = reply_count + 1, last_reply_at = ?, updated_at = ?
			WHERE id = ?
		`, now.Format(time.RFC3339), now.Format(time.RFC3339), *msg.ThreadParentID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CreateSystemMessage creates a system message for channel events
func (r *Repository) CreateSystemMessage(ctx context.Context, channelID string, event *SystemEventData) (*Message, error) {
	// Build content based on event type
	content := ""
	switch event.EventType {
	case SystemEventUserJoined:
		content = "joined #" + event.ChannelName
	case SystemEventUserLeft:
		content = "left #" + event.ChannelName
	case SystemEventUserAdded:
		if event.ActorDisplayName != nil {
			content = "was added by " + *event.ActorDisplayName
		} else {
			content = "was added to #" + event.ChannelName
		}
	}

	msg := &Message{
		ChannelID:   channelID,
		UserID:      &event.UserID,
		Content:     content,
		Type:        MessageTypeSystem,
		SystemEvent: event,
	}

	if err := r.Create(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Message, error) {
	return r.scanMessage(r.db.QueryRowContext(ctx, `
		SELECT id, channel_id, user_id, content, type, system_event, thread_parent_id, also_send_to_channel, reply_count, last_reply_at, edited_at, deleted_at, created_at, updated_at
		FROM messages WHERE id = ?
	`, id))
}

func (r *Repository) GetByIDWithUser(ctx context.Context, id string) (*MessageWithUser, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
		FROM messages m
		LEFT JOIN users u ON u.id = m.user_id
		WHERE m.id = ?
	`, id)

	msg, err := r.scanMessageWithUser(row)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *Repository) Update(ctx context.Context, id, content string) error {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE messages SET content = ?, edited_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, content, now.Format(time.RFC3339), now.Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the message first to check if it's a thread reply
	var threadParentID sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT thread_parent_id FROM messages WHERE id = ? AND deleted_at IS NULL`, id).Scan(&threadParentID)
	if err == sql.ErrNoRows {
		return ErrMessageNotFound
	}
	if err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE messages SET deleted_at = ?, content = '[deleted]', updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now.Format(time.RFC3339), now.Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrMessageNotFound
	}

	// Decrement parent's reply_count if this is a thread reply
	if threadParentID.Valid {
		_, err = tx.ExecContext(ctx, `
			UPDATE messages SET reply_count = MAX(reply_count - 1, 0), updated_at = ?
			WHERE id = ?
		`, now.Format(time.RFC3339), threadParentID.String)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) List(ctx context.Context, channelID string, opts ListOptions) (*ListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}

	var query string
	var args []interface{}

	// Get top-level messages and thread replies marked as "also send to channel"
	if opts.Cursor == "" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = ? AND (m.thread_parent_id IS NULL OR m.also_send_to_channel = TRUE)
			ORDER BY m.id DESC
			LIMIT ?
		`
		args = []interface{}{channelID, opts.Limit + 1}
	} else if opts.Direction == "after" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = ? AND (m.thread_parent_id IS NULL OR m.also_send_to_channel = TRUE) AND m.id > ?
			ORDER BY m.id ASC
			LIMIT ?
		`
		args = []interface{}{channelID, opts.Cursor, opts.Limit + 1}
	} else {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = ? AND (m.thread_parent_id IS NULL OR m.also_send_to_channel = TRUE) AND m.id < ?
			ORDER BY m.id DESC
			LIMIT ?
		`
		args = []interface{}{channelID, opts.Cursor, opts.Limit + 1}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageWithUser
	for rows.Next() {
		msg, err := r.scanMessageWithUser(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(messages) > opts.Limit
	if hasMore {
		messages = messages[:opts.Limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}

	// Load reactions and thread participants for all messages
	if len(messages) > 0 {
		messageIDs := make([]string, len(messages))
		threadParentIDs := make([]string, 0)
		for i, m := range messages {
			messageIDs[i] = m.ID
			if m.ReplyCount > 0 {
				threadParentIDs = append(threadParentIDs, m.ID)
			}
		}
		reactions, err := r.getReactionsForMessages(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		for i := range messages {
			if r, ok := reactions[messages[i].ID]; ok {
				messages[i].Reactions = r
			}
		}

		// Load thread participants for messages with replies
		if len(threadParentIDs) > 0 {
			participants, err := r.getThreadParticipantsForMessages(ctx, threadParentIDs)
			if err != nil {
				return nil, err
			}
			for i := range messages {
				if p, ok := participants[messages[i].ID]; ok {
					messages[i].ThreadParticipants = p
				}
			}
		}
	}

	if messages == nil {
		messages = []MessageWithUser{}
	}

	return &ListResult{
		Messages:   messages,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (r *Repository) ListThread(ctx context.Context, parentID string, opts ListOptions) (*ListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}

	var query string
	var args []interface{}

	if opts.Cursor == "" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.thread_parent_id = ?
			ORDER BY m.id ASC
			LIMIT ?
		`
		args = []interface{}{parentID, opts.Limit + 1}
	} else {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.thread_parent_id = ? AND m.id > ?
			ORDER BY m.id ASC
			LIMIT ?
		`
		args = []interface{}{parentID, opts.Cursor, opts.Limit + 1}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageWithUser
	for rows.Next() {
		msg, err := r.scanMessageWithUser(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(messages) > opts.Limit
	if hasMore {
		messages = messages[:opts.Limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}

	// Load reactions
	if len(messages) > 0 {
		messageIDs := make([]string, len(messages))
		for i, m := range messages {
			messageIDs[i] = m.ID
		}
		reactions, err := r.getReactionsForMessages(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		for i := range messages {
			if r, ok := reactions[messages[i].ID]; ok {
				messages[i].Reactions = r
			}
		}
	}

	if messages == nil {
		messages = []MessageWithUser{}
	}

	return &ListResult{
		Messages:   messages,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (r *Repository) AddReaction(ctx context.Context, messageID, userID, emoji string) (*Reaction, error) {
	id := ulid.Make().String()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reactions (id, message_id, user_id, emoji, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, messageID, userID, emoji, now.Format(time.RFC3339))
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrReactionExists
		}
		return nil, err
	}

	return &Reaction{
		ID:        id,
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: now,
	}, nil
}

func (r *Repository) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM reactions WHERE message_id = ? AND user_id = ? AND emoji = ?
	`, messageID, userID, emoji)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrReactionNotFound
	}
	return nil
}

// GetReactionsForMessage returns reactions for a single message
func (r *Repository) GetReactionsForMessage(ctx context.Context, messageID string) ([]Reaction, error) {
	reactions, err := r.getReactionsForMessages(ctx, []string{messageID})
	if err != nil {
		return nil, err
	}
	return reactions[messageID], nil
}

// GetThreadParticipants returns thread participants for a single parent message
func (r *Repository) GetThreadParticipants(ctx context.Context, parentID string) ([]ThreadParticipant, error) {
	participants, err := r.getThreadParticipantsForMessages(ctx, []string{parentID})
	if err != nil {
		return nil, err
	}
	return participants[parentID], nil
}

func (r *Repository) getThreadParticipantsForMessages(ctx context.Context, messageIDs []string) (map[string][]ThreadParticipant, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	// Get distinct users who replied to each thread, ordered by first reply, limited to 3
	query := `
		SELECT m.thread_parent_id, m.user_id, COALESCE(u.display_name, '') as display_name, u.avatar_url
		FROM (
			SELECT thread_parent_id, user_id, MIN(id) as first_reply_id
			FROM messages
			WHERE thread_parent_id IN (` + strings.Join(placeholders, ",") + `) AND user_id IS NOT NULL
			GROUP BY thread_parent_id, user_id
		) m
		LEFT JOIN users u ON u.id = m.user_id
		ORDER BY m.thread_parent_id, m.first_reply_id
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := make(map[string][]ThreadParticipant)
	for rows.Next() {
		var parentID, userID, displayName string
		var avatarURL sql.NullString
		if err := rows.Scan(&parentID, &userID, &displayName, &avatarURL); err != nil {
			return nil, err
		}
		// Limit to 3 participants per thread
		if len(participants[parentID]) < 3 {
			p := ThreadParticipant{
				UserID:      userID,
				DisplayName: displayName,
			}
			if avatarURL.Valid {
				p.AvatarURL = &avatarURL.String
			}
			participants[parentID] = append(participants[parentID], p)
		}
	}

	return participants, rows.Err()
}

func (r *Repository) getReactionsForMessages(ctx context.Context, messageIDs []string) (map[string][]Reaction, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(messageIDs))
	args := make([]interface{}, len(messageIDs))
	for i, id := range messageIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT id, message_id, user_id, emoji, created_at
		FROM reactions
		WHERE message_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reactions := make(map[string][]Reaction)
	for rows.Next() {
		var reaction Reaction
		var createdAt string
		err := rows.Scan(&reaction.ID, &reaction.MessageID, &reaction.UserID, &reaction.Emoji, &createdAt)
		if err != nil {
			return nil, err
		}
		reaction.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		reactions[reaction.MessageID] = append(reactions[reaction.MessageID], reaction)
	}

	return reactions, rows.Err()
}

func (r *Repository) scanMessage(row *sql.Row) (*Message, error) {
	var msg Message
	var userID, threadParentID, lastReplyAt, editedAt, deletedAt, systemEventJSON sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&msg.ID, &msg.ChannelID, &userID, &msg.Content, &msg.Type, &systemEventJSON, &threadParentID, &msg.AlsoSendToChannel, &msg.ReplyCount, &lastReplyAt, &editedAt, &deletedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}

	// Default type if empty
	if msg.Type == "" {
		msg.Type = MessageTypeUser
	}

	if userID.Valid {
		msg.UserID = &userID.String
	}
	if systemEventJSON.Valid {
		var eventData SystemEventData
		if err := json.Unmarshal([]byte(systemEventJSON.String), &eventData); err == nil {
			msg.SystemEvent = &eventData
		}
	}
	if threadParentID.Valid {
		msg.ThreadParentID = &threadParentID.String
	}
	if lastReplyAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastReplyAt.String)
		msg.LastReplyAt = &t
	}
	if editedAt.Valid {
		t, _ := time.Parse(time.RFC3339, editedAt.String)
		msg.EditedAt = &t
	}
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		msg.DeletedAt = &t
	}
	msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	msg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &msg, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func (r *Repository) scanMessageWithUser(row rowScanner) (*MessageWithUser, error) {
	var msg MessageWithUser
	var userID, threadParentID, lastReplyAt, editedAt, deletedAt, avatarURL, systemEventJSON sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&msg.ID, &msg.ChannelID, &userID, &msg.Content, &msg.Type, &systemEventJSON, &threadParentID, &msg.AlsoSendToChannel, &msg.ReplyCount, &lastReplyAt, &editedAt, &deletedAt, &createdAt, &updatedAt,
		&msg.UserDisplayName, &avatarURL)
	if err != nil {
		return nil, err
	}

	// Default type if empty
	if msg.Type == "" {
		msg.Type = MessageTypeUser
	}

	if userID.Valid {
		msg.UserID = &userID.String
	}
	if systemEventJSON.Valid {
		var eventData SystemEventData
		if err := json.Unmarshal([]byte(systemEventJSON.String), &eventData); err == nil {
			msg.SystemEvent = &eventData
		}
	}
	if threadParentID.Valid {
		msg.ThreadParentID = &threadParentID.String
	}
	if lastReplyAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastReplyAt.String)
		msg.LastReplyAt = &t
	}
	if editedAt.Valid {
		t, _ := time.Parse(time.RFC3339, editedAt.String)
		msg.EditedAt = &t
	}
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		msg.DeletedAt = &t
	}
	if avatarURL.Valid {
		msg.UserAvatarURL = &avatarURL.String
	}
	msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	msg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &msg, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "duplicate key"))
}

// ListAllUnreads lists all unread messages across channels the user is a member of
func (r *Repository) ListAllUnreads(ctx context.Context, workspaceID, userID string, opts ListOptions) (*UnreadListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}

	var query string
	var args []interface{}

	// Get messages from channels user is a member of that are newer than last_read_message_id
	if opts.Cursor == "" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url,
			       c.name as channel_name, c.type as channel_type
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			JOIN channels c ON c.id = m.channel_id
			JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
			WHERE c.workspace_id = ?
			  AND (m.thread_parent_id IS NULL OR m.also_send_to_channel = TRUE)
			  AND m.deleted_at IS NULL
			  AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
			ORDER BY m.id DESC
			LIMIT ?
		`
		args = []interface{}{userID, workspaceID, opts.Limit + 1}
	} else {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url,
			       c.name as channel_name, c.type as channel_type
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			JOIN channels c ON c.id = m.channel_id
			JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
			WHERE c.workspace_id = ?
			  AND (m.thread_parent_id IS NULL OR m.also_send_to_channel = TRUE)
			  AND m.deleted_at IS NULL
			  AND (cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)
			  AND m.id < ?
			ORDER BY m.id DESC
			LIMIT ?
		`
		args = []interface{}{userID, workspaceID, opts.Cursor, opts.Limit + 1}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []UnreadMessage
	for rows.Next() {
		msg, channelName, channelType, err := r.scanUnreadMessage(rows)
		if err != nil {
			return nil, err
		}
		msg.ChannelName = channelName
		msg.ChannelType = channelType
		messages = append(messages, *msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(messages) > opts.Limit
	if hasMore {
		messages = messages[:opts.Limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].ID
	}

	// Load reactions for all messages
	if len(messages) > 0 {
		messageIDs := make([]string, len(messages))
		for i, m := range messages {
			messageIDs[i] = m.ID
		}
		reactions, err := r.getReactionsForMessages(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		for i := range messages {
			if r, ok := reactions[messages[i].ID]; ok {
				messages[i].Reactions = r
			}
		}
	}

	if messages == nil {
		messages = []UnreadMessage{}
	}

	return &UnreadListResult{
		Messages:   messages,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (r *Repository) scanUnreadMessage(row rowScanner) (*UnreadMessage, string, string, error) {
	var msg UnreadMessage
	var userID, threadParentID, lastReplyAt, editedAt, deletedAt, avatarURL, systemEventJSON sql.NullString
	var createdAt, updatedAt, channelName, channelType string

	err := row.Scan(&msg.ID, &msg.ChannelID, &userID, &msg.Content, &msg.Type, &systemEventJSON, &threadParentID, &msg.AlsoSendToChannel, &msg.ReplyCount, &lastReplyAt, &editedAt, &deletedAt, &createdAt, &updatedAt,
		&msg.UserDisplayName, &avatarURL, &channelName, &channelType)
	if err != nil {
		return nil, "", "", err
	}

	// Default type if empty
	if msg.Type == "" {
		msg.Type = MessageTypeUser
	}

	if userID.Valid {
		msg.UserID = &userID.String
	}
	if systemEventJSON.Valid {
		var eventData SystemEventData
		if err := json.Unmarshal([]byte(systemEventJSON.String), &eventData); err == nil {
			msg.SystemEvent = &eventData
		}
	}
	if threadParentID.Valid {
		msg.ThreadParentID = &threadParentID.String
	}
	if lastReplyAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastReplyAt.String)
		msg.LastReplyAt = &t
	}
	if editedAt.Valid {
		t, _ := time.Parse(time.RFC3339, editedAt.String)
		msg.EditedAt = &t
	}
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		msg.DeletedAt = &t
	}
	if avatarURL.Valid {
		msg.UserAvatarURL = &avatarURL.String
	}
	msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	msg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &msg, channelName, channelType, nil
}

// sanitizeFTSQuery quotes each word in the query to prevent FTS5 syntax injection
func sanitizeFTSQuery(query string) string {
	words := strings.Fields(query)
	quoted := make([]string, 0, len(words))
	for _, w := range words {
		// Remove any double quotes to prevent injection, then quote the word
		w = strings.ReplaceAll(w, "\"", "")
		if w != "" {
			quoted = append(quoted, "\""+w+"\"")
		}
	}
	return strings.Join(quoted, " ")
}

// Search searches messages across channels in a workspace using FTS5
func (r *Repository) Search(ctx context.Context, workspaceID, currentUserID string, opts SearchOptions) (*SearchResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	sanitized := sanitizeFTSQuery(opts.Query)
	if sanitized == "" {
		return &SearchResult{
			Messages: []SearchMessage{},
			Query:    opts.Query,
		}, nil
	}

	// Build WHERE clauses and args for both count and data queries
	whereClauses := []string{
		"m.deleted_at IS NULL",
		"m.type != 'system'",
		"c.workspace_id = ?",
		"messages_fts.content MATCH ?",
		// Access control: user must be a channel member OR channel must be public
		"(cm.user_id IS NOT NULL OR c.type = 'public')",
	}
	baseArgs := []interface{}{workspaceID, sanitized}

	if opts.ChannelID != "" {
		whereClauses = append(whereClauses, "m.channel_id = ?")
		baseArgs = append(baseArgs, opts.ChannelID)
	}
	if opts.UserID != "" {
		whereClauses = append(whereClauses, "m.user_id = ?")
		baseArgs = append(baseArgs, opts.UserID)
	}
	if opts.Before != nil {
		whereClauses = append(whereClauses, "m.created_at < ?")
		baseArgs = append(baseArgs, opts.Before.Format("2006-01-02T15:04:05Z07:00"))
	}
	if opts.After != nil {
		whereClauses = append(whereClauses, "m.created_at > ?")
		baseArgs = append(baseArgs, opts.After.Format("2006-01-02T15:04:05Z07:00"))
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	joinSQL := `
		FROM messages_fts
		JOIN messages m ON m.rowid = messages_fts.rowid
		JOIN channels c ON c.id = m.channel_id
		LEFT JOIN users u ON u.id = m.user_id
		LEFT JOIN channel_memberships cm ON cm.channel_id = c.id AND cm.user_id = ?
	`
	// Prepend currentUserID for the channel_memberships join
	joinArgs := append([]interface{}{currentUserID}, baseArgs...)

	// Count query
	countQuery := "SELECT COUNT(*) " + joinSQL + " WHERE " + whereSQL
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, joinArgs...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Data query
	dataQuery := `
		SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		       COALESCE(u.display_name, '') as user_display_name, u.avatar_url,
		       c.name as channel_name, c.type as channel_type
	` + joinSQL + " WHERE " + whereSQL + `
		ORDER BY messages_fts.rank
		LIMIT ? OFFSET ?
	`
	dataArgs := append(joinArgs, opts.Limit, opts.Offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []SearchMessage
	for rows.Next() {
		msg, channelName, channelType, err := r.scanUnreadMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, SearchMessage{
			MessageWithUser: msg.MessageWithUser,
			ChannelName:     channelName,
			ChannelType:     channelType,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if messages == nil {
		messages = []SearchMessage{}
	}

	return &SearchResult{
		Messages:   messages,
		TotalCount: totalCount,
		HasMore:    opts.Offset+len(messages) < totalCount,
		Query:      opts.Query,
	}, nil
}

// ListUserThreads lists threads the user is subscribed to in a workspace, ordered by last_reply_at DESC
func (r *Repository) ListUserThreads(ctx context.Context, workspaceID, userID string, opts ListOptions) (*ThreadListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	var query string
	var args []interface{}

	// Base query: get parent messages of threads the user is subscribed to
	if opts.Cursor == "" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url,
			       c.name as channel_name, c.type as channel_type,
			       CASE WHEN ts.last_read_reply_id IS NULL THEN 1
			            WHEN EXISTS (SELECT 1 FROM messages r WHERE r.thread_parent_id = m.id AND r.id > ts.last_read_reply_id AND r.deleted_at IS NULL LIMIT 1) THEN 1
			            ELSE 0 END as has_new_replies
			FROM thread_subscriptions ts
			JOIN messages m ON m.id = ts.thread_parent_id
			LEFT JOIN users u ON u.id = m.user_id
			JOIN channels c ON c.id = m.channel_id
			WHERE ts.user_id = ?
			  AND ts.status = 'subscribed'
			  AND c.workspace_id = ?
			  AND m.deleted_at IS NULL
			  AND m.reply_count > 0
			ORDER BY m.last_reply_at DESC, m.id DESC
			LIMIT ?
		`
		args = []interface{}{userID, workspaceID, opts.Limit + 1}
	} else {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.type, m.system_event, m.thread_parent_id, m.also_send_to_channel, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url,
			       c.name as channel_name, c.type as channel_type,
			       CASE WHEN ts.last_read_reply_id IS NULL THEN 1
			            WHEN EXISTS (SELECT 1 FROM messages r WHERE r.thread_parent_id = m.id AND r.id > ts.last_read_reply_id AND r.deleted_at IS NULL LIMIT 1) THEN 1
			            ELSE 0 END as has_new_replies
			FROM thread_subscriptions ts
			JOIN messages m ON m.id = ts.thread_parent_id
			LEFT JOIN users u ON u.id = m.user_id
			JOIN channels c ON c.id = m.channel_id
			WHERE ts.user_id = ?
			  AND ts.status = 'subscribed'
			  AND c.workspace_id = ?
			  AND m.deleted_at IS NULL
			  AND m.reply_count > 0
			  AND (m.last_reply_at < (SELECT last_reply_at FROM messages WHERE id = ?)
			       OR (m.last_reply_at = (SELECT last_reply_at FROM messages WHERE id = ?) AND m.id < ?))
			ORDER BY m.last_reply_at DESC, m.id DESC
			LIMIT ?
		`
		args = []interface{}{userID, workspaceID, opts.Cursor, opts.Cursor, opts.Cursor, opts.Limit + 1}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []ThreadMessage
	for rows.Next() {
		var msg ThreadMessage
		var msgUserID, threadParentID, lastReplyAt, editedAt, deletedAt, avatarURL, systemEventJSON sql.NullString
		var createdAt, updatedAt, channelName, channelType string
		var hasNewReplies int

		err := rows.Scan(&msg.ID, &msg.ChannelID, &msgUserID, &msg.Content, &msg.Type, &systemEventJSON, &threadParentID, &msg.AlsoSendToChannel, &msg.ReplyCount, &lastReplyAt, &editedAt, &deletedAt, &createdAt, &updatedAt,
			&msg.UserDisplayName, &avatarURL, &channelName, &channelType, &hasNewReplies)
		if err != nil {
			return nil, err
		}

		if msg.Type == "" {
			msg.Type = MessageTypeUser
		}
		if msgUserID.Valid {
			msg.UserID = &msgUserID.String
		}
		if systemEventJSON.Valid {
			var eventData SystemEventData
			if err := json.Unmarshal([]byte(systemEventJSON.String), &eventData); err == nil {
				msg.SystemEvent = &eventData
			}
		}
		if threadParentID.Valid {
			msg.ThreadParentID = &threadParentID.String
		}
		if lastReplyAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastReplyAt.String)
			msg.LastReplyAt = &t
		}
		if editedAt.Valid {
			t, _ := time.Parse(time.RFC3339, editedAt.String)
			msg.EditedAt = &t
		}
		if deletedAt.Valid {
			t, _ := time.Parse(time.RFC3339, deletedAt.String)
			msg.DeletedAt = &t
		}
		if avatarURL.Valid {
			msg.UserAvatarURL = &avatarURL.String
		}
		msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		msg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		msg.ChannelName = channelName
		msg.ChannelType = channelType
		msg.HasNewReplies = hasNewReplies == 1

		threads = append(threads, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(threads) > opts.Limit
	if hasMore {
		threads = threads[:opts.Limit]
	}

	var nextCursor string
	if hasMore && len(threads) > 0 {
		nextCursor = threads[len(threads)-1].ID
	}

	// Load reactions, thread participants, and attachments
	if len(threads) > 0 {
		messageIDs := make([]string, len(threads))
		for i, m := range threads {
			messageIDs[i] = m.ID
		}

		reactions, err := r.getReactionsForMessages(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		participants, err := r.getThreadParticipantsForMessages(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		for i := range threads {
			if r, ok := reactions[threads[i].ID]; ok {
				threads[i].Reactions = r
			}
			if p, ok := participants[threads[i].ID]; ok {
				threads[i].ThreadParticipants = p
			}
		}
	}

	if threads == nil {
		threads = []ThreadMessage{}
	}

	return &ThreadListResult{
		Threads:    threads,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}
