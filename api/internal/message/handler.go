package message

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/channel"
	"github.com/feather/api/internal/sse"
	"github.com/feather/api/internal/workspace"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo          *Repository
	channelRepo   *channel.Repository
	workspaceRepo *workspace.Repository
	hub           *sse.Hub
}

func NewHandler(repo *Repository, channelRepo *channel.Repository, workspaceRepo *workspace.Repository, hub *sse.Hub) *Handler {
	return &Handler{
		repo:          repo,
		channelRepo:   channelRepo,
		workspaceRepo: workspaceRepo,
		hub:           hub,
	}
}

type SendMessageInput struct {
	Content        string  `json:"content"`
	ThreadParentID *string `json:"thread_parent_id,omitempty"`
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	ch, err := h.channelRepo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check channel is not archived
	if ch.ArchivedAt != nil {
		writeError(w, http.StatusBadRequest, "CHANNEL_ARCHIVED", "Cannot post to archived channel")
		return
	}

	// Check channel membership
	membership, err := h.channelRepo.GetMembership(r.Context(), userID, channelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			// For public channels, check if user is workspace member
			if ch.Type == channel.TypePublic {
				_, err = h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
				if err != nil {
					writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
					return
				}
				// Auto-join public channel
				_, _ = h.channelRepo.AddMember(r.Context(), userID, channelID, nil)
			} else {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	} else if !channel.CanPost(membership.ChannelRole) {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to post in this channel")
		return
	}

	var input SendMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if strings.TrimSpace(input.Content) == "" {
		writeError(w, http.StatusBadRequest, "CONTENT_REQUIRED", "Message content is required")
		return
	}

	// Validate thread parent if provided
	if input.ThreadParentID != nil {
		parent, err := h.repo.GetByID(r.Context(), *input.ThreadParentID)
		if err != nil {
			if errors.Is(err, ErrMessageNotFound) {
				writeError(w, http.StatusBadRequest, "PARENT_NOT_FOUND", "Thread parent message not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
		if parent.ChannelID != channelID {
			writeError(w, http.StatusBadRequest, "INVALID_PARENT", "Thread parent must be in the same channel")
			return
		}
		// Can't reply to a reply
		if parent.ThreadParentID != nil {
			writeError(w, http.StatusBadRequest, "INVALID_PARENT", "Cannot reply to a thread reply")
			return
		}
	}

	msg := &Message{
		ChannelID:      channelID,
		UserID:         &userID,
		Content:        input.Content,
		ThreadParentID: input.ThreadParentID,
	}

	if err := h.repo.Create(r.Context(), msg); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Fetch message with user info for response and broadcast
	msgWithUser, err := h.repo.GetByIDWithUser(r.Context(), msg.ID)
	if err != nil {
		// Fall back to basic message if user fetch fails
		msgWithUser = &MessageWithUser{Message: *msg}
	}

	// Broadcast message via SSE
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, channelID, sse.Event{
			Type: sse.EventMessageNew,
			Data: msgWithUser,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": msgWithUser,
	})
}

type ListMessagesInput struct {
	Cursor    string `json:"cursor,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Direction string `json:"direction,omitempty"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	ch, err := h.channelRepo.GetByID(r.Context(), channelID)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check access
	_, err = h.channelRepo.GetMembership(r.Context(), userID, channelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
			// Public channels: verify workspace membership
			_, err = h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
			if err != nil {
				writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	}

	var input ListMessagesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	result, err := h.repo.List(r.Context(), channelID, ListOptions{
		Cursor:    input.Cursor,
		Limit:     input.Limit,
		Direction: input.Direction,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

type UpdateMessageInput struct {
	Content string `json:"content"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	msg, err := h.repo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Only message author can edit
	if msg.UserID == nil || *msg.UserID != userID {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You can only edit your own messages")
		return
	}

	// Can't edit deleted messages
	if msg.DeletedAt != nil {
		writeError(w, http.StatusBadRequest, "MESSAGE_DELETED", "Cannot edit deleted message")
		return
	}

	var input UpdateMessageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if strings.TrimSpace(input.Content) == "" {
		writeError(w, http.StatusBadRequest, "CONTENT_REQUIRED", "Message content is required")
		return
	}

	if err := h.repo.Update(r.Context(), messageID, input.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Get updated message with user info
	msgWithUser, _ := h.repo.GetByIDWithUser(r.Context(), messageID)

	// Get channel for workspace ID
	ch, _ := h.channelRepo.GetByID(r.Context(), msg.ChannelID)

	// Broadcast update via SSE
	if h.hub != nil && ch != nil && msgWithUser != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventMessageUpdated,
			Data: msgWithUser,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": msgWithUser,
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	msg, err := h.repo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check permissions: author or workspace admin
	ch, err := h.channelRepo.GetByID(r.Context(), msg.ChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	canDelete := msg.UserID != nil && *msg.UserID == userID

	if !canDelete {
		membership, err := h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
		if err == nil && workspace.CanManageMembers(membership.Role) {
			canDelete = true
		}
	}

	if !canDelete {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "You don't have permission to delete this message")
		return
	}

	if err := h.repo.Delete(r.Context(), messageID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Broadcast delete via SSE
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventMessageDeleted,
			Data: map[string]string{"id": messageID},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

type AddReactionInput struct {
	Emoji string `json:"emoji"`
}

func (h *Handler) AddReaction(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	msg, err := h.repo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check channel membership
	ch, err := h.channelRepo.GetByID(r.Context(), msg.ChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	_, err = h.channelRepo.GetMembership(r.Context(), userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	}

	var input AddReactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if strings.TrimSpace(input.Emoji) == "" {
		writeError(w, http.StatusBadRequest, "EMOJI_REQUIRED", "Emoji is required")
		return
	}

	reaction, err := h.repo.AddReaction(r.Context(), messageID, userID, input.Emoji)
	if err != nil {
		if errors.Is(err, ErrReactionExists) {
			writeError(w, http.StatusConflict, "REACTION_EXISTS", "You already added this reaction")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Broadcast reaction via SSE
	if h.hub != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventReactionAdded,
			Data: reaction,
		})
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"reaction": reaction,
	})
}

type RemoveReactionInput struct {
	Emoji string `json:"emoji"`
}

func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	msg, err := h.repo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	var input RemoveReactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	err = h.repo.RemoveReaction(r.Context(), messageID, userID, input.Emoji)
	if err != nil {
		if errors.Is(err, ErrReactionNotFound) {
			writeError(w, http.StatusNotFound, "REACTION_NOT_FOUND", "Reaction not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Get channel for broadcasting
	ch, _ := h.channelRepo.GetByID(r.Context(), msg.ChannelID)

	// Broadcast removal via SSE
	if h.hub != nil && ch != nil {
		h.hub.BroadcastToChannel(ch.WorkspaceID, msg.ChannelID, sse.Event{
			Type: sse.EventReactionRemoved,
			Data: map[string]string{
				"message_id": messageID,
				"user_id":    userID,
				"emoji":      input.Emoji,
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) ListThread(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "id")
	userID := auth.GetUserID(r.Context())

	msg, err := h.repo.GetByID(r.Context(), messageID)
	if err != nil {
		if errors.Is(err, ErrMessageNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Message not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	// Check channel access
	ch, err := h.channelRepo.GetByID(r.Context(), msg.ChannelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	_, err = h.channelRepo.GetMembership(r.Context(), userID, msg.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				writeError(w, http.StatusForbidden, "NOT_CHANNEL_MEMBER", "You are not a member of this channel")
				return
			}
			// Verify workspace membership for public channels
			_, err = h.workspaceRepo.GetMembership(r.Context(), userID, ch.WorkspaceID)
			if err != nil {
				writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "You are not a member of this workspace")
				return
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
			return
		}
	}

	var input ListMessagesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	result, err := h.repo.ListThread(r.Context(), messageID, ListOptions{
		Cursor: input.Cursor,
		Limit:  input.Limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
		return
	}

	writeJSON(w, http.StatusOK, result)
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
