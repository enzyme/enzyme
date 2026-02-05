package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/feather/api/internal/user"
)

type Handler struct {
	service        *Service
	sessionManager *SessionManager
	workspaceRepo  WorkspaceRepoForAuth
}

type WorkspaceRepoForAuth interface {
	GetWorkspacesForUser(r *http.Request, userID string) ([]WorkspaceSummary, error)
}

type WorkspaceSummary struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	IconURL   *string `json:"icon_url,omitempty"`
	Role      string  `json:"role"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

func NewHandler(service *Service, sessionManager *SessionManager, workspaceRepo WorkspaceRepoForAuth) *Handler {
	return &Handler{
		service:        service,
		sessionManager: sessionManager,
		workspaceRepo:  workspaceRepo,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	u, err := h.service.Register(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrEmailAlreadyInUse):
			writeError(w, http.StatusConflict, "EMAIL_IN_USE", "Email is already registered")
		case errors.Is(err, ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters")
		case errors.Is(err, ErrDisplayNameRequired):
			writeError(w, http.StatusBadRequest, "DISPLAY_NAME_REQUIRED", "Display name is required")
		case errors.Is(err, ErrInvalidEmail):
			writeError(w, http.StatusBadRequest, "INVALID_EMAIL", "Invalid email address")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		}
		return
	}

	// Auto-login after registration
	h.sessionManager.SetUserID(r, u.ID)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user": u,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	u, err := h.service.Login(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		case errors.Is(err, ErrUserDeactivated):
			writeError(w, http.StatusForbidden, "USER_DEACTIVATED", "Account is deactivated")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		}
		return
	}

	h.sessionManager.SetUserID(r, u.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": u,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessionManager.Destroy(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := h.sessionManager.GetUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
		return
	}

	u, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	response := map[string]interface{}{
		"user": u,
	}

	// Include workspaces if available
	if h.workspaceRepo != nil {
		workspaces, err := h.workspaceRepo.GetWorkspacesForUser(r, userID)
		if err == nil {
			response["workspaces"] = workspaces
		}
	}

	writeJSON(w, http.StatusOK, response)
}

type ForgotPasswordInput struct {
	Email string `json:"email"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input ForgotPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Always return success to not reveal if email exists
	_, _ = h.service.CreatePasswordResetToken(r.Context(), input.Email)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "If the email exists, a reset link will be sent",
	})
}

type ResetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	err := h.service.ResetPassword(r.Context(), input.Token, input.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidResetToken):
			writeError(w, http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token")
		case errors.Is(err, ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
