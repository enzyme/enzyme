package linkpreview

import (
	"net/url"
	"strings"
)

// InternalMessageRef holds parsed data from an internal message URL.
type InternalMessageRef struct {
	WorkspaceID string
	ChannelID   string
	MessageID   string
}

// IsInternalURL returns true if the URL looks like an internal app URL
// (contains /workspaces/ in its path). Used to avoid fetching our own pages
// as external OG previews.
func IsInternalURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.Contains(u.Path, "/workspaces/")
}

// ParseInternalMessageURL parses a URL matching /workspaces/{id}/channels/{id}?msg={id}
// (or ?thread={id}). Returns nil if the URL doesn't match.
// Host-agnostic: only checks path and query params.
func ParseInternalMessageURL(rawURL string) *InternalMessageRef {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	// Normalize: strip trailing slash
	path := strings.TrimSuffix(u.Path, "/")

	// Split path into segments, filtering empty ones
	segments := strings.Split(path, "/")
	var parts []string
	for _, s := range segments {
		if s != "" {
			parts = append(parts, s)
		}
	}

	// Expect: workspaces/{id}/channels/{id}
	if len(parts) != 4 {
		return nil
	}
	if parts[0] != "workspaces" || parts[2] != "channels" {
		return nil
	}

	workspaceID := parts[1]
	channelID := parts[3]

	if workspaceID == "" || channelID == "" {
		return nil
	}

	// Check for msg= or thread= query param
	msgID := u.Query().Get("msg")
	if msgID == "" {
		msgID = u.Query().Get("thread")
	}
	if msgID == "" {
		return nil
	}

	return &InternalMessageRef{
		WorkspaceID: workspaceID,
		ChannelID:   channelID,
		MessageID:   msgID,
	}
}
