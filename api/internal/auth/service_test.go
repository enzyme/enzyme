package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/enzyme/api/internal/testutil"
	"github.com/enzyme/api/internal/user"
)

// mockPasswordResetRepository is a mock implementation of PasswordResetRepository
type mockPasswordResetRepository struct {
	Resets    map[string]*PasswordReset
	CreateErr error
	GetErr    error
	MarkErr   error
}

func newMockPasswordResetRepository() *mockPasswordResetRepository {
	return &mockPasswordResetRepository{
		Resets: make(map[string]*PasswordReset),
	}
}

func (m *mockPasswordResetRepository) Create(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.Resets[token] = &PasswordReset{
		ID:        "reset-" + token[:8],
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (m *mockPasswordResetRepository) GetByToken(ctx context.Context, token string) (*PasswordReset, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	reset, ok := m.Resets[token]
	if !ok {
		return nil, ErrInvalidResetToken
	}
	return reset, nil
}

func (m *mockPasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	if m.MarkErr != nil {
		return m.MarkErr
	}
	for _, reset := range m.Resets {
		if reset.ID == id {
			now := time.Now()
			reset.UsedAt = &now
			return nil
		}
	}
	return nil
}

// mockEmailVerificationRepository is a mock implementation of EmailVerificationRepository
type mockEmailVerificationRepository struct {
	Verifications map[string]*EmailVerification
	CreateErr     error
	GetErr        error
	DeleteErr     error
}

func newMockEmailVerificationRepository() *mockEmailVerificationRepository {
	return &mockEmailVerificationRepository{
		Verifications: make(map[string]*EmailVerification),
	}
}

func (m *mockEmailVerificationRepository) Create(ctx context.Context, userID string, token string, expiresAt time.Time) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	// Delete existing tokens for user (mirrors real behavior)
	for k, v := range m.Verifications {
		if v.UserID == userID {
			delete(m.Verifications, k)
		}
	}
	m.Verifications[token] = &EmailVerification{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (m *mockEmailVerificationRepository) GetByToken(ctx context.Context, token string) (*EmailVerification, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	ev, ok := m.Verifications[token]
	if !ok {
		return nil, ErrInvalidVerificationToken
	}
	return ev, nil
}

func (m *mockEmailVerificationRepository) DeleteForUser(ctx context.Context, userID string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	for k, v := range m.Verifications {
		if v.UserID == userID {
			delete(m.Verifications, k)
		}
	}
	return nil
}

func newTestService(t *testing.T) (*Service, *user.Repository, *mockPasswordResetRepository, *mockEmailVerificationRepository) {
	t.Helper()
	db := testutil.TestDB(t)
	userRepo := user.NewRepository(db)
	mockResets := newMockPasswordResetRepository()
	mockVerifications := newMockEmailVerificationRepository()
	svc := NewService(userRepo, mockResets, mockVerifications, 4) // Low bcrypt cost for tests
	return svc, userRepo, mockResets, mockVerifications
}

func TestService_Register(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	u, err := svc.Register(ctx, RegisterInput{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if u.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "test@example.com")
	}
	if u.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", u.DisplayName, "Test User")
	}
}

func TestService_Register_PasswordTooShort(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Email:       "test@example.com",
		Password:    "short",
		DisplayName: "Test User",
	})
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("Register() error = %v, want %v", err, ErrPasswordTooShort)
	}
}

func TestService_Register_DisplayNameRequired(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterInput{
		Email:       "test@example.com",
		Password:    "password123",
		DisplayName: "",
	})
	if !errors.Is(err, ErrDisplayNameRequired) {
		t.Errorf("Register() error = %v, want %v", err, ErrDisplayNameRequired)
	}
}

func TestService_Register_InvalidEmail(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"no at symbol", "testexample.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, RegisterInput{
				Email:       tt.email,
				Password:    "password123",
				DisplayName: "Test User",
			})
			if !errors.Is(err, ErrInvalidEmail) {
				t.Errorf("Register() error = %v, want %v", err, ErrInvalidEmail)
			}
		})
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	input := RegisterInput{
		Email:       "duplicate@example.com",
		Password:    "password123",
		DisplayName: "Test User",
	}

	// First registration should succeed
	_, err := svc.Register(ctx, input)
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	// Second registration with same email should fail
	_, err = svc.Register(ctx, input)
	if !errors.Is(err, user.ErrEmailAlreadyInUse) {
		t.Errorf("second Register() error = %v, want %v", err, user.ErrEmailAlreadyInUse)
	}
}

func TestService_Login(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	// Register a user first
	svc.Register(ctx, RegisterInput{
		Email:       "login@example.com",
		Password:    "password123",
		DisplayName: "Login User",
	})

	// Login should succeed with correct credentials
	u, err := svc.Login(ctx, LoginInput{
		Email:    "login@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if u.Email != "login@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "login@example.com")
	}
}

