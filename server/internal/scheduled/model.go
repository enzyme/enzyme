package scheduled

import (
	"errors"
	"time"
)

var ErrScheduledMessageNotFound = errors.New("scheduled message not found")

const (
	StatusPending = "pending"
	StatusSending = "sending"
	StatusFailed  = "failed"
	MaxRetries    = 3
)

// PermanentError wraps errors that should not be retried.
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

type ScheduledMessage struct {
	ID                string    `json:"id"`
	ChannelID         string    `json:"channel_id"`
	UserID            string    `json:"user_id"`
	Content           string    `json:"content"`
	ThreadParentID    *string   `json:"thread_parent_id,omitempty"`
	AlsoSendToChannel bool      `json:"also_send_to_channel"`
	AttachmentIDs     []string  `json:"attachment_ids"`
	ScheduledFor      time.Time `json:"scheduled_for"`
	Status            string    `json:"status"`
	RetryCount        int       `json:"retry_count"`
	LastError         string    `json:"last_error,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ScheduledMessageWithChannel struct {
	ScheduledMessage
	ChannelName string `json:"channel_name"`
	WorkspaceID string `json:"workspace_id"`
}
