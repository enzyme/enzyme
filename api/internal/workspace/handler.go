package workspace

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/feather/api/internal/auth"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

type CreateWorkspaceInput struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var input CreateWorkspaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if err := validateSlug(input.Slug); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SLUG", err.Error())
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "Name is required")
		return
	}

	workspace := &Workspace{
		Slug:     input.Slug,
		Name:     input.Name,
		Settings: "{}",
	}

	if err := h.repo.Create(r.Context(), workspace, userID); err != nil {
		if errors.Is(err, ErrSlugAlreadyInUse) {
			writeError(w, http.StatusConflict, "SLUG_IN_USE", "This slug is already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"workspace": workspace,
	})
}

type UpdateWorkspaceInput struct {
	Slug *string `json:"slug,omitempty"`
	Name *string `json:"name,omitempty"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check permissions
	membership, err := h.repo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if !CanManageMembers(membership.Role) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to update this workspace")
		return
	}

	workspace, err := h.repo.GetByID(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	var input UpdateWorkspaceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if input.Slug != nil {
		if err := validateSlug(*input.Slug); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_SLUG", err.Error())
			return
		}
		workspace.Slug = *input.Slug
	}
	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "Name cannot be empty")
			return
		}
		workspace.Name = *input.Name
	}

	if err := h.repo.Update(r.Context(), workspace); err != nil {
		if errors.Is(err, ErrSlugAlreadyInUse) {
			writeError(w, http.StatusConflict, "SLUG_IN_USE", "This slug is already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workspace": workspace,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check membership
	_, err := h.repo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	workspace, err := h.repo.GetByID(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Workspace not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workspace": workspace,
	})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check membership
	_, err := h.repo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	members, err := h.repo.ListMembers(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if members == nil {
		members = []MemberWithUser{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
	})
}

type RemoveMemberInput struct {
	UserID string `json:"user_id"`
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check permissions
	membership, err := h.repo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	var input RemoveMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Users can remove themselves, admins/owners can remove others
	if input.UserID != userID && !CanManageMembers(membership.Role) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to remove members")
		return
	}

	if err := h.repo.RemoveMember(r.Context(), input.UserID, workspaceID); err != nil {
		if errors.Is(err, ErrCannotRemoveOwner) {
			writeError(w, http.StatusForbidden, "CANNOT_REMOVE_OWNER", "Cannot remove the workspace owner")
			return
		}
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusNotFound, "NOT_A_MEMBER", "User is not a member")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

type UpdateRoleInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check permissions
	membership, err := h.repo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if !CanChangeRole(membership.Role) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to change roles")
		return
	}

	var input UpdateRoleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Validate role
	if input.Role != RoleAdmin && input.Role != RoleMember && input.Role != RoleGuest {
		writeError(w, http.StatusBadRequest, "INVALID_ROLE", "Invalid role")
		return
	}

	// Can't change owner role
	targetMembership, err := h.repo.GetMembership(r.Context(), input.UserID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusNotFound, "NOT_A_MEMBER", "User is not a member")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if targetMembership.Role == RoleOwner {
		writeError(w, http.StatusForbidden, "CANNOT_CHANGE_OWNER", "Cannot change owner's role")
		return
	}

	// Admins can't promote to admin
	if membership.Role == RoleAdmin && input.Role == RoleAdmin {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "Admins cannot promote to admin")
		return
	}

	if err := h.repo.UpdateMemberRole(r.Context(), input.UserID, workspaceID, input.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

type CreateInviteInput struct {
	InvitedEmail *string `json:"invited_email,omitempty"`
	Role         string  `json:"role"`
	MaxUses      *int    `json:"max_uses,omitempty"`
	ExpiresIn    *int    `json:"expires_in_hours,omitempty"`
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check permissions
	membership, err := h.repo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if !CanManageMembers(membership.Role) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to create invites")
		return
	}

	var input CreateInviteInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Validate role
	if input.Role != RoleAdmin && input.Role != RoleMember && input.Role != RoleGuest {
		input.Role = RoleMember
	}

	// Can't invite as owner
	if input.Role == RoleOwner {
		input.Role = RoleMember
	}

	invite := &Invite{
		WorkspaceID:  workspaceID,
		InvitedEmail: input.InvitedEmail,
		Role:         input.Role,
		CreatedBy:    &userID,
		MaxUses:      input.MaxUses,
	}

	if input.ExpiresIn != nil && *input.ExpiresIn > 0 {
		t := expiresAt(*input.ExpiresIn)
		invite.ExpiresAt = &t
	}

	if err := h.repo.CreateInvite(r.Context(), invite); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"invite": invite,
	})
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	userID := auth.GetUserID(r.Context())

	workspace, err := h.repo.AcceptInvite(r.Context(), code, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			writeError(w, http.StatusNotFound, "INVITE_NOT_FOUND", "Invite not found")
		case errors.Is(err, ErrInviteExpired):
			writeError(w, http.StatusGone, "INVITE_EXPIRED", "This invite has expired")
		case errors.Is(err, ErrInviteMaxUsed):
			writeError(w, http.StatusGone, "INVITE_MAX_USED", "This invite has reached its maximum uses")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workspace": workspace,
	})
}

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

func validateSlug(slug string) error {
	if !slugRegex.MatchString(slug) {
		return errors.New("slug must be 3-50 characters, lowercase letters, numbers, and hyphens only")
	}
	return nil
}

func expiresAt(hours int) time.Time {
	return time.Now().Add(time.Duration(hours) * time.Hour)
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
