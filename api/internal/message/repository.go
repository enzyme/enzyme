package message

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ErrMessageNotFound   = errors.New("message not found")
	ErrReactionExists    = errors.New("reaction already exists")
	ErrReactionNotFound  = errors.New("reaction not found")
	ErrCannotEditMessage = errors.New("cannot edit this message")
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

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (id, channel_id, user_id, content, thread_parent_id, reply_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`, msg.ID, msg.ChannelID, msg.UserID, msg.Content, msg.ThreadParentID, now.Format(time.RFC3339), now.Format(time.RFC3339))
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

func (r *Repository) GetByID(ctx context.Context, id string) (*Message, error) {
	return r.scanMessage(r.db.QueryRowContext(ctx, `
		SELECT id, channel_id, user_id, content, thread_parent_id, reply_count, last_reply_at, edited_at, deleted_at, created_at, updated_at
		FROM messages WHERE id = ?
	`, id))
}

func (r *Repository) GetByIDWithUser(ctx context.Context, id string) (*MessageWithUser, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT m.id, m.channel_id, m.user_id, m.content, m.thread_parent_id, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
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
	result, err := r.db.ExecContext(ctx, `
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
	return nil
}

func (r *Repository) List(ctx context.Context, channelID string, opts ListOptions) (*ListResult, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 50
	}

	var query string
	var args []interface{}

	// Only get top-level messages (not thread replies)
	if opts.Cursor == "" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.thread_parent_id, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = ? AND m.thread_parent_id IS NULL
			ORDER BY m.id DESC
			LIMIT ?
		`
		args = []interface{}{channelID, opts.Limit + 1}
	} else if opts.Direction == "after" {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.thread_parent_id, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = ? AND m.thread_parent_id IS NULL AND m.id > ?
			ORDER BY m.id ASC
			LIMIT ?
		`
		args = []interface{}{channelID, opts.Cursor, opts.Limit + 1}
	} else {
		query = `
			SELECT m.id, m.channel_id, m.user_id, m.content, m.thread_parent_id, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
			       COALESCE(u.display_name, '') as user_display_name, u.avatar_url
			FROM messages m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.channel_id = ? AND m.thread_parent_id IS NULL AND m.id < ?
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
			SELECT m.id, m.channel_id, m.user_id, m.content, m.thread_parent_id, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
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
			SELECT m.id, m.channel_id, m.user_id, m.content, m.thread_parent_id, m.reply_count, m.last_reply_at, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
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
	var userID, threadParentID, lastReplyAt, editedAt, deletedAt sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&msg.ID, &msg.ChannelID, &userID, &msg.Content, &threadParentID, &msg.ReplyCount, &lastReplyAt, &editedAt, &deletedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrMessageNotFound
	}
	if err != nil {
		return nil, err
	}

	if userID.Valid {
		msg.UserID = &userID.String
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
	var userID, threadParentID, lastReplyAt, editedAt, deletedAt, avatarURL sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&msg.ID, &msg.ChannelID, &userID, &msg.Content, &threadParentID, &msg.ReplyCount, &lastReplyAt, &editedAt, &deletedAt, &createdAt, &updatedAt,
		&msg.UserDisplayName, &avatarURL)
	if err != nil {
		return nil, err
	}

	if userID.Valid {
		msg.UserID = &userID.String
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
