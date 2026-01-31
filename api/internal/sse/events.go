package sse

const (
	EventConnected       = "connected"
	EventHeartbeat       = "heartbeat"
	EventMessageNew      = "message.new"
	EventMessageUpdated  = "message.updated"
	EventMessageDeleted  = "message.deleted"
	EventReactionAdded   = "reaction.added"
	EventReactionRemoved = "reaction.removed"
	EventChannelCreated  = "channel.created"
	EventChannelUpdated  = "channel.updated"
	EventChannelArchived = "channel.archived"
	EventMemberAdded     = "channel.member_added"
	EventMemberRemoved   = "channel.member_removed"
	EventChannelRead     = "channel.read"
	EventTypingStart     = "typing.start"
	EventTypingStop      = "typing.stop"
	EventPresenceChanged = "presence.changed"
)

type Event struct {
	ID   string      `json:"id,omitempty"`
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}
