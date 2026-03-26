package user

import (
	"time"
)

type User struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	PasswordHash    string     `json:"-"`
	DisplayName     string     `json:"display_name"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreateUserInput struct {
	Email        string
	Password     string
	DisplayName  string
	PasswordHash string
}
