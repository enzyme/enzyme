package emoji

import (
	"time"
)

type CustomEmoji struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	CreatedBy   string    `json:"created_by"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StoragePath string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
