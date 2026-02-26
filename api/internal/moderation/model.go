package moderation

import (
	"time"
)

// Ban represents a workspace ban on a user
type Ban struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspace_id"`
	UserID       string     `json:"user_id"`
	BannedBy     string     `json:"banned_by"`
	Reason       *string    `json:"reason,omitempty"`
	HideMessages bool       `json:"hide_messages"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// BanWithUser includes user display info for the banned user
type BanWithUser struct {
	Ban
	UserDisplayName string  `json:"user_display_name"`
	UserEmail       string  `json:"user_email"`
	UserAvatarURL   *string `json:"user_avatar_url,omitempty"`
	BannedByName    string  `json:"banned_by_name"`
}

// Block represents one user blocking another within a workspace
type Block struct {
	WorkspaceID string    `json:"workspace_id"`
	BlockerID   string    `json:"blocker_id"`
	BlockedID   string    `json:"blocked_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// BlockWithUser includes display info for the blocked user
type BlockWithUser struct {
	Block
	DisplayName string  `json:"display_name"`
	Email       string  `json:"email"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// AuditLogEntry represents a moderation action in the audit log
type AuditLogEntry struct {
	ID         string    `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Metadata   *string   `json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuditLogEntryWithActor includes actor and target display info
type AuditLogEntryWithActor struct {
	AuditLogEntry
	ActorDisplayName  string  `json:"actor_display_name"`
	ActorAvatarURL    *string `json:"actor_avatar_url,omitempty"`
	TargetDisplayName *string `json:"target_display_name,omitempty"`
}

// Moderation action constants
const (
	ActionUserBanned      = "user.banned"
	ActionUserUnbanned    = "user.unbanned"
	ActionMessageDeleted  = "message.deleted"
	ActionMemberRemoved   = "member.removed"
	ActionMemberRoleChanged = "member.role_changed"
	ActionChannelArchived = "channel.archived"
)

// Target type constants
const (
	TargetTypeUser    = "user"
	TargetTypeMessage = "message"
	TargetTypeChannel = "channel"
)
