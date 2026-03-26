package thread

import "time"

// Subscription status constants
const (
	StatusSubscribed   = "subscribed"
	StatusUnsubscribed = "unsubscribed"
)

// Subscription represents a user's subscription to a thread
type Subscription struct {
	ID              string    `json:"id"`
	ThreadParentID  string    `json:"thread_parent_id"`
	UserID          string    `json:"user_id"`
	Status          string    `json:"status"`
	LastReadReplyID *string   `json:"last_read_reply_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// IsSubscribed returns true if the subscription status is "subscribed"
func (s *Subscription) IsSubscribed() bool {
	return s.Status == StatusSubscribed
}
