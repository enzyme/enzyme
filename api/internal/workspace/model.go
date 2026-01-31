package workspace

import (
	"time"
)

type Workspace struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	IconURL   *string   `json:"icon_url,omitempty"`
	Settings  string    `json:"settings"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Membership struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	WorkspaceID         string    `json:"workspace_id"`
	Role                string    `json:"role"`
	DisplayNameOverride *string   `json:"display_name_override,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type MemberWithUser struct {
	Membership
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
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

// CanCreateChannels returns true if the role can create channels
func CanCreateChannels(role string) bool {
	return role == RoleOwner || role == RoleAdmin || role == RoleMember
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
