package handler

import (
	"context"
	"errors"

	"github.com/feather/api/internal/api"
	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/user"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Register handles user registration
func (h *Handler) Register(ctx context.Context, request api.RegisterRequestObject) (api.RegisterResponseObject, error) {
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
		return api.Register400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(newErrorResponse(code, msg)),
		}, nil
	}

	// Auto-login after registration
	h.setUserID(ctx, u.ID)

	return api.Register200JSONResponse{
		User: userToAPI(u),
	}, nil
}

// Login handles user login
func (h *Handler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
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
		return api.Login401JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(newErrorResponse(code, msg)),
		}, nil
	}

	h.setUserID(ctx, u.ID)

	return api.Login200JSONResponse{
		User: userToAPI(u),
	}, nil
}

// Logout handles user logout
func (h *Handler) Logout(ctx context.Context, request api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	if err := h.destroySession(ctx); err != nil {
		return nil, err
	}

	return api.Logout200JSONResponse{
		Success: true,
	}, nil
}

// GetMe returns the current user's information
func (h *Handler) GetMe(ctx context.Context, request api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return api.GetMe401JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	u, err := h.authService.GetCurrentUser(ctx, userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return api.GetMe401JSONResponse{
				UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
			}, nil
		}
		return nil, err
	}

	response := api.GetMe200JSONResponse{
		User: userToAPI(u),
	}

	// Include workspaces
	workspaces, err := h.workspaceRepo.GetWorkspacesForUser(GetRequest(ctx), userID)
	if err == nil && len(workspaces) > 0 {
		apiWorkspaces := make([]api.WorkspaceSummary, len(workspaces))
		for i, ws := range workspaces {
			apiWorkspaces[i] = api.WorkspaceSummary{
				Id:      ws.ID,
				Slug:    ws.Slug,
				Name:    ws.Name,
				IconUrl: ws.IconURL,
				Role:    api.WorkspaceRole(ws.Role),
			}
		}
		response.Workspaces = &apiWorkspaces
	}

	return response, nil
}

// ForgotPassword handles password reset requests
func (h *Handler) ForgotPassword(ctx context.Context, request api.ForgotPasswordRequestObject) (api.ForgotPasswordResponseObject, error) {
	// Always return success to not reveal if email exists
	_, _ = h.authService.CreatePasswordResetToken(ctx, string(request.Body.Email))

	success := true
	msg := "If the email exists, a reset link will be sent"
	return api.ForgotPassword200JSONResponse{
		Success: &success,
		Message: &msg,
	}, nil
}

// ResetPassword handles password reset with token
func (h *Handler) ResetPassword(ctx context.Context, request api.ResetPasswordRequestObject) (api.ResetPasswordResponseObject, error) {
	err := h.authService.ResetPassword(ctx, request.Body.Token, request.Body.NewPassword)
	if err != nil {
		// For reset password, we return success even on error to not leak info
		// But we should handle specific errors
		switch {
		case errors.Is(err, auth.ErrInvalidResetToken):
			// Return an error response for invalid token
			return nil, err
		case errors.Is(err, auth.ErrPasswordTooShort):
			return nil, err
		default:
			return nil, err
		}
	}

	return api.ResetPassword200JSONResponse{
		Success: true,
	}, nil
}

// userToAPI converts a user.User to api.User
func userToAPI(u *user.User) api.User {
	apiUser := api.User{
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
	return apiUser
}
