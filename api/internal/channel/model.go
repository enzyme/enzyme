package channel

import (
	"time"
)

type Channel struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspace_id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	Type              string     `json:"type"`
	DMParticipantHash *string    `json:"dm_participant_hash,omitempty"`
	ArchivedAt        *time.Time `json:"archived_at,omitempty"`
	CreatedBy         *string    `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ChannelMembership struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	ChannelID         string    `json:"channel_id"`
	ChannelRole       *string   `json:"channel_role,omitempty"`
	LastReadMessageID *string   `json:"last_read_message_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ChannelWithMembership struct {
	Channel
	ChannelRole       *string `json:"channel_role,omitempty"`
	LastReadMessageID *string `json:"last_read_message_id,omitempty"`
	UnreadCount       int     `json:"unread_count"`
}

type MemberInfo struct {
	UserID      string  `json:"user_id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	ChannelRole *string `json:"channel_role,omitempty"`
}

const (
	TypePublic  = "public"
	TypePrivate = "private"
	TypeDM      = "dm"
	TypeGroupDM = "group_dm"
)

const (
	ChannelRoleAdmin  = "admin"
	ChannelRolePoster = "poster"
	ChannelRoleViewer = "viewer"
)

// CanPost returns true if the role allows posting messages
func CanPost(role *string) bool {
	if role == nil {
		return true // Default allows posting
	}
	return *role == ChannelRoleAdmin || *role == ChannelRolePoster
}

// CanManageChannel returns true if the role allows managing channel settings
func CanManageChannel(role *string) bool {
	if role == nil {
		return false
	}
	return *role == ChannelRoleAdmin
}
