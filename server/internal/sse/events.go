package sse

import (
	"encoding/json"

	"github.com/enzyme/server/internal/openapi"
	"github.com/oklog/ulid/v2"
)

// Event type constants derived from the generated OpenAPI enum.
// Using string() on the generated constants ensures compile-time linkage:
// if the spec changes, the generated type changes, and these still track it.
const (
	EventConnected       = string(openapi.SSEEventTypeConnected)
	EventHeartbeat       = string(openapi.SSEEventTypeHeartbeat)
	EventMessageNew      = string(openapi.SSEEventTypeMessageNew)
	EventMessageUpdated  = string(openapi.SSEEventTypeMessageUpdated)
	EventMessageDeleted  = string(openapi.SSEEventTypeMessageDeleted)
	EventReactionAdded   = string(openapi.SSEEventTypeReactionAdded)
	EventReactionRemoved = string(openapi.SSEEventTypeReactionRemoved)
	EventChannelCreated  = string(openapi.SSEEventTypeChannelCreated)
	EventChannelUpdated  = string(openapi.SSEEventTypeChannelUpdated)
	EventChannelArchived = string(openapi.SSEEventTypeChannelArchived)
	EventMemberAdded     = string(openapi.SSEEventTypeChannelMemberAdded)
	EventMemberRemoved   = string(openapi.SSEEventTypeChannelMemberRemoved)
	EventChannelRead     = string(openapi.SSEEventTypeChannelRead)
	EventTypingStart     = string(openapi.SSEEventTypeTypingStart)
	EventTypingStop      = string(openapi.SSEEventTypeTypingStop)
	EventPresenceChanged = string(openapi.SSEEventTypePresenceChanged)
	EventPresenceInitial = string(openapi.SSEEventTypePresenceInitial)
	EventNotification    = string(openapi.SSEEventTypeNotification)
	EventEmojiCreated    = string(openapi.SSEEventTypeEmojiCreated)
	EventEmojiDeleted    = string(openapi.SSEEventTypeEmojiDeleted)

	EventMessagePinned     = string(openapi.SSEEventTypeMessagePinned)
	EventMessageUnpinned   = string(openapi.SSEEventTypeMessageUnpinned)
	EventMemberBanned      = string(openapi.SSEEventTypeMemberBanned)
	EventMemberUnbanned    = string(openapi.SSEEventTypeMemberUnbanned)
	EventMemberLeft        = string(openapi.SSEEventTypeMemberLeft)
	EventMemberRoleChanged = string(openapi.SSEEventTypeMemberRoleChanged)

	EventWorkspaceUpdated   = string(openapi.SSEEventTypeWorkspaceUpdated)
	EventChannelsInvalidate = string(openapi.SSEEventTypeChannelsInvalidate)

	EventScheduledMessageCreated = string(openapi.SSEEventTypeScheduledMessageCreated)
	EventScheduledMessageUpdated = string(openapi.SSEEventTypeScheduledMessageUpdated)
	EventScheduledMessageDeleted = string(openapi.SSEEventTypeScheduledMessageDeleted)
	EventScheduledMessageSent    = string(openapi.SSEEventTypeScheduledMessageSent)
	EventScheduledMessageFailed  = string(openapi.SSEEventTypeScheduledMessageFailed)
)

type Event struct {
	ID   string      `json:"id,omitempty"`
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// SerializedEvent is a pre-marshaled SSE event ready for writing to clients.
// The JSON is marshaled once in the broadcast path rather than per-subscriber.
type SerializedEvent struct {
	ID   string
	Data []byte // pre-marshaled JSON of the full Event
}

// Serialize marshals an Event into a SerializedEvent.
// The event ID is assigned if empty.
func (e Event) Serialize() (SerializedEvent, error) {
	if e.ID == "" {
		e.ID = ulid.Make().String()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return SerializedEvent{}, err
	}
	return SerializedEvent{ID: e.ID, Data: data}, nil
}
