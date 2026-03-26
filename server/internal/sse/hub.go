package sse

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/enzyme/server/internal/openapi"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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

	retention time.Duration

	register   chan *Client
	unregister chan *Client

	// Async event storage queue — drained by a background goroutine
	// so broadcast callers never block on DB writes.
	storeQueue chan storeRequest

	// OTel metrics (no-op when telemetry is disabled)
	connectionsActive metric.Int64UpDownCounter
	eventsBroadcast   metric.Int64Counter
}

type storeRequest struct {
	workspaceID string
	channelID   string // empty for workspace-scoped events
	event       Event
}

// Pre-computed metric attribute sets to avoid allocation per broadcast under lock.
var (
	broadcastAttrsWorkspace = metric.WithAttributes(attribute.String("scope", "workspace"))
	broadcastAttrsChannel   = metric.WithAttributes(attribute.String("scope", "channel"))
	broadcastAttrsUser      = metric.WithAttributes(attribute.String("scope", "user"))
)

func NewHub(db *sql.DB, retention time.Duration) *Hub {
	meter := otel.Meter("enzyme.sse")
	connectionsActive, err := meter.Int64UpDownCounter("sse.connections.active",
		metric.WithDescription("Number of active SSE connections"),
	)
	if err != nil {
		slog.Error("failed to create sse.connections.active metric", "error", err)
	}
	eventsBroadcast, err := meter.Int64Counter("sse.events.broadcast",
		metric.WithDescription("Total SSE events broadcast"),
	)
	if err != nil {
		slog.Error("failed to create sse.events.broadcast metric", "error", err)
	}

	return &Hub{
		workspaces:        make(map[string]map[string][]*Client),
		channelMembers:    make(map[string]map[string]bool),
		db:                db,
		retention:         retention,
		register:          make(chan *Client, 256),
		unregister:        make(chan *Client, 256),
		storeQueue:        make(chan storeRequest, 1024),
		connectionsActive: connectionsActive,
		eventsBroadcast:   eventsBroadcast,
	}
}

func (h *Hub) Run(ctx context.Context) {
	go h.runStoreLoop(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			isFirstConnection := h.addClient(client)
			if isFirstConnection {
				// User just came online - broadcast to workspace
				h.BroadcastToWorkspace(client.WorkspaceID, NewPresenceChangedEvent(openapi.PresenceData{
					UserId: client.UserID,
					Status: openapi.Online,
				}))
			}
		case client := <-h.unregister:
			isLastConnection := h.removeClient(client)
			if isLastConnection {
				// User just went offline - broadcast to workspace
				h.BroadcastToWorkspace(client.WorkspaceID, NewPresenceChangedEvent(openapi.PresenceData{
					UserId: client.UserID,
					Status: openapi.Offline,
				}))
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
	h.connectionsActive.Add(context.Background(), 1)
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
	h.connectionsActive.Add(context.Background(), -1)
	return isLast
}

func (h *Hub) BroadcastToWorkspace(workspaceID string, event Event) {
	if event.ID == "" {
		event.ID = ulid.Make().String()
	}

	h.eventsBroadcast.Add(context.Background(), 1, broadcastAttrsWorkspace)

	// Queue event storage asynchronously (no DB I/O on this goroutine)
	h.enqueueStoreEvent(workspaceID, "", event)

	h.mu.RLock()
	defer h.mu.RUnlock()

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
	if event.ID == "" {
		event.ID = ulid.Make().String()
	}

	h.eventsBroadcast.Add(context.Background(), 1, broadcastAttrsChannel)

	// Queue event storage asynchronously (no DB I/O on this goroutine)
	h.enqueueStoreEvent(workspaceID, channelID, event)

	// Resolve channel members before taking the broadcast lock.
	// getChannelMembers manages its own locking internally.
	members := h.getChannelMembers(channelID)

	h.mu.RLock()
	defer h.mu.RUnlock()

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

	h.eventsBroadcast.Add(context.Background(), 1, broadcastAttrsUser)

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
	// Fast path: check cache under RLock.
	h.mu.RLock()
	if members, ok := h.channelMembers[channelID]; ok {
		h.mu.RUnlock()
		return members
	}
	h.mu.RUnlock()

	// Slow path: query database without holding any lock.
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
			if err := rows.Err(); err != nil {
				slog.Error("error iterating channel members", "channel_id", channelID, "error", err)
			}
		}
	}

	// Cache the result for future lookups.
	h.mu.Lock()
	// Re-check: another goroutine may have populated the cache while we queried.
	if _, ok := h.channelMembers[channelID]; !ok {
		h.channelMembers[channelID] = members
	} else {
		// Use the already-cached version (it may be more up-to-date).
		members = h.channelMembers[channelID]
	}
	h.mu.Unlock()

	return members
}

