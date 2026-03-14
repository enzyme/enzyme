package workspace

import (
	"encoding/json"
	"time"
)

// WorkspaceSettings contains parsed workspace settings
type WorkspaceSettings struct {
	ShowJoinLeaveMessages   bool            `json:"show_join_leave_messages"`
	WhoCanCreateChannels    PermissionLevel `json:"who_can_create_channels"`
	WhoCanCreateInvites     PermissionLevel `json:"who_can_create_invites"`
	WhoCanPinMessages       PermissionLevel `json:"who_can_pin_messages"`
	WhoCanManageCustomEmoji PermissionLevel `json:"who_can_manage_custom_emoji"`
}

// DefaultSettings returns the default workspace settings
func DefaultSettings() WorkspaceSettings {
	return WorkspaceSettings{
		ShowJoinLeaveMessages:   true,
		WhoCanCreateChannels:    PermissionMembers,
		WhoCanCreateInvites:     PermissionAdmins,
		WhoCanPinMessages:       PermissionMembers,
		WhoCanManageCustomEmoji: PermissionMembers,
	}
}

// ParseSettings parses the settings JSON string into WorkspaceSettings
func ParseSettings(settingsJSON string) WorkspaceSettings {
	settings := DefaultSettings()
	if settingsJSON != "" && settingsJSON != "{}" {
		_ = json.Unmarshal([]byte(settingsJSON), &settings)
	}
	// Reset invalid permission levels to defaults
	defaults := DefaultSettings()
	if !IsValidPermissionLevel(settings.WhoCanCreateChannels) {
		settings.WhoCanCreateChannels = defaults.WhoCanCreateChannels
	}
	if !IsValidPermissionLevel(settings.WhoCanCreateInvites) {
		settings.WhoCanCreateInvites = defaults.WhoCanCreateInvites
	}
	if !IsValidPermissionLevel(settings.WhoCanPinMessages) {
		settings.WhoCanPinMessages = defaults.WhoCanPinMessages
	}
	if !IsValidPermissionLevel(settings.WhoCanManageCustomEmoji) {
		settings.WhoCanManageCustomEmoji = defaults.WhoCanManageCustomEmoji
	}
	return settings
}

// ToJSON serializes WorkspaceSettings to a JSON string
func (s WorkspaceSettings) ToJSON() string {
	data, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}
	return string(data)
}

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IconURL   *string   `json:"icon_url,omitempty"`
	Settings  string    `json:"settings"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ParsedSettings returns the parsed workspace settings
func (w *Workspace) ParsedSettings() WorkspaceSettings {
	return ParseSettings(w.Settings)
}

type Membership struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	WorkspaceID         string    `json:"workspace_id"`
	Role                string    `json:"role"`
	DisplayNameOverride *string   `json:"display_name_override,omitempty"`
	SortOrder           *int      `json:"sort_order,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type MemberWithUser struct {
	Membership
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	IsBanned    bool    `json:"is_banned"`
}

type Invite struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspace_id"`
	Code         string     `json:"code"`
	InvitedEmail *string    `json:"invited_email,omitempty"`
	Role         string     `json:"role"`
	CreatedBy    *string    `json:"created_by,omitempty"`
	MaxUses      *int       `json:"max_uses,omitempty"`
	UseCount     int        `json:"use_count"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// PermissionLevel controls which roles can perform a given action
type PermissionLevel string

const (
	PermissionEveryone PermissionLevel = "everyone"
	PermissionMembers  PermissionLevel = "members"
	PermissionAdmins   PermissionLevel = "admins"
)

// HasPermission returns true if the given role satisfies the required permission level
func HasPermission(role string, level PermissionLevel) bool {
	switch level {
	case PermissionEveryone:
		return role == RoleOwner || role == RoleAdmin || role == RoleMember || role == RoleGuest
	case PermissionMembers:
		return role == RoleOwner || role == RoleAdmin || role == RoleMember
	case PermissionAdmins:
		return role == RoleOwner || role == RoleAdmin
	default:
		return false
	}
}

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleGuest  = "guest"
)

// CanManageMembers returns true if the role can add/remove members
func CanManageMembers(role string) bool {
	return role == RoleOwner || role == RoleAdmin
}

// CanChangeRole returns true if the role can change other members' roles
func CanChangeRole(role string) bool {
	return role == RoleOwner || role == RoleAdmin
}

// CanDeleteWorkspace returns true if the role can delete the workspace
func CanDeleteWorkspace(role string) bool {
	return role == RoleOwner
}

// IsValidPermissionLevel returns true if the level is a known permission level
func IsValidPermissionLevel(level PermissionLevel) bool {
	return level == PermissionEveryone || level == PermissionMembers || level == PermissionAdmins
}

// RoleRank returns the rank of a role (higher = more permissions)
func RoleRank(role string) int {
	switch role {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleMember:
		return 2
	case RoleGuest:
		return 1
	default:
		return 0
	}
}
