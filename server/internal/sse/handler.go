package sse

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/enzyme/server/internal/auth"
	"github.com/enzyme/server/internal/channel"
	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/workspace"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

type Handler struct {
	hub               *Hub
	workspaceRepo     *workspace.Repository
	channelRepo       *channel.Repository
	heartbeatInterval time.Duration
	clientBufferSize  int
}

func NewHandler(hub *Hub, workspaceRepo *workspace.Repository, channelRepo *channel.Repository, heartbeatInterval time.Duration, clientBufferSize int) *Handler {
	return &Handler{
		hub:               hub,
		workspaceRepo:     workspaceRepo,
		channelRepo:       channelRepo,
		heartbeatInterval: heartbeatInterval,
		clientBufferSize:  clientBufferSize,
	}
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			http.Error(w, "Not a member of this workspace", http.StatusForbidden)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Disable write deadline for SSE (otherwise server's WriteTimeout kills the connection)
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{}) // Zero time = no deadline

	// Create client
	client := &Client{
		ID:          ulid.Make().String(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		Send:        make(chan SerializedEvent, h.clientBufferSize),
		Done:        make(chan struct{}),
	}

	h.hub.Register(client)
	defer h.hub.Unregister(client)

	// Send connected event
	h.writeLocalEvent(w, flusher, NewConnectedEvent(openapi.ConnectedData{ClientId: client.ID}))

	// Send initial presence - list of currently online users
	onlineUserIDs := h.hub.GetConnectedUserIDs(workspaceID)
	h.writeLocalEvent(w, flusher, NewPresenceInitialEvent(openapi.PresenceInitialData{
		OnlineUserIds: onlineUserIDs,
	}))

	// Handle reconnection - replay missed events
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		channelIDs, _ := h.channelRepo.ListMemberChannelIDs(r.Context(), workspaceID, userID)
		events, err := h.hub.GetEventsSince(workspaceID, lastEventID, channelIDs)
		if err == nil {
			for _, event := range events {
				h.writeLocalEvent(w, flusher, event)
			}
		}
	}

	// Start heartbeat
	heartbeat := time.NewTicker(h.heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-client.Done:
			return
		case event := <-client.Send:
			h.writeSerializedEvent(w, event)
			// Drain any pending events before flushing (batch flush)
			h.drainAndFlush(w, flusher, client)
		case <-heartbeat.C:
			h.writeLocalEvent(w, flusher, NewHeartbeatEvent(openapi.HeartbeatData{Timestamp: time.Now().Unix()}))
		}
	}
}

// writeSerializedEvent writes a pre-formatted SSE frame to the response without flushing.
// The caller is responsible for flushing (enables batch flush).
func (h *Handler) writeSerializedEvent(w http.ResponseWriter, event SerializedEvent) {
	_, _ = w.Write(event.Frame)
}

// drainAndFlush drains pending events from the client channel and flushes once.
// Capped to avoid unbounded draining if events arrive faster than writes.
func (h *Handler) drainAndFlush(w http.ResponseWriter, flusher http.Flusher, client *Client) {
	const maxDrain = 64
	for range maxDrain {
		select {
		case event := <-client.Send:
			h.writeSerializedEvent(w, event)
		default:
			flusher.Flush()
			return
		}
	}
	flusher.Flush()
}

// writeLocalEvent serializes and writes an event generated locally (not from broadcast).
// Used for connected, heartbeat, presence_initial, and reconnection replay events.
func (h *Handler) writeLocalEvent(w http.ResponseWriter, flusher http.Flusher, event Event) {
	serialized, err := event.Serialize()
	if err != nil {
		slog.Error("failed to serialize local SSE event", "type", event.Type, "error", err)
		return
	}
	h.writeSerializedEvent(w, serialized)
	flusher.Flush()
}

type TypingInput struct {
	ChannelID string `json:"channel_id"`
}

// checkTypingAccess verifies workspace membership and channel access for typing endpoints.
// Returns the decoded input and true if access is granted; writes an error response and returns false otherwise.
func (h *Handler) checkTypingAccess(w http.ResponseWriter, r *http.Request, workspaceID, userID string) (TypingInput, bool) {
	var input TypingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return input, false
	}

	if input.ChannelID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "channel_id is required")
		return input, false
	}

	// Check workspace membership
	_, err := h.workspaceRepo.GetMembership(r.Context(), userID, workspaceID)
	if err != nil {
		if errors.Is(err, workspace.ErrNotAMember) {
			writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "Not a member of this workspace")
			return input, false
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
		return input, false
	}

	// Verify channel belongs to this workspace
	ch, err := h.channelRepo.GetByID(r.Context(), input.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrChannelNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
			return input, false
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
		return input, false
	}
	if ch.WorkspaceID != workspaceID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Channel not found")
		return input, false
	}

	// Check channel membership (public channels allow any workspace member)
	_, err = h.channelRepo.GetMembership(r.Context(), userID, input.ChannelID)
	if err != nil {
		if errors.Is(err, channel.ErrNotChannelMember) {
			if ch.Type != channel.TypePublic {
				writeError(w, http.StatusForbidden, "NOT_A_MEMBER", "Not a member of this channel")
				return input, false
			}
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal error")
			return input, false
		}
	}

	return input, true
}

func (h *Handler) StartTyping(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	input, ok := h.checkTypingAccess(w, r, workspaceID, userID)
	if !ok {
		return
	}

	h.hub.BroadcastToChannel(workspaceID, input.ChannelID, NewTypingStartEvent(openapi.TypingEventData{
		UserId:    userID,
		ChannelId: input.ChannelID,
	}))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) StopTyping(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	input, ok := h.checkTypingAccess(w, r, workspaceID, userID)
	if !ok {
		return
	}

	h.hub.BroadcastToChannel(workspaceID, input.ChannelID, NewTypingStopEvent(openapi.TypingEventData{
		UserId:    userID,
		ChannelId: input.ChannelID,
	}))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