// enqueueStoreEvent sends an event to the background store goroutine.
// Non-blocking: if the queue is full the event is dropped with a warning.
// channelID should be set for channel-scoped events, empty for workspace-scoped.
func (h *Hub) enqueueStoreEvent(workspaceID, channelID string, event Event) {
	if h.db == nil {
		return
	}
	select {
	case h.storeQueue <- storeRequest{workspaceID: workspaceID, channelID: channelID, event: event}:
	default:
		slog.Error("sse store queue full, dropping event", "event_id", event.ID)
	}
}

// runStoreLoop drains the storeQueue and persists events to the database.
// Runs as a background goroutine started by Hub.Run.
func (h *Hub) runStoreLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Drain remaining queued events before exiting
			for {
				select {
				case req := <-h.storeQueue:
					h.storeEvent(req.workspaceID, req.channelID, req.event)
				default:
					return
				}
			}
		case req := <-h.storeQueue:
			h.storeEvent(req.workspaceID, req.channelID, req.event)
		}
	}
}

func (h *Hub) storeEvent(workspaceID, channelID string, event Event) {
	if h.db == nil {
		return
	}

	data, err := json.Marshal(event.Data)
	if err != nil {
		slog.Error("failed to marshal SSE event data", "event_id", event.ID, "error", err)
		return
	}

	var chID *string
	if channelID != "" {
		chID = &channelID
	}

	now := time.Now().UTC()
	if _, err := h.db.Exec(`
		INSERT INTO workspace_events (id, workspace_id, event_type, payload, channel_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID, workspaceID, event.Type, string(data), chID, now.Format(time.RFC3339)); err != nil {
		slog.Error("failed to store SSE event", "event_id", event.ID, "error", err)
	}
}

// CleanupOldEvents deletes SSE events older than the retention period.
func (h *Hub) CleanupOldEvents(ctx context.Context) error {
	if h.db == nil || h.retention <= 0 {
		return nil
	}

	cutoff := time.Now().UTC().Add(-h.retention).Format(time.RFC3339)
	result, err := h.db.ExecContext(ctx, `DELETE FROM workspace_events WHERE created_at < ?`, cutoff)
	if err != nil {
		return err
	}

	if deleted, err := result.RowsAffected(); err == nil && deleted > 0 {
		slog.Debug("sse event cleanup complete", "deleted", deleted)
	}
	return nil
}

func (h *Hub) GetEventsSince(workspaceID, lastEventID string, memberChannelIDs []string) ([]Event, error) {
	if h.db == nil {
		return nil, nil
	}

	// Build query that filters events by channel membership.
	// Workspace-scoped events (NULL channel_id) are always included.
	query := `
		SELECT id, event_type, payload, created_at
		FROM workspace_events
		WHERE workspace_id = ? AND id > ?
		  AND (channel_id IS NULL`
	args := []any{workspaceID, lastEventID}

	if len(memberChannelIDs) > 0 {
		query += ` OR channel_id IN (`
		for i, id := range memberChannelIDs {
			if i > 0 {
				query += `, `
			}
			query += `?`
			args = append(args, id)
		}
		query += `)`
	}

	query += `)
		ORDER BY id ASC
		LIMIT 100`

	rows, err := h.db.Query(query, args...)
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

// DisconnectUserClients forcefully disconnects all SSE clients for a user in a workspace.
// Used when a user is banned to immediately terminate their connections.
func (h *Hub) DisconnectUserClients(workspaceID, userID string) {
	h.mu.RLock()
	var clientsToClose []*Client
	if workspace, ok := h.workspaces[workspaceID]; ok {
		if clients, ok := workspace[userID]; ok {
			clientsToClose = make([]*Client, len(clients))
			copy(clientsToClose, clients)
		}
	}
	h.mu.RUnlock()

	// Close Done channels outside the lock to trigger disconnect
	for _, client := range clientsToClose {
		select {
		case <-client.Done:
			// Already closed
		default:
			close(client.Done)
		}
	}
}
