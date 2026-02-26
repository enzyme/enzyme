package message

import (
	"time"

	"github.com/enzyme/api/internal/file"
	"github.com/enzyme/api/internal/linkpreview"
)

// Message types
const (
	MessageTypeUser   = "user"
	MessageTypeSystem = "system"
)

// System event types
const (
	SystemEventUserJoined                = "user_joined"
	SystemEventUserLeft                  = "user_left"
	SystemEventUserAdded                 = "user_added"
	SystemEventChannelConverted          = "user_converted_channel"
	SystemEventChannelRenamed            = "channel_renamed"
	SystemEventChannelVisibilityChanged  = "channel_visibility_changed"
	SystemEventChannelDescriptionUpdated = "channel_description_updated"
	SystemEventMessagePinned             = "message_pinned"
	SystemEventMessageUnpinned           = "message_unpinned"
)

// SystemEventData contains metadata for system messages
type SystemEventData struct {
	EventType        string  `json:"event_type"`
	UserID           string  `json:"user_id"`
	UserDisplayName  string  `json:"user_display_name"`
	ChannelName      string  `json:"channel_name"`
	ActorID          *string `json:"actor_id,omitempty"`
	ActorDisplayName *string `json:"actor_display_name,omitempty"`
	OldChannelName   *string `json:"old_channel_name,omitempty"`
	ChannelType      *string `json:"channel_type,omitempty"`
	MessageID        *string `json:"message_id,omitempty"`
}

type Message struct {
	ID                string           `json:"id"`
	ChannelID         string           `json:"channel_id"`
	UserID            *string          `json:"user_id,omitempty"`
	Content           string           `json:"content"`
	Type              string           `json:"type"`
	SystemEvent       *SystemEventData `json:"system_event,omitempty"`
	Mentions          []string         `json:"mentions,omitempty"`
	ThreadParentID    *string          `json:"thread_parent_id,omitempty"`
	AlsoSendToChannel bool             `json:"also_send_to_channel"`
	ReplyCount        int              `json:"reply_count"`
	LastReplyAt       *time.Time       `json:"last_reply_at,omitempty"`
	EditedAt          *time.Time       `json:"edited_at,omitempty"`
	DeletedAt         *time.Time       `json:"deleted_at,omitempty"`
	PinnedAt          *time.Time       `json:"pinned_at,omitempty"`
	PinnedBy          *string          `json:"pinned_by,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

type MessageWithUser struct {
	Message
	UserDisplayName    string               `json:"user_display_name,omitempty"`
	UserAvatarURL      *string              `json:"user_avatar_url,omitempty"`
	UserEmail          string               `json:"-"`
	Reactions          []Reaction           `json:"reactions,omitempty"`
	ThreadParticipants []ThreadParticipant  `json:"thread_participants,omitempty"`
	Attachments        []file.Attachment    `json:"attachments,omitempty"`
	LinkPreview        *linkpreview.Preview `json:"link_preview,omitempty"`
}

type ThreadParticipant struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Email       string  `json:"-"`
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
	Direction string // "before", "after", or "around"
}

// FilterOptions carries context for ban-hide and block filtering in message queries.
// When non-nil, messages from banned users (with hide_messages=1) and blocked users
// are excluded from results. Reactions and thread participants from those users are
// also filtered.
type FilterOptions struct {
	WorkspaceID      string // Required for ban-hide (workspace_bans) and block (user_blocks) filters
	RequestingUserID string // Required for block filter (blocker_id)
}

type ListResult struct {
	Messages   []MessageWithUser `json:"messages"`
	HasMore    bool              `json:"has_more"`
	HasNewer   bool              `json:"has_newer,omitempty"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type UnreadMessage struct {
	MessageWithUser
	ChannelName string `json:"channel_name"`
	ChannelType string `json:"channel_type"`
}

type UnreadListResult struct {
	Messages   []UnreadMessage `json:"messages"`
	HasMore    bool            `json:"has_more"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type ThreadMessage struct {
	MessageWithUser
	ChannelName   string `json:"channel_name"`
	ChannelType   string `json:"channel_type"`
	HasNewReplies bool   `json:"has_new_replies"`
}

type ThreadListResult struct {
	Threads           []ThreadMessage `json:"threads"`
	HasMore           bool            `json:"has_more"`
	NextCursor        string          `json:"next_cursor,omitempty"`
	UnreadThreadCount int             `json:"unread_thread_count"`
}

type SearchOptions struct {
	Query     string
	ChannelID string
	UserID    string
	Before    *time.Time
	After     *time.Time
	Limit     int
	Offset    int
}

type SearchMessage struct {
	MessageWithUser
	ChannelName string `json:"channel_name"`
	ChannelType string `json:"channel_type"`
}

type SearchResult struct {
	Messages   []SearchMessage `json:"messages"`
	TotalCount int             `json:"total_count"`
	HasMore    bool            `json:"has_more"`
	Query      string          `json:"query"`
}
