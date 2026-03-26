package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/enzyme/server/internal/auth"
	"github.com/enzyme/server/internal/gravatar"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/pushnotification"
	"github.com/enzyme/server/internal/user"
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

	// Verification email is best-effort; registration succeeds regardless of email config.
	// When email is later enabled, unverified users will see the verification banner.
	if h.emailService.IsEnabled() {
		verifyToken, err := h.authService.CreateEmailVerificationToken(ctx, u.ID)
		if err != nil {
			slog.Error("failed to create email verification token", "user_id", u.ID, "error", err)
		} else {
			go func() {
				sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := h.emailService.SendEmailVerification(sendCtx, u.Email, verifyToken); err != nil {
					slog.Error("failed to send verification email", "user_id", u.ID, "error", err)
				}
			}()
		}
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
	if !h.emailService.IsEnabled() {
		return openapi.ForgotPassword400JSONResponse{
			BadRequestJSONResponse: badRequestResponse("EMAIL_NOT_ENABLED", "Email is not configured on this server"),
		}, nil
	}

	token, err := h.authService.CreatePasswordResetToken(ctx, string(request.Body.Email))
	if err != nil {
		slog.Error("failed to create password reset token", "error", err)
	}

	if token != "" {
		emailAddr := string(request.Body.Email)
		go func() {
			sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.emailService.SendPasswordReset(sendCtx, emailAddr, token); err != nil {
				slog.Error("failed to send password reset email", "error", err)
			}
		}()
	}

	return openapi.ForgotPassword200JSONResponse{
		Success: true,
		Message: "If the email exists, a reset link will be sent",
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

// VerifyEmail handles email verification with a token
func (h *Handler) VerifyEmail(ctx context.Context, request openapi.VerifyEmailRequestObject) (openapi.VerifyEmailResponseObject, error) {
	err := h.authService.VerifyEmail(ctx, request.Body.Token)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidVerificationToken) {
			return openapi.VerifyEmail400JSONResponse{
				BadRequestJSONResponse: badRequestResponse("INVALID_VERIFICATION_TOKEN", "Invalid or expired verification token"),
			}, nil
		}
		return nil, err
	}

	return openapi.VerifyEmail200JSONResponse{Success: true}, nil
}

// ResendVerification resends the verification email to the current user
func (h *Handler) ResendVerification(ctx context.Context, request openapi.ResendVerificationRequestObject) (openapi.ResendVerificationResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.ResendVerification401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	if !h.emailService.IsEnabled() {
		return openapi.ResendVerification400JSONResponse{
			BadRequestJSONResponse: badRequestResponse("EMAIL_NOT_ENABLED", "Email is not configured on this server"),
		}, nil
	}

	u, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if u.EmailVerifiedAt != nil {
		return openapi.ResendVerification400JSONResponse{
			BadRequestJSONResponse: badRequestResponse("ALREADY_VERIFIED", "Email is already verified"),
		}, nil
	}

	token, err := h.authService.CreateEmailVerificationToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.emailService.SendEmailVerification(sendCtx, u.Email, token); err != nil {
			slog.Error("failed to send verification email", "user_id", userID, "error", err)
		}
	}()

	return openapi.ResendVerification200JSONResponse{
		Success: true,
	}, nil
}

// RegisterDeviceToken registers a device token for push notifications
func (h *Handler) RegisterDeviceToken(ctx context.Context, request openapi.RegisterDeviceTokenRequestObject) (openapi.RegisterDeviceTokenResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.RegisterDeviceToken401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	if h.pushTokenRepo == nil {
		return openapi.RegisterDeviceToken400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse("PUSH_NOT_ENABLED", "Push notifications are not enabled on this server")),
		}, nil
	}

	if len(request.Body.Token) == 0 || len(request.Body.Token) > 4096 {
		return openapi.RegisterDeviceToken400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse("INVALID_TOKEN", "Token must be between 1 and 4096 characters")),
		}, nil
	}
	if len(request.Body.DeviceId) == 0 || len(request.Body.DeviceId) > 256 {
		return openapi.RegisterDeviceToken400JSONResponse{
			BadRequestJSONResponse: openapi.BadRequestJSONResponse(newErrorResponse("INVALID_DEVICE_ID", "Device ID must be between 1 and 256 characters")),
		}, nil
	}

	token := &pushnotification.DeviceToken{
		UserID:   userID,
		Token:    request.Body.Token,
		Platform: string(request.Body.Platform),
		DeviceID: request.Body.DeviceId,
	}

	if err := h.pushTokenRepo.Upsert(ctx, token); err != nil {
		return nil, fmt.Errorf("upserting device token: %w", err)
	}

	return openapi.RegisterDeviceToken200JSONResponse{
		Id: token.ID,
	}, nil
}

// UnregisterDeviceToken removes a device token
func (h *Handler) UnregisterDeviceToken(ctx context.Context, request openapi.UnregisterDeviceTokenRequestObject) (openapi.UnregisterDeviceTokenResponseObject, error) {
	userID := h.getUserID(ctx)
	if userID == "" {
		return openapi.UnregisterDeviceToken401JSONResponse{
			UnauthorizedJSONResponse: openapi.UnauthorizedJSONResponse(newErrorResponse(ErrCodeNotAuthenticated, "Not authenticated")),
		}, nil
	}

	if h.pushTokenRepo == nil {
		return openapi.UnregisterDeviceToken404JSONResponse{
			NotFoundJSONResponse: openapi.NotFoundJSONResponse(newErrorResponse("PUSH_NOT_ENABLED", "Push notifications are not enabled on this server")),
		}, nil
	}

	err := h.pushTokenRepo.DeleteByID(ctx, userID, request.Id)
	if err != nil {
		if errors.Is(err, pushnotification.ErrTokenNotFound) {
			return openapi.UnregisterDeviceToken404JSONResponse{
				NotFoundJSONResponse: openapi.NotFoundJSONResponse(newErrorResponse("TOKEN_NOT_FOUND", "Device token not found")),
			}, nil
		}
		return nil, err
	}

	return openapi.UnregisterDeviceToken204Response{}, nil
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
