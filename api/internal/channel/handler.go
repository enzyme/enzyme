package channel

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/workspace"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo          *Repository
	workspaceRepo *workspace.Repository
}

func NewHandler(repo *Repository, workspaceRepo *workspace.Repository) *Handler {
	return &Handler{
		repo:          repo,
		workspaceRepo: workspaceRepo,
	}
}

type CreateChannelInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Type        string  `json:"type"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check workspace membership and permissions
	membership, err := h.workspaceRepo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if !workspace.CanCreateChannels(membership.Role) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to create channels")
		return
	}

	var input CreateChannelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if strings.TrimSpace(input.Name) == "" {
		writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "Channel name is required")
		return
	}

	// Validate type
	if input.Type != TypePublic && input.Type != TypePrivate {
		input.Type = TypePublic
	}

	channel := &Channel{
		WorkspaceID: workspaceID,
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
	}

	if err := h.repo.Create(r.Context(), channel, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"channel": channel,
	})
}

type CreateDMInput struct {
	UserIDs []string `json:"user_ids"`
}

func (h *Handler) CreateDM(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	var input CreateDMInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Always include current user
	userIDs := append(input.UserIDs, userID)
	uniqueIDs := make(map[string]bool)
	var deduped []string
	for _, id := range userIDs {
		if !uniqueIDs[id] {
			uniqueIDs[id] = true
			deduped = append(deduped, id)
		}
	}

	if len(deduped) < 2 {
		writeError(w, http.StatusBadRequest, "INVALID_PARTICIPANTS", "DM requires at least 2 participants")
		return
	}

	channel, err := h.repo.CreateDM(r.Context(), workspaceID, deduped)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"channel": channel,
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	channels, err := h.repo.ListForWorkspace(r.Context(), workspaceID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Ensure we return an empty array, not null
	if channels == nil {
		channels = []ChannelWithMembership{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
	})
}

type UpdateChannelInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	channel, err := h.repo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(r.Context(), userID, channel.WorkspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check channel membership and role
	channelMembership, err := h.repo.GetMembership(r.Context(), userID, channelID)
	if err != nil && !errors.Is(err, ErrNotChannelMember) {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Workspace admins or channel admins can update
	canUpdate := workspace.CanManageMembers(membership.Role) || (channelMembership != nil && CanManageChannel(channelMembership.ChannelRole))
	if !canUpdate {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to update this channel")
		return
	}

	var input UpdateChannelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if input.Name != nil {
		if strings.TrimSpace(*input.Name) == "" {
			writeError(w, http.StatusBadRequest, "NAME_REQUIRED", "Channel name cannot be empty")
			return
		}
		channel.Name = *input.Name
	}
	if input.Description != nil {
		channel.Description = input.Description
	}

	if err := h.repo.Update(r.Context(), channel); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channel": channel,
	})
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	channel, err := h.repo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Can't archive DMs
	if channel.Type == TypeDM || channel.Type == TypeGroupDM {
		writeError(w, http.StatusBadRequest, "CANNOT_ARCHIVE_DM", "Cannot archive DM channels")
		return
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(r.Context(), userID, channel.WorkspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if !workspace.CanManageMembers(membership.Role) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to archive channels")
		return
	}

	if err := h.repo.Archive(r.Context(), channelID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

type AddMemberInput struct {
	UserID string  `json:"user_id"`
	Role   *string `json:"role,omitempty"`
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	channel, err := h.repo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check workspace membership
	membership, err := h.workspaceRepo.GetMembership(r.Context(), userID, channel.WorkspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check permissions - workspace admins or channel members can add
	channelMembership, _ := h.repo.GetMembership(r.Context(), userID, channelID)
	canAdd := workspace.CanManageMembers(membership.Role) || channelMembership != nil
	if !canAdd {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to add members")
		return
	}

	var input AddMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Verify target user is workspace member
	_, err = h.workspaceRepo.GetMembership(r.Context(), input.UserID, channel.WorkspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusBadRequest, "NOT_WORKSPACE_MEMBER", "User is not a member of the workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	_, err = h.repo.AddMember(r.Context(), input.UserID, channelID, input.Role)
	if err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			writeError(w, http.StatusConflict, "ALREADY_MEMBER", "User is already a member")
			return
		}
		if errors.Is(err, ErrChannelArchived) {
			writeError(w, http.StatusBadRequest, "CHANNEL_ARCHIVED", "Cannot add members to archived channel")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	channel, err := h.repo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Only public channels can be joined without invite
	if channel.Type != TypePublic {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "Cannot join private channels without an invite")
		return
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(r.Context(), userID, channel.WorkspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	_, err = h.repo.AddMember(r.Context(), userID, channelID, nil)
	if err != nil {
		if errors.Is(err, ErrAlreadyMember) {
			writeError(w, http.StatusConflict, "ALREADY_MEMBER", "You are already a member")
			return
		}
		if errors.Is(err, ErrChannelArchived) {
			writeError(w, http.StatusBadRequest, "CHANNEL_ARCHIVED", "Cannot join archived channel")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	err := h.repo.RemoveMember(r.Context(), userID, channelID)
	if err != nil {
		if errors.Is(err, ErrNotChannelMember) {
			writeError(w, http.StatusNotFound, "NOT_A_MEMBER", "You are not a member of this channel")
			return
		}
		if errors.Is(err, ErrCannotLeaveChannel) {
			writeError(w, http.StatusBadRequest, "CANNOT_LEAVE", "Cannot leave DM channels")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	channel, err := h.repo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check workspace membership
	_, err = h.workspaceRepo.GetMembership(r.Context(), userID, channel.WorkspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// For private channels, must be a member to see members
	if channel.Type == TypePrivate {
		_, err = h.repo.GetMembership(r.Context(), userID, channelID)
		if err != nil {
			if errors.Is(err, ErrNotChannelMember) {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	}

	members, err := h.repo.ListMembers(r.Context(), channelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	if members == nil {
		members = []MemberInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"members": members,
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