func TestService_Login_InvalidCredentials(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	// Register a user first
	svc.Register(ctx, RegisterInput{
		Email:       "login@example.com",
		Password:    "password123",
		DisplayName: "Login User",
	})

	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"wrong password", "login@example.com", "wrongpassword"},
		{"nonexistent email", "nonexistent@example.com", "password123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(ctx, LoginInput{
				Email:    tt.email,
				Password: tt.password,
			})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Login() error = %v, want %v", err, ErrInvalidCredentials)
			}
		})
	}
}

func TestService_Login_DeactivatedUser(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)

	ctx := context.Background()

	// Register and deactivate user
	u, _ := svc.Register(ctx, RegisterInput{
		Email:       "deactivated@example.com",
		Password:    "password123",
		DisplayName: "Deactivated User",
	})

	u.Status = "deactivated"
	userRepo.Update(ctx, u)

	_, err := svc.Login(ctx, LoginInput{
		Email:    "deactivated@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrUserDeactivated) {
		t.Errorf("Login() error = %v, want %v", err, ErrUserDeactivated)
	}
}

func TestService_GetCurrentUser(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	registered, _ := svc.Register(ctx, RegisterInput{
		Email:       "current@example.com",
		Password:    "password123",
		DisplayName: "Current User",
	})

	u, err := svc.GetCurrentUser(ctx, registered.ID)
	if err != nil {
		t.Fatalf("GetCurrentUser() error = %v", err)
	}

	if u.ID != registered.ID {
		t.Errorf("ID = %q, want %q", u.ID, registered.ID)
	}
}

