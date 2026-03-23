package pushnotification

import "time"

// DeviceToken represents a registered push notification device token.
type DeviceToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"` // "fcm" or "apns"
	DeviceID  string    `json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationData contains the data needed to send a push notification.
type NotificationData struct {
	Title       string
	Body        string
	ChannelID   string
	MessageID   string
	WorkspaceID string
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
	ServerURL   string `json:"server_url"`
}

// RelayResponse is the response from the push relay service.
type RelayResponse struct {
	Status string `json:"status"` // "sent", "invalid_token", "error"
	Error  string `json:"error,omitempty"`
}
