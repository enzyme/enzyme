package handler

import (
	"context"
	"errors"
	"time"

	"github.com/enzyme/api/internal/auth"
	"github.com/enzyme/api/internal/gravatar"
	"github.com/enzyme/api/internal/openapi"
	"github.com/enzyme/api/internal/user"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Register handles user registration
func (h *Handler) Register(ctx context.Context, request openapi.RegisterRequestObject) (openapi.RegisterResponseObject, error) {
	input := auth.RegisterInput{
		Email:       string(request.Body.Email),
		Password:    request.Body.Password,
		DisplayName: request.Body.DisplayName,
	}

	u, err := h.authService.Register(ctx, input)
	if err != nil {
		var code, msg string
		switch {
		case errors.Is(err, user.ErrEmailAlreadyInUse):
			code, msg = "EMAIL_IN_USE", "Email is already registered"
		case errors.Is(err, auth.ErrPasswordTooShort):
			code, msg = "PASSWORD_TOO_SHORT", "Password must be at least 8 characters"
		case errors.Is(err, auth.ErrDisplayNameRequired):
			code, msg = "DISPLAY_NAME_REQUIRED", "Display name is required"
		case errors.Is(err, auth.ErrInvalidEmail):
			code, msg = "INVALID_EMAIL", "Invalid email address"
		default:
			code, msg = ErrCodeInternalError, "An error occurred"
		}
		return openapi.Register400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse(code, msg)),
		}, nil
	}

	// Create session token
	token, err := h.sessionStore.Create(u.ID)
	if err != nil {
		return nil, err
	}

	return openapi.Register200JSONResponse{
		User:  userToAPI(u),
		Token: token,
	}, nil
}

// Login handles user login
func (h *Handler) Login(ctx context.Context, request openapi.LoginRequestObject) (openapi.LoginResponseObject, error) {
	input := auth.LoginInput{
		Email:    string(request.Body.Email),
		Password: request.Body.Password,
	}

	u, err := h.authService.Login(ctx, input)
	if err != nil {
		var code, msg string
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			code, msg = "INVALID_CREDENTIALS", "Invalid email or password"
		case errors.Is(err, auth.ErrUserDeactivated):
			code, msg = "USER_DEACTIVATED", "Account is deactivated"
		default:
			code, msg = ErrCodeInternalError, "An error occurred"
		}
		return openapi.Login401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(code, msg)),
		}, nil
	}

	// Create session token
	token, err := h.sessionStore.Create(u.ID)
	if err != nil {
		return nil, err
	}

	return openapi.Login200JSONResponse{
		User:  userToAPI(u),
		Token: token,
	}, nil
}

// Logout handles user logout
func (h *Handler) Logout(ctx context.Context, request openapi.LogoutRequestObject) (openapi.LogoutResponseObject, error) {
	token := auth.GetToken(ctx)
	if token != "" {
		if err := h.sessionStore.Delete(token); err != nil {
			return nil, err
		}
	}

	return openapi.Logout200JSONResponse{
		Success: true,
	}, nil
}

// GetMe returns the current user's information
func (h *Handler) GetMe(ctx context.Context, request openapi.GetMeRequestObject) (openapi.GetMeResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.GetMe401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	u, err := h.authService.GetCurrentUser(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return openapi.GetMe401JSONResponse{
				UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
			}, nil
		}
		return nil, err
	}

	response := openapi.GetMe200JSONResponse{
		User: userToAPI(u),
	}

	// Include workspaces
	workspaces, err := h.workspaceRepo.GetWorkspacesForUser(GetRequest(ctx), userID)
	if err == nil && len(workspaces) > 0 {
		apiWorkspaces := make([]openapi.WorkspaceSummary, len(workspaces))
		for i, ws := range workspaces {
			apiWorkspaces[i] = openapi.WorkspaceSummary{
				Id:      ws.ID,
				Name:    ws.Name,
				IconUrl: ws.IconURL,
				Role:    openapi.WorkspaceRole(ws.Role),
			}
			// Check for active ban
			if ban, err := h.moderationRepo.GetActiveBan(ctx, ws.ID, userID); err == nil && ban != nil {
				apiWorkspaces[i].Ban = &struct {
					ExpiresAt *time.Time `json:"expires_at,omitempty"`
					Reason    *string    `json:"reason,omitempty"`
				}{
					ExpiresAt: ban.ExpiresAt,
				}
			}
		}
		response.Workspaces = &apiWorkspaces
	}

	return response, nil
}

// ForgotPassword handles password reset requests
func (h *Handler) ForgotPassword(ctx context.Context, request openapi.ForgotPasswordRequestObject) (openapi.ForgotPasswordResponseObject, error) {
	// Always return success to not reveal if email exists
	_, _ = h.authService.CreatePasswordResetToken(ctx, string(request.Body.Email))

	success := true
	msg := "If the email exists, a reset link will be sent"
	return openapi.ForgotPassword200JSONResponse{
		Success: &success,
		Message: &msg,
	}, nil
}

// ResetPassword handles password reset with token
func (h *Handler) ResetPassword(ctx context.Context, request openapi.ResetPasswordRequestObject) (openapi.ResetPasswordResponseObject, error) {
	err := h.authService.ResetPassword(ctx, request.Body.Token, request.Body.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidResetToken):
			return openapi.ResetPassword400JSONResponse{
				BadRequestJSONResponse: badRequestResponse("INVALID_RESET_TOKEN", "Invalid or expired reset token"),
			}, nil
		case errors.Is(err, auth.ErrPasswordTooShort):
			return openapi.ResetPassword400JSONResponse{
				BadRequestJSONResponse: badRequestResponse("PASSWORD_TOO_SHORT", "Password must be at least 8 characters"),
			}, nil
		default:
			return nil, err
		}
	}

	return openapi.ResetPassword200JSONResponse{
		Success: true,
	}, nil
}

// userToAPI converts a user.User to openapi.User
func userToAPI(u *user.User) openapi.User {
	apiUser := openapi.User{
		Id:          u.ID,
		Email:       openapi_types.Email(u.Email),
		DisplayName: u.DisplayName,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
	if u.EmailVerifiedAt != nil {
		apiUser.EmailVerifiedAt = u.EmailVerifiedAt
	}
	if u.AvatarURL != nil {
		apiUser.AvatarUrl = u.AvatarURL
	}
	if g := gravatar.URL(u.Email); g != "" {
		apiUser.GravatarUrl = &g
	}
	return apiUser
}
