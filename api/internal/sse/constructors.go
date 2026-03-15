package sse

import (
	"time"

	"github.com/enzyme/api/internal/openapi"
)

// -- Data types for events with inline/simple schemas --

// PresenceStatus defines the online/offline status of a user.
type PresenceStatus string

const (
	Online  PresenceStatus = "online"
	Offline PresenceStatus = "offline"
)

// PresenceData is the data payload for presence.changed events.
type PresenceData struct {
	Status PresenceStatus `json:"status"`
	UserId string         `json:"user_id"`
}

// PresenceInitialData is the data payload for presence.initial events.
type PresenceInitialData struct {
	OnlineUserIds []string `json:"online_user_ids"`
}

// TypingEventData is the data payload for typing.start and typing.stop events.
type TypingEventData struct {
	ChannelId       string  `json:"channel_id"`
	UserDisplayName *string `json:"user_display_name,omitempty"`
	UserId          string  `json:"user_id"`
}

// ChannelReadEventData is the data payload for channel.read events.
type ChannelReadEventData struct {
	ChannelId         string `json:"channel_id"`
	LastReadMessageId string `json:"last_read_message_id"`
}

// NotificationDataType defines the type of notification.
type NotificationDataType string

// NotificationData is the data payload for notification events.
type NotificationData struct {
	ChannelId      string               `json:"channel_id"`
	ChannelName    *string              `json:"channel_name,omitempty"`
	MessageId      string               `json:"message_id"`
	Preview        *string              `json:"preview,omitempty"`
	SenderName     *string              `json:"sender_name,omitempty"`
	ThreadParentId *string              `json:"thread_parent_id,omitempty"`
	Type           NotificationDataType `json:"type"`
}

// ConnectedData is the data payload for connected events.
type ConnectedData struct {
	ClientID string `json:"client_id"`
}

// HeartbeatData is the data payload for heartbeat events.
type HeartbeatData struct {
	Timestamp int64 `json:"timestamp"`
}

// MessageDeletedData is the data payload for message.deleted events.
type MessageDeletedData struct {
	ID             string  `json:"id"`
	ThreadParentID *string `json:"thread_parent_id,omitempty"`
}

// ReactionRemovedData is the data payload for reaction.removed events.
type ReactionRemovedData struct {
	MessageID string `json:"message_id"`
	UserID    string `json:"user_id"`
	Emoji     string `json:"emoji"`
}

// ChannelMemberData is the data payload for channel.member_added and channel.member_removed events.
type ChannelMemberData struct {
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
}

