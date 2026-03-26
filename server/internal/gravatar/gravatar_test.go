package gravatar

import (
	"testing"
)

func TestURL(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "empty email",
			email: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			email: "   ",
			want:  "",
		},
		{
			name:  "normal email",
			email: "test@example.com",
			want:  "https://gravatar.com/avatar/55502f40dc8b7c769880b10874abc9d0?d=404&s=256",
		},
		{
			name:  "uppercase normalized",
			email: "Test@Example.COM",
			want:  "https://gravatar.com/avatar/55502f40dc8b7c769880b10874abc9d0?d=404&s=256",
		},
		{
			name:  "leading/trailing whitespace trimmed",
			email: "  test@example.com  ",
			want:  "https://gravatar.com/avatar/55502f40dc8b7c769880b10874abc9d0?d=404&s=256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URL(tt.email)
			if got != tt.want {
				t.Errorf("URL(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}
