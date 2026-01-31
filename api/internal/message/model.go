package message

import (
	"time"
)

type Message struct {
	ID             string     `json:"id"`
	ChannelID      string     `json:"channel_id"`
	UserID         *string    `json:"user_id,omitempty"`
	Content        string     `json:"content"`
	ThreadParentID *string    `json:"thread_parent_id,omitempty"`
	ReplyCount     int        `json:"reply_count"`
	LastReplyAt    *time.Time `json:"last_reply_at,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type MessageWithUser struct {
	Message
	UserDisplayName    string              `json:"user_display_name,omitempty"`
	UserAvatarURL      *string             `json:"user_avatar_url,omitempty"`
	Reactions          []Reaction          `json:"reactions,omitempty"`
	ThreadParticipants []ThreadParticipant `json:"thread_participants,omitempty"`
}

type ThreadParticipant struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type Reaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type ReactionSummary struct {
	Emoji   string   `json:"emoji"`
	Count   int      `json:"count"`
	UserIDs []string `json:"user_ids"`
}

type ListOptions struct {
	Cursor    string
	Limit     int
	Direction string // "before" or "after"
}

type ListResult struct {
	Messages   []MessageWithUser `json:"messages"`
	HasMore    bool              `json:"has_more"`
	NextCursor string            `json:"next_cursor,omitempty"`
}
