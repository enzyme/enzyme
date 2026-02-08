package sse

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

type Client struct {
	ID          string
	UserID      string
	WorkspaceID string
	Send        chan Event
	Done        chan struct{}
}

type Hub struct {
	mu sync.RWMutex

	// workspaceID -> userID -> []*Client
	workspaces map[string]map[string][]*Client

	// channelID -> set of userIDs (for scoped broadcasts)
	channelMembers map[string]map[string]bool

	db *sql.DB

	register   chan *Client
	unregister chan *Client
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		workspaces:     make(map[string]map[string][]*Client),
		channelMembers: make(map[string]map[string]bool),
		db:             db,
		register:       make(chan *Client, 256),
		unregister:     make(chan *Client, 256),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			isFirstConnection := h.addClient(client)
			if isFirstConnection {
				// User just came online - broadcast to workspace
				h.BroadcastToWorkspace(client.WorkspaceID, Event{
					Type: EventPresenceChanged,
					Data: map[string]interface{}{
						"user_id": client.UserID,
						"status":  "online",
					},
				})
			}
		case client := <-h.unregister:
			isLastConnection := h.removeClient(client)
			if isLastConnection {
				// User just went offline - broadcast to workspace
				h.BroadcastToWorkspace(client.WorkspaceID, Event{
					Type: EventPresenceChanged,
					Data: map[string]interface{}{
						"user_id": client.UserID,
						"status":  "offline",
					},
				})
			}
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) addClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.workspaces[client.WorkspaceID] == nil {
		h.workspaces[client.WorkspaceID] = make(map[string][]*Client)
	}
	isFirst := len(h.workspaces[client.WorkspaceID][client.UserID]) == 0
	h.workspaces[client.WorkspaceID][client.UserID] = append(h.workspaces[client.WorkspaceID][client.UserID], client)
	return isFirst
}

func (h *Hub) removeClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	isLast := false
	if workspace, ok := h.workspaces[client.WorkspaceID]; ok {
		if clients, ok := workspace[client.UserID]; ok {
			for i, c := range clients {
				if c.ID == client.ID {
					workspace[client.UserID] = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			if len(workspace[client.UserID]) == 0 {
				delete(workspace, client.UserID)
				isLast = true
			}
		}
		if len(workspace) == 0 {
			delete(h.workspaces, client.WorkspaceID)
		}
	}

	close(client.Send)
	return isLast
}

func (h *Hub) BroadcastToWorkspace(workspaceID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if event.ID == "" {
		event.ID = ulid.Make().String()
	}

	// Store event for replay
	h.storeEvent(workspaceID, event)

	if workspace, ok := h.workspaces[workspaceID]; ok {
		for _, clients := range workspace {
			for _, client := range clients {
				select {
				case client.Send <- event:
				default:
					// Client buffer full, skip
				}
			}
		}
	}
}

func (h *Hub) BroadcastToChannel(workspaceID, channelID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if event.ID == "" {
		event.ID = ulid.Make().String()
	}

	// Store event for replay
	h.storeEvent(workspaceID, event)

	// Get channel members from cache or database
	members := h.getChannelMembers(channelID)

	if workspace, ok := h.workspaces[workspaceID]; ok {
		for userID, clients := range workspace {
			if members[userID] {
				for _, client := range clients {
					select {
					case client.Send <- event:
					default:
						// Client buffer full, skip
					}
				}
			}
		}
	}
}

func (h *Hub) BroadcastToUser(workspaceID, userID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if event.ID == "" {
		event.ID = ulid.Make().String()
	}

	if workspace, ok := h.workspaces[workspaceID]; ok {
		if clients, ok := workspace[userID]; ok {
			for _, client := range clients {
				select {
				case client.Send <- event:
				default:
				}
			}
		}
	}
}

func (h *Hub) UpdateChannelMembers(channelID string, userIDs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	members := make(map[string]bool)
	for _, id := range userIDs {
		members[id] = true
	}
	h.channelMembers[channelID] = members
}

func (h *Hub) AddChannelMember(channelID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channelMembers[channelID] == nil {
		h.channelMembers[channelID] = make(map[string]bool)
	}
	h.channelMembers[channelID][userID] = true
}

func (h *Hub) RemoveChannelMember(channelID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channelMembers[channelID] != nil {
		delete(h.channelMembers[channelID], userID)
	}
}

func (h *Hub) getChannelMembers(channelID string) map[string]bool {
	// Check cache first (safe under RLock)
	if members, ok := h.channelMembers[channelID]; ok {
		return members
	}

	// Load from database if not cached
	// Note: We don't cache the result here because we may only hold RLock.
	// Caching happens via AddChannelMember/UpdateChannelMembers when membership changes.
	members := make(map[string]bool)
	if h.db != nil {
		rows, err := h.db.Query(`
			SELECT user_id FROM channel_memberships WHERE channel_id = ?
		`, channelID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var userID string
				if rows.Scan(&userID) == nil {
					members[userID] = true
				}
			}
		}
	}

	return members
}

func (h *Hub) storeEvent(workspaceID string, event Event) {
	if h.db == nil {
		return
	}

	data, err := json.Marshal(event.Data)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	_, _ = h.db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, event.ID, workspaceID, event.Type, string(data), now.Format(time.RFC3339))
}

func (h *Hub) GetEventsSince(workspaceID, lastEventID string) ([]Event, error) {
	if h.db == nil {
		return nil, nil
	}

	rows, err := h.db.Query(`
		SELECT id, event_type, payload, created_at
		FROM workspace_events
		WHERE workspace_id = ? AND id > ?
		ORDER BY id ASC
		LIMIT 100
	`, workspaceID, lastEventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var id, eventType, payload, createdAt string
		if err := rows.Scan(&id, &eventType, &payload, &createdAt); err != nil {
			continue
		}

		var data interface{}
		if err := json.Unmarshal([]byte(payload), &data); err != nil {
			// TODO: log corrupt event payload
			continue
		}

		events = append(events, Event{
			ID:   id,
			Type: eventType,
			Data: data,
		})
	}

	return events, rows.Err()
}

func (h *Hub) GetConnectedUserIDs(workspaceID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := []string{}
	if workspace, ok := h.workspaces[workspaceID]; ok {
		for userID := range workspace {
			userIDs = append(userIDs, userID)
		}
	}
	return userIDs
}

func (h *Hub) IsUserConnected(workspaceID, userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if workspace, ok := h.workspaces[workspaceID]; ok {
		_, connected := workspace[userID]
		return connected
	}
	return false
}

// IsUserOnline is an alias for IsUserConnected
func (h *Hub) IsUserOnline(workspaceID, userID string) bool {
	return h.IsUserConnected(workspaceID, userID)
}
