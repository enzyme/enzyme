package presence

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/enzyme/server/internal/openapi"
	"github.com/enzyme/server/internal/sse"
	"github.com/oklog/ulid/v2"
)

const (
	StatusOnline  = "online"
	StatusOffline = "offline"

	OfflineTimeout = 30 * time.Second
)

type UserPresence struct {
	UserID      string
	WorkspaceID string
	Status      string
	LastSeenAt  time.Time
}

type Manager struct {
	mu sync.RWMutex

	// workspaceID -> userID -> presence
	presence map[string]map[string]*UserPresence

	db  *sql.DB
	hub *sse.Hub
}

func NewManager(db *sql.DB, hub *sse.Hub) *Manager {
	return &Manager{
		presence: make(map[string]map[string]*UserPresence),
		db:       db,
		hub:      hub,
	}
}

// Init loads persisted presence state from the database. Call before scheduling CheckPresence.
func (m *Manager) Init() {
	m.loadFromDB()
}

// CheckPresence marks users as offline if they've been disconnected beyond the timeout.
func (m *Manager) CheckPresence(ctx context.Context) error {
	m.checkPresence(ctx)
	return nil
}

func (m *Manager) loadFromDB() {
	if m.db == nil {
		return
	}

	rows, err := m.db.Query(`
		SELECT user_id, workspace_id, status, last_seen_at
		FROM user_presence
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	for rows.Next() {
		var p UserPresence
		var lastSeenAt string
		if err := rows.Scan(&p.UserID, &p.WorkspaceID, &p.Status, &lastSeenAt); err != nil {
			continue
		}
		p.LastSeenAt, _ = time.Parse(time.RFC3339, lastSeenAt)

		if m.presence[p.WorkspaceID] == nil {
			m.presence[p.WorkspaceID] = make(map[string]*UserPresence)
		}
		m.presence[p.WorkspaceID][p.UserID] = &p
	}
	if err := rows.Err(); err != nil {
		slog.Error("error iterating presence rows", "error", err)
	}
}

func (m *Manager) SetOnline(workspaceID, userID string) {
	now := time.Now().UTC()
	var shouldBroadcast bool

	m.mu.Lock()
	if m.presence[workspaceID] == nil {
		m.presence[workspaceID] = make(map[string]*UserPresence)
	}

	prev := m.presence[workspaceID][userID]
	prevStatus := StatusOffline
	if prev != nil {
		prevStatus = prev.Status
	}

	m.presence[workspaceID][userID] = &UserPresence{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Status:      StatusOnline,
		LastSeenAt:  now,
	}
	shouldBroadcast = prevStatus != StatusOnline
	m.mu.Unlock()

	m.persistPresence(context.Background(), workspaceID, userID, StatusOnline, now)

	if shouldBroadcast {
		m.broadcastPresenceChange(workspaceID, userID, openapi.Online)
	}
}

func (m *Manager) SetOffline(workspaceID, userID string) {
	now := time.Now().UTC()

	m.mu.Lock()
	if m.presence[workspaceID] == nil {
		m.mu.Unlock()
		return
	}

	prev := m.presence[workspaceID][userID]
	if prev == nil || prev.Status == StatusOffline {
		m.mu.Unlock()
		return
	}

	m.presence[workspaceID][userID] = &UserPresence{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Status:      StatusOffline,
		LastSeenAt:  now,
	}
	m.mu.Unlock()

	m.persistPresence(context.Background(), workspaceID, userID, StatusOffline, now)
	m.broadcastPresenceChange(workspaceID, userID, openapi.Offline)
}

func (m *Manager) SetStatus(workspaceID, userID, status string) {
	if status != StatusOnline && status != StatusOffline {
		return
	}

	now := time.Now().UTC()
	var shouldBroadcast bool

	m.mu.Lock()
	if m.presence[workspaceID] == nil {
		m.presence[workspaceID] = make(map[string]*UserPresence)
	}

	prev := m.presence[workspaceID][userID]
	prevStatus := StatusOffline
	if prev != nil {
		prevStatus = prev.Status
	}

	m.presence[workspaceID][userID] = &UserPresence{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Status:      status,
		LastSeenAt:  now,
	}
	shouldBroadcast = prevStatus != status
	m.mu.Unlock()

	m.persistPresence(context.Background(), workspaceID, userID, status, now)

	if shouldBroadcast {
		m.broadcastPresenceChange(workspaceID, userID, openapi.PresenceStatus(status))
	}
}

func (m *Manager) GetPresence(workspaceID, userID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if workspace, ok := m.presence[workspaceID]; ok {
		if p, ok := workspace[userID]; ok {
			return p.Status
		}
	}
	return StatusOffline
}

func (m *Manager) GetWorkspacePresence(workspaceID string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	if workspace, ok := m.presence[workspaceID]; ok {
		for userID, p := range workspace {
			result[userID] = p.Status
		}
	}
	return result
}

type presenceChange struct {
	workspaceID string
	userID      string
}

func (m *Manager) checkPresence(ctx context.Context) {
	now := time.Now().UTC()

	// Snapshot candidates under read lock — no calls into hub while holding mu.
	type candidate struct {
		workspaceID, userID string
		lastSeenAt          time.Time
	}
	var candidates []candidate

	m.mu.RLock()
	for workspaceID, workspace := range m.presence {
		for userID, p := range workspace {
			if p.Status != StatusOffline {
				candidates = append(candidates, candidate{workspaceID, userID, p.LastSeenAt})
			}
		}
	}
	m.mu.RUnlock()

	// Check connectivity without holding any presence lock.
	var offlineChanges []presenceChange
	for _, c := range candidates {
		if now.Sub(c.lastSeenAt) <= OfflineTimeout {
			continue
		}
		if m.hub != nil && m.hub.IsUserConnected(c.workspaceID, c.userID) {
			continue
		}
		offlineChanges = append(offlineChanges, presenceChange{c.workspaceID, c.userID})
	}

	// Apply status changes under write lock.
	m.mu.Lock()
	for _, c := range offlineChanges {
		if p, ok := m.presence[c.workspaceID][c.userID]; ok && p.Status != StatusOffline {
			p.Status = StatusOffline
		}
	}
	m.mu.Unlock()

	for _, c := range offlineChanges {
		m.persistPresence(ctx, c.workspaceID, c.userID, StatusOffline, now)
		m.broadcastPresenceChange(c.workspaceID, c.userID, openapi.Offline)
	}
}

func (m *Manager) persistPresence(ctx context.Context, workspaceID, userID, status string, lastSeen time.Time) {
	if m.db == nil {
		return
	}

	id := ulid.Make().String()
	_, _ = m.db.ExecContext(ctx, `
		INSERT INTO user_presence (id, user_id, workspace_id, status, last_seen_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, workspace_id) DO UPDATE SET status = excluded.status, last_seen_at = excluded.last_seen_at
	`, id, userID, workspaceID, status, lastSeen.Format(time.RFC3339))
}

func (m *Manager) broadcastPresenceChange(workspaceID, userID string, status openapi.PresenceStatus) {
	if m.hub == nil {
		return
	}

	m.hub.BroadcastToWorkspace(workspaceID, sse.NewPresenceChangedEvent(openapi.PresenceData{
		UserId: userID,
		Status: status,
	}))
}
