package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/feather/api/internal/user"
)

var (
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrUserDeactivated      = errors.New("user account is deactivated")
	ErrInvalidResetToken    = errors.New("invalid or expired reset token")
	ErrPasswordTooShort     = errors.New("password must be at least 8 characters")
	ErrDisplayNameRequired  = errors.New("display name is required")
	ErrInvalidEmail         = errors.New("invalid email address")
)

type Service struct {
	userRepo       *user.Repository
	passwordResets PasswordResetRepository
	bcryptCost     int
}

type PasswordResetRepository interface {
	Create(ctx context.Context, userID string, token string, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*PasswordReset, error)
	MarkUsed(ctx context.Context, id string) error
}

type PasswordReset struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func NewService(userRepo *user.Repository, passwordResets PasswordResetRepository, bcryptCost int) *Service {
	return &Service{
		userRepo:       userRepo,
		passwordResets: passwordResets,
		bcryptCost:     bcryptCost,
	}
}

type RegisterInput struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*user.User, error) {
	if err := validateEmail(input.Email); err != nil {
		return nil, err
	}
	if len(input.Password) < 8 {
		return nil, ErrPasswordTooShort
	}
	if input.DisplayName == "" {
		return nil, ErrDisplayNameRequired
	}

	hash, err := HashPassword(input.Password, s.bcryptCost)
	if err != nil {
		return nil, err
	}

	return s.userRepo.Create(ctx, user.CreateUserInput{
		Email:        input.Email,
		DisplayName:  input.DisplayName,
		PasswordHash: hash,
	})
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*user.User, error) {
	u, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if u.Status == "deactivated" {
		return nil, ErrUserDeactivated
	}

	if !CheckPassword(input.Password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*user.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *Service) CreatePasswordResetToken(ctx context.Context, email string) (string, error) {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			// Don't reveal if email exists
			return "", nil
		}
		return "", err
	}

	token := generateSecureToken(32)
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.passwordResets.Create(ctx, u.ID, token, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	reset, err := s.passwordResets.GetByToken(ctx, token)
	if err != nil {
		return ErrInvalidResetToken
	}

	if reset.UsedAt != nil || time.Now().After(reset.ExpiresAt) {
		return ErrInvalidResetToken
	}

	hash, err := HashPassword(newPassword, s.bcryptCost)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, reset.UserID, hash); err != nil {
		return err
	}

	return s.passwordResets.MarkUsed(ctx, reset.ID)
}

func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	// Basic validation - just check for @ symbol
	hasAt := false
	for _, c := range email {
		if c == '@' {
			hasAt = true
			break
		}
	}
	if !hasAt {
		return ErrInvalidEmail
	}
	return nil
}

func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
