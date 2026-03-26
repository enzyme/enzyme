package linkpreview

import "testing"

func TestParseInternalMessageURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantNil bool
		want    *InternalMessageRef
	}{
		{
			name:   "valid msg param",
			rawURL: "https://chat.example.com/workspaces/WS1/channels/CH1?msg=MSG1",
			want:   &InternalMessageRef{WorkspaceID: "WS1", ChannelID: "CH1", MessageID: "MSG1"},
		},
		{
			name:   "valid thread param",
			rawURL: "https://chat.example.com/workspaces/WS1/channels/CH1?thread=MSG2",
			want:   &InternalMessageRef{WorkspaceID: "WS1", ChannelID: "CH1", MessageID: "MSG2"},
		},
		{
			name:   "msg param preferred over thread",
			rawURL: "https://example.com/workspaces/WS1/channels/CH1?msg=MSG1&thread=MSG2",
			want:   &InternalMessageRef{WorkspaceID: "WS1", ChannelID: "CH1", MessageID: "MSG1"},
		},
		{
			name:   "no host (relative URL)",
			rawURL: "/workspaces/WS1/channels/CH1?msg=MSG1",
			want:   &InternalMessageRef{WorkspaceID: "WS1", ChannelID: "CH1", MessageID: "MSG1"},
		},
		{
			name:   "trailing slash",
			rawURL: "https://example.com/workspaces/WS1/channels/CH1/?msg=MSG1",
			want:   &InternalMessageRef{WorkspaceID: "WS1", ChannelID: "CH1", MessageID: "MSG1"},
		},
		{
			name:    "missing msg and thread params",
			rawURL:  "https://example.com/workspaces/WS1/channels/CH1",
			wantNil: true,
		},
		{
			name:    "wrong path structure",
			rawURL:  "https://example.com/foo/bar?msg=MSG1",
			wantNil: true,
		},
		{
			name:    "extra path segments",
			rawURL:  "https://example.com/workspaces/WS1/channels/CH1/extra?msg=MSG1",
			wantNil: true,
		},
		{
			name:    "trailing slash with empty channel segment",
			rawURL:  "https://example.com/workspaces/WS1/channels/?msg=MSG1",
			wantNil: true,
		},
		{
			name:    "totally different URL",
			rawURL:  "https://google.com/search?q=hello",
			wantNil: true,
		},
		{
			name:    "empty string",
			rawURL:  "",
			wantNil: true,
		},
		{
			name:    "empty msg param",
			rawURL:  "https://example.com/workspaces/WS1/channels/CH1?msg=",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseInternalMessageURL(tt.rawURL)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.WorkspaceID != tt.want.WorkspaceID {
				t.Errorf("WorkspaceID = %q, want %q", got.WorkspaceID, tt.want.WorkspaceID)
			}
			if got.ChannelID != tt.want.ChannelID {
				t.Errorf("ChannelID = %q, want %q", got.ChannelID, tt.want.ChannelID)
			}
			if got.MessageID != tt.want.MessageID {
				t.Errorf("MessageID = %q, want %q", got.MessageID, tt.want.MessageID)
			}
		})
	}
}

func TestIsInternalURL(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		want   bool
	}{
		{"workspace channel URL", "https://example.com/workspaces/WS1/channels/CH1", true},
		{"workspace channel with msg", "https://example.com/workspaces/WS1/channels/CH1?msg=M1", true},
		{"relative workspace URL", "/workspaces/WS1/channels/CH1", true},
		{"external URL", "https://google.com/search?q=hello", false},
		{"empty string", "", false},
		{"API URL", "https://example.com/api/auth/login", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInternalURL(tt.rawURL); got != tt.want {
				t.Errorf("IsInternalURL(%q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}
