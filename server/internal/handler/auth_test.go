package handler

import (
	"context"
	"testing"
	"time"

	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/testutil"
	"github.com/enzyme/server/internal/user"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestUserToAPI(t *testing.T) {
	now := time.Now()
	avatarURL := "https://example.com/avatar.png"

	u := &user.User{
		ID:              "user-123",
		Email:           "test@example.com",
		EmailVerifiedAt: &now,
		DisplayName:     "Test User",
		AvatarURL:       &avatarURL,
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	apiUser := userToAPI(u)

	if apiUser.Id != "user-123" {
		t.Errorf("Id = %q, want %q", apiUser.Id, "user-123")
	}
	if string(apiUser.Email) != "test@example.com" {
		t.Errorf("Email = %q, want %q", apiUser.Email, "test@example.com")
	}
	if apiUser.DisplayName != "Test User" {
		t.Errorf("DisplayName = %q, want %q", apiUser.DisplayName, "Test User")
	}
	if apiUser.AvatarUrl == nil || *apiUser.AvatarUrl != avatarURL {
		t.Errorf("AvatarUrl = %v, want %q", apiUser.AvatarUrl, avatarURL)
	}
	if apiUser.EmailVerifiedAt == nil {
		t.Error("expected EmailVerifiedAt to be set")
	}
	if apiUser.Status != "active" {
		t.Errorf("Status = %q, want %q", apiUser.Status, "active")
	}
}

func TestUserToAPI_NilOptionalFields(t *testing.T) {
	now := time.Now()

	u := &user.User{
		ID:          "user-123",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	apiUser := userToAPI(u)

	if apiUser.AvatarUrl != nil {
		t.Errorf("AvatarUrl = %v, want nil", apiUser.AvatarUrl)
	}
	if apiUser.EmailVerifiedAt != nil {
		t.Error("expected EmailVerifiedAt to be nil")
	}
}

func TestForgotPassword(t *testing.T) {
	h, db := testHandlerWithEmail(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		email string
	}{
		{"registered email returns success", "alice@example.com"},
		{"unknown email returns success", "nobody@example.com"},
	}

	// Create a test user
	testutil.CreateTestUser(t, db, "alice@example.com", "Alice")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := openapi.ForgotPasswordJSONRequestBody{
				Email: openapi_types.Email(tt.email),
			}
			resp, err := h.ForgotPassword(ctx, openapi.ForgotPasswordRequestObject{Body: &body})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			okResp, ok := resp.(openapi.ForgotPassword200JSONResponse)
			if !ok {
				t.Fatalf("expected 200 response, got %T", resp)
			}
			if !okResp.Success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestForgotPassword_EmailDisabled(t *testing.T) {
	h, _ := testHandler(t)
	ctx := context.Background()

	body := openapi.ForgotPasswordJSONRequestBody{
		Email: openapi_types.Email("test@example.com"),
	}
	resp, err := h.ForgotPassword(ctx, openapi.ForgotPasswordRequestObject{Body: &body})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ForgotPassword400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestResendVerification_EmailDisabled(t *testing.T) {
	h, db := testHandler(t)
	u := testutil.CreateTestUser(t, db, "test@example.com", "Test")
	ctx := ctxWithUser(t, h, u.ID)

	resp, err := h.ResendVerification(ctx, openapi.ResendVerificationRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := resp.(openapi.ResendVerification400JSONResponse); !ok {
		t.Fatalf("expected 400 response, got %T", resp)
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := newErrorResponse("TEST_ERROR", "Test error message")

	if resp.Error.Code != "TEST_ERROR" {
		t.Errorf("Code = %q, want %q", resp.Error.Code, "TEST_ERROR")
	}
	if resp.Error.Message != "Test error message" {
		t.Errorf("Message = %q, want %q", resp.Error.Message, "Test error message")
	}
}

func TestNewError(t *testing.T) {
	err := newError("ERROR_CODE", "Error message")

	if err.Code != "ERROR_CODE" {
		t.Errorf("Code = %q, want %q", err.Code, "ERROR_CODE")
	}
	if err.Message != "Error message" {
		t.Errorf("Message = %q, want %q", err.Message, "Error message")
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify error code constants have expected values
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ErrCodeInvalidJSON", ErrCodeInvalidJSON, "INVALID_JSON"},
		{"ErrCodeInternalError", ErrCodeInternalError, "INTERNAL_ERROR"},
		{"ErrCodeNotAuthenticated", ErrCodeNotAuthenticated, "NOT_AUTHENTICATED"},
		{"ErrCodeNotAMember", ErrCodeNotAMember, "NOT_A_MEMBER"},
		{"ErrCodeNotFound", ErrCodeNotFound, "NOT_FOUND"},
		{"ErrCodePermissionDenied", ErrCodePermissionDenied, "PERMISSION_DENIED"},
		{"ErrCodeValidationError", ErrCodeValidationError, "VALIDATION_ERROR"},
		{"ErrCodeConflict", ErrCodeConflict, "CONFLICT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}
