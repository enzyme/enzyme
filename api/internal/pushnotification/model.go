package pushnotification

import "time"

// DeviceToken represents a registered push notification device token.
type DeviceToken struct {
	ID        string
	UserID    string
	Token     string
	Platform  string // "fcm" or "apns"
	DeviceID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MaxTokensPerUser is the maximum number of device tokens a single user can register.
// When the limit is reached, the least-recently-updated token is evicted.
const MaxTokensPerUser = 10

// NotificationData contains the data needed to send a push notification.
type NotificationData struct {
	Title       string
	Body        string
	ChannelID   string
	MessageID   string
	WorkspaceID string
	ChannelName string
	ServerURL   string
}

// RelayRequest is the payload sent to the push relay service.
type RelayRequest struct {
	DeviceToken string           `json:"device_token"`
	Platform    string           `json:"platform"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	Data        RelayRequestData `json:"data"`
}

// RelayRequestData contains deep-linking metadata for the push notification.
type RelayRequestData struct {
	ChannelID   string `json:"channel_id"`
	MessageID   string `json:"message_id"`
	WorkspaceID string `json:"workspace_id"`
	ChannelName string `json:"channel_name"`
	ServerURL   string `json:"server_url"`
}

// RelayResponse is the response from the push relay service.
type RelayResponse struct {
	Status string `json:"status"` // "sent", "invalid_token", "error"
	Error  string `json:"error,omitempty"`
}
