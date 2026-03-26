package gravatar

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// URL returns the Gravatar URL for the given email address.
// Returns an empty string if the email is empty.
func URL(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ""
	}
	hash := fmt.Sprintf("%x", md5.Sum([]byte(email)))
	return "https://gravatar.com/avatar/" + hash + "?d=404&s=256"
}
