package linkpreview

import "time"

const (
	// CacheTTL is how long successful fetches are cached.
	CacheTTL = 24 * time.Hour
	// ErrorCacheTTL is how long failed fetches are cached.
	ErrorCacheTTL = 1 * time.Hour
)

// Preview types.
const (
	PreviewTypeExternal = "external"
	PreviewTypeMessage  = "message"
)

// CacheEntry is a URL-level cache row shared across messages.
type CacheEntry struct {
	URL         string
	Title       string
	Description string
	ImageURL    string
	SiteName    string
	FetchedAt   time.Time
	ExpiresAt   time.Time
	FetchError  string
}

// Preview is a per-message link preview row.
type Preview struct {
	ID          string
	MessageID   string
	URL         string
	Type        string // "external" or "message"
	Title       string
	Description string
	ImageURL    string
	SiteName    string
	CreatedAt   time.Time

	// Internal message preview fields (Type == "message")
	LinkedMessageID        string
	LinkedChannelID        string
	LinkedChannelName      string
	LinkedChannelType      string
	MessageAuthorID        string
	MessageAuthorName      string
	MessageAuthorAvatarURL string
	MessageAuthorGravatar  string
	MessageContent         string
	MessageCreatedAt       string
}
