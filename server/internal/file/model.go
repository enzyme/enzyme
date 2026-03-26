package file

import (
	"time"
)

type Attachment struct {
	ID          string    `json:"id"`
	MessageID   *string   `json:"message_id,omitempty"`
	ChannelID   string    `json:"channel_id"`
	UserID      *string   `json:"user_id,omitempty"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StoragePath string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
