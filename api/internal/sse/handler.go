package sse

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/feather/api/internal/auth"
	"github.com/feather/api/internal/workspace"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

const (
	HeartbeatInterval = 30 * time.Second
	ClientBufferSize  = 256
)

type Handler struct {
	hub           *Hub
	workspaceRepo *workspace.Repository
}

func NewHandler(hub *Hub, workspaceRepo *workspace.Repository) *Handler {
	return &Handler{
		hub:           hub,
		workspaceRepo: workspaceRepo,
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
		Send:        make(chan Event, ClientBufferSize),
		Done:        make(chan struct{}),
	}

	h.hub.Register(client)
	defer h.hub.Unregister(client)

	// Send connected event
	h.writeEvent(w, flusher, Event{
		ID:   ulid.Make().String(),
		Type: EventConnected,
		Data: map[string]string{
			"client_id": client.ID,
		},
	})

	// Send initial presence - list of currently online users
	onlineUserIDs := h.hub.GetConnectedUserIDs(workspaceID)
	h.writeEvent(w, flusher, Event{
		ID:   ulid.Make().String(),
		Type: EventPresenceInitial,
		Data: map[string]interface{}{
			"online_user_ids": onlineUserIDs,
		},
	})

	// Handle reconnection - replay missed events
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		events, err := h.hub.GetEventsSince(workspaceID, lastEventID)
		if err == nil {
			for _, event := range events {
				h.writeEvent(w, flusher, event)
			}
		}
	}

	// Start heartbeat
	heartbeat := time.NewTicker(HeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-client.Done:
			return
		case event := <-client.Send:
			h.writeEvent(w, flusher, event)
		case <-heartbeat.C:
			h.writeEvent(w, flusher, Event{
				ID:   ulid.Make().String(),
				Type: EventHeartbeat,
				Data: map[string]int64{
					"timestamp": time.Now().Unix(),
				},
			})
		}
	}
}

func (h *Handler) writeEvent(w http.ResponseWriter, flusher http.Flusher, event Event) {
	if event.ID != "" {
		_, _ = fmt.Fprintf(w, "id: %s\n", event.ID)
	}

	// Marshal the full event (including type) so the client can dispatch by type
	data, err := json.Marshal(event)
	if err == nil {
		_, _ = fmt.Fprintf(w, "data: %s\n", data)
	}

	_, _ = fmt.Fprintf(w, "\n")
	flusher.Flush()
}

type TypingInput struct {
	ChannelID string `json:"channel_id"`
}

func (h *Handler) StartTyping(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	var input TypingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	h.hub.BroadcastToChannel(workspaceID, input.ChannelID, Event{
		Type: EventTypingStart,
		Data: map[string]string{
			"user_id":    userID,
			"channel_id": input.ChannelID,
		},
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) StopTyping(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "wid")
	userID := auth.GetUserID(r.Context())

	var input TypingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	h.hub.BroadcastToChannel(workspaceID, input.ChannelID, Event{
		Type: EventTypingStop,
		Data: map[string]string{
			"user_id":    userID,
			"channel_id": input.ChannelID,
		},
	})

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