func TestService_CreatePasswordResetToken(t *testing.T) {
	svc, _, mockResets, _ := newTestService(t)

	ctx := context.Background()

	// Register a user first
	svc.Register(ctx, RegisterInput{
		Email:       "reset@example.com",
		Password:    "password123",
		DisplayName: "Reset User",
	})

	token, err := svc.CreatePasswordResetToken(ctx, "reset@example.com")
	if err != nil {
		t.Fatalf("CreatePasswordResetToken() error = %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}

	// Verify token was stored
	if len(mockResets.Resets) != 1 {
		t.Errorf("len(Resets) = %d, want 1", len(mockResets.Resets))
	}
}

func TestService_CreatePasswordResetToken_NonexistentEmail(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	// Should return empty token without error (don't reveal if email exists)
	token, err := svc.CreatePasswordResetToken(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("CreatePasswordResetToken() error = %v", err)
	}

	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

func TestService_ResetPassword(t *testing.T) {
	svc, _, mockResets, _ := newTestService(t)

	ctx := context.Background()

	// Register a user
	svc.Register(ctx, RegisterInput{
		Email:       "reset@example.com",
		Password:    "oldpassword123",
		DisplayName: "Reset User",
	})

	// Create reset token
	token, _ := svc.CreatePasswordResetToken(ctx, "reset@example.com")

	// Reset password
	err := svc.ResetPassword(ctx, token, "newpassword123")
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	// Verify new password works
	_, err = svc.Login(ctx, LoginInput{
		Email:    "reset@example.com",
		Password: "newpassword123",
	})
	if err != nil {
		t.Errorf("Login with new password error = %v", err)
	}

	// Verify old password doesn't work
	_, err = svc.Login(ctx, LoginInput{
		Email:    "reset@example.com",
		Password: "oldpassword123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("Login with old password error = %v, want %v", err, ErrInvalidCredentials)
	}

	// Verify token was marked as used
	reset := mockResets.Resets[token]
	if reset.UsedAt == nil {
		t.Error("expected token to be marked as used")
	}
}

func TestService_ResetPassword_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	err := svc.ResetPassword(ctx, "invalid-token", "newpassword123")
	if !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("ResetPassword() error = %v, want %v", err, ErrInvalidResetToken)
	}
}

func TestService_ResetPassword_ExpiredToken(t *testing.T) {
	svc, _, mockResets, _ := newTestService(t)

	ctx := context.Background()

	// Register a user
	svc.Register(ctx, RegisterInput{
		Email:       "reset@example.com",
		Password:    "password123",
		DisplayName: "Reset User",
	})

	// Manually create an expired reset token
	expiredToken := "expired-token"
	mockResets.Resets[expiredToken] = &PasswordReset{
		ID:        "reset-id",
		UserID:    "user-id",
		Token:     expiredToken,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	err := svc.ResetPassword(ctx, expiredToken, "newpassword123")
	if !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("ResetPassword() error = %v, want %v", err, ErrInvalidResetToken)
	}
}

func TestService_ResetPassword_UsedToken(t *testing.T) {
	svc, _, mockResets, _ := newTestService(t)

	ctx := context.Background()

	// Register a user
	svc.Register(ctx, RegisterInput{
		Email:       "reset@example.com",
		Password:    "password123",
		DisplayName: "Reset User",
	})

	// Manually create a used reset token
	usedToken := "used-token"
	usedAt := time.Now()
	mockResets.Resets[usedToken] = &PasswordReset{
		ID:        "reset-id",
		UserID:    "user-id",
		Token:     usedToken,
		ExpiresAt: time.Now().Add(1 * time.Hour), // Not expired
		UsedAt:    &usedAt,                       // Already used
	}

	err := svc.ResetPassword(ctx, usedToken, "newpassword123")
	if !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("ResetPassword() error = %v, want %v", err, ErrInvalidResetToken)
	}
}

func TestService_ResetPassword_PasswordTooShort(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	ctx := context.Background()

	// Register a user
	svc.Register(ctx, RegisterInput{
		Email:       "reset@example.com",
		Password:    "password123",
		DisplayName: "Reset User",
	})

	token, _ := svc.CreatePasswordResetToken(ctx, "reset@example.com")

	err := svc.ResetPassword(ctx, token, "short")
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("ResetPassword() error = %v, want %v", err, ErrPasswordTooShort)
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("password123", 4)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == "password123" {
		t.Error("hash should not equal plain password")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("password123", 4)

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"correct password", "password123", true},
		{"wrong password", "wrongpassword", false},
		{"empty password", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPassword(tt.password, hash); got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_CreateEmailVerificationToken(t *testing.T) {
	svc, _, _, mockVerifications := newTestService(t)
	ctx := context.Background()

	u, _ := svc.Register(ctx, RegisterInput{
		Email:       "verify@example.com",
		Password:    "password123",
		DisplayName: "Verify User",
	})

	token, err := svc.CreateEmailVerificationToken(ctx, u.ID)
	if err != nil {
		t.Fatalf("CreateEmailVerificationToken() error = %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}

	if len(mockVerifications.Verifications) != 1 {
		t.Errorf("len(Verifications) = %d, want 1", len(mockVerifications.Verifications))
	}
}

func TestService_CreateEmailVerificationToken_ReplacesExisting(t *testing.T) {
	svc, _, _, mockVerifications := newTestService(t)
	ctx := context.Background()

	u, _ := svc.Register(ctx, RegisterInput{
		Email:       "verify@example.com",
		Password:    "password123",
		DisplayName: "Verify User",
	})

	token1, _ := svc.CreateEmailVerificationToken(ctx, u.ID)
	token2, _ := svc.CreateEmailVerificationToken(ctx, u.ID)

	if token1 == token2 {
		t.Error("expected different tokens")
	}

	// Only one token should exist (old one deleted)
	if len(mockVerifications.Verifications) != 1 {
		t.Errorf("len(Verifications) = %d, want 1", len(mockVerifications.Verifications))
	}

	// The remaining token should be the new one
	if _, ok := mockVerifications.Verifications[token2]; !ok {
		t.Error("expected second token to be stored")
	}
}

func TestService_VerifyEmail(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)
	ctx := context.Background()

	u, _ := svc.Register(ctx, RegisterInput{
		Email:       "verify@example.com",
		Password:    "password123",
		DisplayName: "Verify User",
	})

	// User should not be verified yet
	fetched, _ := userRepo.GetByID(ctx, u.ID)
	if fetched.EmailVerifiedAt != nil {
		t.Error("expected EmailVerifiedAt to be nil before verification")
	}

	token, _ := svc.CreateEmailVerificationToken(ctx, u.ID)

	err := svc.VerifyEmail(ctx, token)
	if err != nil {
		t.Fatalf("VerifyEmail() error = %v", err)
	}

	// User should now be verified
	fetched, _ = userRepo.GetByID(ctx, u.ID)
	if fetched.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set after verification")
	}
}

func TestService_VerifyEmail_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	ctx := context.Background()

	err := svc.VerifyEmail(ctx, "invalid-token")
	if !errors.Is(err, ErrInvalidVerificationToken) {
		t.Errorf("VerifyEmail() error = %v, want %v", err, ErrInvalidVerificationToken)
	}
}

func TestService_VerifyEmail_ExpiredToken(t *testing.T) {
	svc, _, _, mockVerifications := newTestService(t)
	ctx := context.Background()

	u, _ := svc.Register(ctx, RegisterInput{
		Email:       "verify@example.com",
		Password:    "password123",
		DisplayName: "Verify User",
	})

	// Manually create an expired verification token
	expiredToken := "expired-verification-token-1234567890"
	mockVerifications.Verifications[expiredToken] = &EmailVerification{
		UserID:    u.ID,
		Token:     expiredToken,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	err := svc.VerifyEmail(ctx, expiredToken)
	if !errors.Is(err, ErrInvalidVerificationToken) {
		t.Errorf("VerifyEmail() error = %v, want %v", err, ErrInvalidVerificationToken)
	}
}
