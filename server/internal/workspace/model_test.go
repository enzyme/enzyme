package workspace

import "testing"

func TestCanManageMembers(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"owner can manage", RoleOwner, true},
		{"admin can manage", RoleAdmin, true},
		{"member cannot manage", RoleMember, false},
		{"guest cannot manage", RoleGuest, false},
		{"invalid role cannot manage", "invalid", false},
		{"empty role cannot manage", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanManageMembers(tt.role); got != tt.want {
				t.Errorf("CanManageMembers(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestCanChangeRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"owner can change", RoleOwner, true},
		{"admin can change", RoleAdmin, true},
		{"member cannot change", RoleMember, false},
		{"guest cannot change", RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanChangeRole(tt.role); got != tt.want {
				t.Errorf("CanChangeRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestCanDeleteWorkspace(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"owner can delete", RoleOwner, true},
		{"admin cannot delete", RoleAdmin, false},
		{"member cannot delete", RoleMember, false},
		{"guest cannot delete", RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanDeleteWorkspace(tt.role); got != tt.want {
				t.Errorf("CanDeleteWorkspace(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name  string
		role  string
		level PermissionLevel
		want  bool
	}{
		// everyone level: all roles allowed
		{"everyone/owner", RoleOwner, PermissionEveryone, true},
		{"everyone/admin", RoleAdmin, PermissionEveryone, true},
		{"everyone/member", RoleMember, PermissionEveryone, true},
		{"everyone/guest", RoleGuest, PermissionEveryone, true},
		{"everyone/invalid", "invalid", PermissionEveryone, false},

		// members level: guest excluded
		{"members/owner", RoleOwner, PermissionMembers, true},
		{"members/admin", RoleAdmin, PermissionMembers, true},
		{"members/member", RoleMember, PermissionMembers, true},
		{"members/guest", RoleGuest, PermissionMembers, false},
		{"members/invalid", "invalid", PermissionMembers, false},

		// admins level: only admin and owner
		{"admins/owner", RoleOwner, PermissionAdmins, true},
		{"admins/admin", RoleAdmin, PermissionAdmins, true},
		{"admins/member", RoleMember, PermissionAdmins, false},
		{"admins/guest", RoleGuest, PermissionAdmins, false},
		{"admins/invalid", "invalid", PermissionAdmins, false},

		// invalid level
		{"invalid level", RoleOwner, PermissionLevel("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasPermission(tt.role, tt.level); got != tt.want {
				t.Errorf("HasPermission(%q, %q) = %v, want %v", tt.role, tt.level, got, tt.want)
			}
		})
	}
}

func TestRoleRank(t *testing.T) {
	tests := []struct {
		name string
		role string
		want int
	}{
		{"owner has highest rank", RoleOwner, 4},
		{"admin has second highest", RoleAdmin, 3},
		{"member has third highest", RoleMember, 2},
		{"guest has lowest valid rank", RoleGuest, 1},
		{"invalid role has zero rank", "invalid", 0},
		{"empty role has zero rank", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleRank(tt.role); got != tt.want {
				t.Errorf("RoleRank(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRoleRank_Ordering(t *testing.T) {
	// Verify that role ranks are properly ordered
	if RoleRank(RoleOwner) <= RoleRank(RoleAdmin) {
		t.Error("owner should rank higher than admin")
	}
	if RoleRank(RoleAdmin) <= RoleRank(RoleMember) {
		t.Error("admin should rank higher than member")
	}
	if RoleRank(RoleMember) <= RoleRank(RoleGuest) {
		t.Error("member should rank higher than guest")
	}
	if RoleRank(RoleGuest) <= RoleRank("invalid") {
		t.Error("guest should rank higher than invalid")
	}
}

func TestParseSettings(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected WorkspaceSettings
	}{
		{
			name:     "empty string returns defaults",
			json:     "",
			expected: DefaultSettings(),
		},
		{
			name:     "empty object returns defaults",
			json:     "{}",
			expected: DefaultSettings(),
		},
		{
			name:     "show_join_leave_messages true",
			json:     `{"show_join_leave_messages":true}`,
			expected: DefaultSettings(),
		},
		{
			name: "show_join_leave_messages false",
			json: `{"show_join_leave_messages":false}`,
			expected: WorkspaceSettings{
				ShowJoinLeaveMessages:   false,
				WhoCanCreateChannels:    PermissionMembers,
				WhoCanCreateInvites:     PermissionAdmins,
				WhoCanPinMessages:       PermissionMembers,
				WhoCanManageCustomEmoji: PermissionMembers,
			},
		},
		{
			name:     "invalid json returns defaults",
			json:     "not json",
			expected: DefaultSettings(),
		},
		{
			name: "permission fields override defaults",
			json: `{"who_can_create_channels":"admins","who_can_create_invites":"members","who_can_pin_messages":"everyone","who_can_manage_custom_emoji":"admins"}`,
			expected: WorkspaceSettings{
				ShowJoinLeaveMessages:   true,
				WhoCanCreateChannels:    PermissionAdmins,
				WhoCanCreateInvites:     PermissionMembers,
				WhoCanPinMessages:       PermissionEveryone,
				WhoCanManageCustomEmoji: PermissionAdmins,
			},
		},
		{
			name:     "backward compat: missing permission fields get defaults",
			json:     `{"show_join_leave_messages":true,"who_can_create_channels":"members"}`,
			expected: DefaultSettings(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSettings(tt.json)
			if got != tt.expected {
				t.Errorf("ParseSettings(%q) = %+v, want %+v", tt.json, got, tt.expected)
			}
		})
	}
}

func TestWorkspaceSettings_ToJSON(t *testing.T) {
	settings := WorkspaceSettings{
		ShowJoinLeaveMessages:   false,
		WhoCanCreateChannels:    PermissionAdmins,
		WhoCanCreateInvites:     PermissionMembers,
		WhoCanPinMessages:       PermissionEveryone,
		WhoCanManageCustomEmoji: PermissionAdmins,
	}
	jsonStr := settings.ToJSON()

	// Verify round-trip
	parsed := ParseSettings(jsonStr)
	if parsed != settings {
		t.Errorf("Round-trip failed: got %+v, want %+v", parsed, settings)
	}
}

func TestDefaultSettings(t *testing.T) {
	defaults := DefaultSettings()
	if !defaults.ShowJoinLeaveMessages {
		t.Error("default ShowJoinLeaveMessages should be true")
	}
	if defaults.WhoCanCreateChannels != PermissionMembers {
		t.Errorf("default WhoCanCreateChannels should be %q, got %q", PermissionMembers, defaults.WhoCanCreateChannels)
	}
	if defaults.WhoCanCreateInvites != PermissionAdmins {
		t.Errorf("default WhoCanCreateInvites should be %q, got %q", PermissionAdmins, defaults.WhoCanCreateInvites)
	}
	if defaults.WhoCanPinMessages != PermissionMembers {
		t.Errorf("default WhoCanPinMessages should be %q, got %q", PermissionMembers, defaults.WhoCanPinMessages)
	}
	if defaults.WhoCanManageCustomEmoji != PermissionMembers {
		t.Errorf("default WhoCanManageCustomEmoji should be %q, got %q", PermissionMembers, defaults.WhoCanManageCustomEmoji)
	}
}

func TestWorkspace_ParsedSettings(t *testing.T) {
	ws := &Workspace{Settings: `{"show_join_leave_messages":false}`}
	settings := ws.ParsedSettings()
	if settings.ShowJoinLeaveMessages {
		t.Error("ParsedSettings should return false for show_join_leave_messages")
	}
}
