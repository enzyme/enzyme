package channel

import "testing"

func TestCanPost(t *testing.T) {
	admin := ChannelRoleAdmin
	poster := ChannelRolePoster
	viewer := ChannelRoleViewer

	tests := []struct {
		name string
		role *string
		want bool
	}{
		{"nil role can post", nil, true},
		{"admin can post", &admin, true},
		{"poster can post", &poster, true},
		{"viewer cannot post", &viewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanPost(tt.role); got != tt.want {
				roleStr := "nil"
				if tt.role != nil {
					roleStr = *tt.role
				}
				t.Errorf("CanPost(%s) = %v, want %v", roleStr, got, tt.want)
			}
		})
	}
}

func TestCanManageChannel(t *testing.T) {
	admin := ChannelRoleAdmin
	poster := ChannelRolePoster
	viewer := ChannelRoleViewer

	tests := []struct {
		name string
		role *string
		want bool
	}{
		{"nil role cannot manage", nil, false},
		{"admin can manage", &admin, true},
		{"poster cannot manage", &poster, false},
		{"viewer cannot manage", &viewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanManageChannel(tt.role); got != tt.want {
				roleStr := "nil"
				if tt.role != nil {
					roleStr = *tt.role
				}
				t.Errorf("CanManageChannel(%s) = %v, want %v", roleStr, got, tt.want)
			}
		})
	}
}

func TestChannelTypeConstants(t *testing.T) {
	// Verify type constants have expected values
	if TypePublic != "public" {
		t.Errorf("TypePublic = %q, want %q", TypePublic, "public")
	}
	if TypePrivate != "private" {
		t.Errorf("TypePrivate = %q, want %q", TypePrivate, "private")
	}
	if TypeDM != "dm" {
		t.Errorf("TypeDM = %q, want %q", TypeDM, "dm")
	}
	if TypeGroupDM != "group_dm" {
		t.Errorf("TypeGroupDM = %q, want %q", TypeGroupDM, "group_dm")
	}
}

func TestChannelRoleConstants(t *testing.T) {
	// Verify role constants have expected values
	if ChannelRoleAdmin != "admin" {
		t.Errorf("ChannelRoleAdmin = %q, want %q", ChannelRoleAdmin, "admin")
	}
	if ChannelRolePoster != "poster" {
		t.Errorf("ChannelRolePoster = %q, want %q", ChannelRolePoster, "poster")
	}
	if ChannelRoleViewer != "viewer" {
		t.Errorf("ChannelRoleViewer = %q, want %q", ChannelRoleViewer, "viewer")
	}
}
