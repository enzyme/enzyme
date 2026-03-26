package sse

import (
	"github.com/enzyme/server/internal/openapi"
)

// -- Typed constructors --
// Each constructor enforces that the correct payload type is paired with its
// event type constant. Passing the wrong type is a compile error.

func NewConnectedEvent(data openapi.ConnectedData) Event {
	return Event{Type: EventConnected, Data: data}
}

func NewHeartbeatEvent(data openapi.HeartbeatData) Event {
	return Event{Type: EventHeartbeat, Data: data}
}

func NewMessageNewEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageNew, Data: data}
}

func NewMessageUpdatedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageUpdated, Data: data}
}

func NewMessageDeletedEvent(data openapi.MessageDeletedData) Event {
	return Event{Type: EventMessageDeleted, Data: data}
}

func NewReactionAddedEvent(data openapi.Reaction) Event {
	return Event{Type: EventReactionAdded, Data: data}
}

func NewReactionRemovedEvent(data openapi.ReactionRemovedData) Event {
	return Event{Type: EventReactionRemoved, Data: data}
}

func NewChannelCreatedEvent(data openapi.Channel) Event {
	return Event{Type: EventChannelCreated, Data: data}
}

// NewChannelsInvalidateEvent sends a channels.invalidate signal.
// Used when multiple channels are created in a batch (e.g., auto-DMs on workspace join)
// and the frontend should refetch its channel list.
func NewChannelsInvalidateEvent() Event {
	return Event{Type: EventChannelsInvalidate, Data: struct{}{}}
}

func NewChannelUpdatedEvent(data openapi.Channel) Event {
	return Event{Type: EventChannelUpdated, Data: data}
}

func NewChannelArchivedEvent(data openapi.Channel) Event {
	return Event{Type: EventChannelArchived, Data: data}
}

func NewChannelMemberAddedEvent(data openapi.ChannelMemberData) Event {
	return Event{Type: EventMemberAdded, Data: data}
}

func NewChannelMemberRemovedEvent(data openapi.ChannelMemberData) Event {
	return Event{Type: EventMemberRemoved, Data: data}
}

func NewChannelReadEvent(data openapi.ChannelReadEventData) Event {
	return Event{Type: EventChannelRead, Data: data}
}

func NewTypingStartEvent(data openapi.TypingEventData) Event {
	return Event{Type: EventTypingStart, Data: data}
}

func NewTypingStopEvent(data openapi.TypingEventData) Event {
	return Event{Type: EventTypingStop, Data: data}
}

func NewPresenceChangedEvent(data openapi.PresenceData) Event {
	return Event{Type: EventPresenceChanged, Data: data}
}

func NewPresenceInitialEvent(data openapi.PresenceInitialData) Event {
	return Event{Type: EventPresenceInitial, Data: data}
}

func NewNotificationEvent(data openapi.NotificationData) Event {
	return Event{Type: EventNotification, Data: data}
}

func NewEmojiCreatedEvent(data openapi.CustomEmoji) Event {
	return Event{Type: EventEmojiCreated, Data: data}
}

func NewEmojiDeletedEvent(data openapi.EmojiDeletedData) Event {
	return Event{Type: EventEmojiDeleted, Data: data}
}

func NewMessagePinnedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessagePinned, Data: data}
}

func NewMessageUnpinnedEvent(data openapi.MessageWithUser) Event {
	return Event{Type: EventMessageUnpinned, Data: data}
}

func NewMemberBannedEvent(data openapi.WorkspaceMemberData) Event {
	return Event{Type: EventMemberBanned, Data: data}
}

func NewMemberUnbannedEvent(data openapi.WorkspaceMemberData) Event {
	return Event{Type: EventMemberUnbanned, Data: data}
}

func NewMemberLeftEvent(data openapi.WorkspaceMemberData) Event {
	return Event{Type: EventMemberLeft, Data: data}
}

func NewMemberRoleChangedEvent(data openapi.MemberRoleChangedData) Event {
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

func NewScheduledMessageDeletedEvent(data openapi.ScheduledMessageDeletedData) Event {
	return Event{Type: EventScheduledMessageDeleted, Data: data}
}

func NewScheduledMessageSentEvent(data openapi.ScheduledMessageSentData) Event {
	return Event{Type: EventScheduledMessageSent, Data: data}
}

func NewScheduledMessageFailedEvent(data openapi.ScheduledMessageFailedData) Event {
	return Event{Type: EventScheduledMessageFailed, Data: data}
}
