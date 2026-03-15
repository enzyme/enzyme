package sse

import (
	"github.com/enzyme/api/internal/openapi"
)

// -- Data types for events with inline schemas --

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
	UserID        string  `json:"user_id"`
	WorkspaceID   string  `json:"workspace_id"`
	BannedBy      *string `json:"banned_by,omitempty"`
	Reason        *string `json:"reason,omitempty"`
	ExpiresAt     *string `json:"expires_at,omitempty"`
	WorkspaceName *string `json:"workspace_name,omitempty"`
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

// ConnectedData is the data payload for connected events.
type ConnectedData struct {
	ClientID string `json:"client_id"`
}

// HeartbeatData is the data payload for heartbeat events.
type HeartbeatData struct {
	Timestamp int64 `json:"timestamp"`
}

// -- Constructors --

// NewConnectedEvent creates a connected event.
func NewConnectedEvent(data ConnectedData) Event {
	return Event{Type: EventConnected, Data: data}
}

// NewHeartbeatEvent creates a heartbeat event.
func NewHeartbeatEvent(data HeartbeatData) Event {
	return Event{Type: EventHeartbeat, Data: data}
}

// NewMessageNewEvent creates a message.new event.
func NewMessageNewEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageNew, Data: data}
}

// NewMessageUpdatedEvent creates a message.updated event.
func NewMessageUpdatedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageUpdated, Data: data}
}

// NewMessageDeletedEvent creates a message.deleted event.
func NewMessageDeletedEvent(data MessageDeletedData) Event {
	return Event{Type: EventMessageDeleted, Data: data}
}

// NewReactionAddedEvent creates a reaction.added event.
func NewReactionAddedEvent(data openapi.Reaction) Event {
	return Event{Type: EventReactionAdded, Data: data}
}

// NewReactionRemovedEvent creates a reaction.removed event.
func NewReactionRemovedEvent(data ReactionRemovedData) Event {
	return Event{Type: EventReactionRemoved, Data: data}
}

// NewChannelUpdatedEvent creates a channel.updated event.
func NewChannelUpdatedEvent(data openapi.Channel) Event {
	return Event{Type: EventChannelUpdated, Data: data}
}

// NewChannelMemberAddedEvent creates a channel.member_added event.
func NewChannelMemberAddedEvent(data ChannelMemberData) Event {
	return Event{Type: EventMemberAdded, Data: data}
}

// NewChannelMemberRemovedEvent creates a channel.member_removed event.
func NewChannelMemberRemovedEvent(data ChannelMemberData) Event {
	return Event{Type: EventMemberRemoved, Data: data}
}

// NewChannelReadEvent creates a channel.read event.
func NewChannelReadEvent(data ChannelReadEventData) Event {
	return Event{Type: EventChannelRead, Data: data}
}

// NewTypingStartEvent creates a typing.start event.
func NewTypingStartEvent(data TypingEventData) Event {
	return Event{Type: EventTypingStart, Data: data}
}

// NewTypingStopEvent creates a typing.stop event.
func NewTypingStopEvent(data TypingEventData) Event {
	return Event{Type: EventTypingStop, Data: data}
}

// NewPresenceChangedEvent creates a presence.changed event.
func NewPresenceChangedEvent(data PresenceData) Event {
	return Event{Type: EventPresenceChanged, Data: data}
}

// NewPresenceInitialEvent creates a presence.initial event.
func NewPresenceInitialEvent(data PresenceInitialData) Event {
	return Event{Type: EventPresenceInitial, Data: data}
}

// NewNotificationEvent creates a notification event.
func NewNotificationEvent(data NotificationData) Event {
	return Event{Type: EventNotification, Data: data}
}

// NewEmojiCreatedEvent creates an emoji.created event.
func NewEmojiCreatedEvent(data openapi.CustomEmoji) Event {
	return Event{Type: EventEmojiCreated, Data: data}
}

// NewEmojiDeletedEvent creates an emoji.deleted event.
func NewEmojiDeletedEvent(data EmojiDeletedData) Event {
	return Event{Type: EventEmojiDeleted, Data: data}
}

// NewMessagePinnedEvent creates a message.pinned event.
func NewMessagePinnedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessagePinned, Data: data}
}

// NewMessageUnpinnedEvent creates a message.unpinned event.
func NewMessageUnpinnedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageUnpinned, Data: data}
}

// NewMemberBannedEvent creates a member.banned event.
func NewMemberBannedEvent(data MemberBannedData) Event {
	return Event{Type: EventMemberBanned, Data: data}
}

// NewMemberUnbannedEvent creates a member.unbanned event.
func NewMemberUnbannedEvent(data MemberUnbannedData) Event {
	return Event{Type: EventMemberUnbanned, Data: data}
}

// NewMemberLeftEvent creates a member.left event.
func NewMemberLeftEvent(data MemberLeftData) Event {
	return Event{Type: EventMemberLeft, Data: data}
}

// NewMemberRoleChangedEvent creates a member.role_changed event.
func NewMemberRoleChangedEvent(data MemberRoleChangedData) Event {
	return Event{Type: EventMemberRoleChanged, Data: data}
}

// NewWorkspaceUpdatedEvent creates a workspace.updated event.
func NewWorkspaceUpdatedEvent(data openapi.Workspace) Event {
	return Event{Type: EventWorkspaceUpdated, Data: data}
}

// NewScheduledMessageCreatedEvent creates a scheduled_message.created event.
func NewScheduledMessageCreatedEvent(data openapi.ScheduledMessage) Event {
	return Event{Type: EventScheduledMessageCreated, Data: data}
}

// NewScheduledMessageUpdatedEvent creates a scheduled_message.updated event.
func NewScheduledMessageUpdatedEvent(data openapi.ScheduledMessage) Event {
	return Event{Type: EventScheduledMessageUpdated, Data: data}
}

// NewScheduledMessageDeletedEvent creates a scheduled_message.deleted event.
func NewScheduledMessageDeletedEvent(data ScheduledMessageDeletedData) Event {
	return Event{Type: EventScheduledMessageDeleted, Data: data}
}

// NewScheduledMessageSentEvent creates a scheduled_message.sent event.
func NewScheduledMessageSentEvent(data ScheduledMessageSentData) Event {
	return Event{Type: EventScheduledMessageSent, Data: data}
}

// NewScheduledMessageFailedEvent creates a scheduled_message.failed event.
func NewScheduledMessageFailedEvent(data ScheduledMessageFailedData) Event {
	return Event{Type: EventScheduledMessageFailed, Data: data}
}