// MemberBannedData is the data payload for member.banned events.
type MemberBannedData struct {
	UserID        string     `json:"user_id"`
	WorkspaceID   string     `json:"workspace_id"`
	BannedBy      *string    `json:"banned_by,omitempty"`
	Reason        *string    `json:"reason,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	WorkspaceName *string    `json:"workspace_name,omitempty"`
}

// MemberUnbannedData is the data payload for member.unbanned events.
type MemberUnbannedData struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
}

// MemberLeftData is the data payload for member.left events.
type MemberLeftData struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
}

// MemberRoleChangedData is the data payload for member.role_changed events.
type MemberRoleChangedData struct {
	UserID  string `json:"user_id"`
	OldRole string `json:"old_role"`
	NewRole string `json:"new_role"`
}

// EmojiDeletedData is the data payload for emoji.deleted events.
type EmojiDeletedData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ScheduledMessageDeletedData is the data payload for scheduled_message.deleted events.
type ScheduledMessageDeletedData struct {
	ID string `json:"id"`
}

// ScheduledMessageSentData is the data payload for scheduled_message.sent events.
type ScheduledMessageSentData struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
}

// ScheduledMessageFailedData is the data payload for scheduled_message.failed events.
type ScheduledMessageFailedData struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Error     string `json:"error"`
}

// -- Typed constructors --
// Each constructor enforces that the correct payload type is paired with its
// event type constant. Passing the wrong type is a compile error.

func NewConnectedEvent(data ConnectedData) Event {
	return Event{Type: EventConnected, Data: data}
}

func NewHeartbeatEvent(data HeartbeatData) Event {
	return Event{Type: EventHeartbeat, Data: data}
}

func NewMessageNewEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageNew, Data: data}
}

func NewMessageUpdatedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageUpdated, Data: data}
}

func NewMessageDeletedEvent(data MessageDeletedData) Event {
	return Event{Type: EventMessageDeleted, Data: data}
}

func NewReactionAddedEvent(data openapi.Reaction) Event {
	return Event{Type: EventReactionAdded, Data: data}
}

func NewReactionRemovedEvent(data ReactionRemovedData) Event {
	return Event{Type: EventReactionRemoved, Data: data}
}

func NewChannelUpdatedEvent(data openapi.Channel) Event {
	return Event{Type: EventChannelUpdated, Data: data}
}

func NewChannelMemberAddedEvent(data ChannelMemberData) Event {
	return Event{Type: EventMemberAdded, Data: data}
}

func NewChannelMemberRemovedEvent(data ChannelMemberData) Event {
	return Event{Type: EventMemberRemoved, Data: data}
}

func NewChannelReadEvent(data ChannelReadEventData) Event {
	return Event{Type: EventChannelRead, Data: data}
}

func NewTypingStartEvent(data TypingEventData) Event {
	return Event{Type: EventTypingStart, Data: data}
}

func NewTypingStopEvent(data TypingEventData) Event {
	return Event{Type: EventTypingStop, Data: data}
}

func NewPresenceChangedEvent(data PresenceData) Event {
	return Event{Type: EventPresenceChanged, Data: data}
}

func NewPresenceInitialEvent(data PresenceInitialData) Event {
	return Event{Type: EventPresenceInitial, Data: data}
}

func NewNotificationEvent(data NotificationData) Event {
	return Event{Type: EventNotification, Data: data}
}

func NewEmojiCreatedEvent(data openapi.CustomEmoji) Event {
	return Event{Type: EventEmojiCreated, Data: data}
}

func NewEmojiDeletedEvent(data EmojiDeletedData) Event {
	return Event{Type: EventEmojiDeleted, Data: data}
}

func NewMessagePinnedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessagePinned, Data: data}
}

func NewMessageUnpinnedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageUnpinned, Data: data}
}

func NewMemberBannedEvent(data MemberBannedData) Event {
	return Event{Type: EventMemberBanned, Data: data}
}

func NewMemberUnbannedEvent(data MemberUnbannedData) Event {
	return Event{Type: EventMemberUnbanned, Data: data}
}

func NewMemberLeftEvent(data MemberLeftData) Event {
	return Event{Type: EventMemberLeft, Data: data}
}

func NewMemberRoleChangedEvent(data MemberRoleChangedData) Event {
	return Event{Type: EventMemberRoleChanged, Data: data}
}

func NewWorkspaceUpdatedEvent(data openapi.Workspace) Event {
	return Event{Type: EventWorkspaceUpdated, Data: data}
}

func NewScheduledMessageCreatedEvent(data openapi.ScheduledMessage) Event {
	return Event{Type: EventScheduledMessageCreated, Data: data}
}

func NewScheduledMessageUpdatedEvent(data openapi.ScheduledMessage) Event {
	return Event{Type: EventScheduledMessageUpdated, Data: data}
}

func NewScheduledMessageDeletedEvent(data ScheduledMessageDeletedData) Event {
	return Event{Type: EventScheduledMessageDeleted, Data: data}
}

func NewScheduledMessageSentEvent(data ScheduledMessageSentData) Event {
	return Event{Type: EventScheduledMessageSent, Data: data}
}

func NewScheduledMessageFailedEvent(data ScheduledMessageFailedData) Event {
	return Event{Type: EventScheduledMessageFailed, Data: data}
}
