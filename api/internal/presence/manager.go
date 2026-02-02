package presence

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/feather/api/internal/sse"
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

func (m *Manager) Start(ctx context.Context) {
	// Load presence from database on startup
	m.loadFromDB()

	// Start presence check loop
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkPresence()
		}
	}
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
}

func (m *Manager) SetOnline(workspaceID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

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

	m.persistPresence(workspaceID, userID, StatusOnline, now)

	if prevStatus != StatusOnline {
		m.broadcastPresenceChange(workspaceID, userID, StatusOnline)
	}
}

func (m *Manager) SetOffline(workspaceID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	if m.presence[workspaceID] == nil {
		return
	}

	prev := m.presence[workspaceID][userID]
	if prev == nil || prev.Status == StatusOffline {
		return
	}

	m.presence[workspaceID][userID] = &UserPresence{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Status:      StatusOffline,
		LastSeenAt:  now,
	}

	m.persistPresence(workspaceID, userID, StatusOffline, now)
	m.broadcastPresenceChange(workspaceID, userID, StatusOffline)
}

func (m *Manager) SetStatus(workspaceID, userID, status string) {
	if status != StatusOnline && status != StatusOffline {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

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

	m.persistPresence(workspaceID, userID, status, now)

	if prevStatus != status {
		m.broadcastPresenceChange(workspaceID, userID, status)
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

func (m *Manager) checkPresence() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	for workspaceID, workspace := range m.presence {
		for userID, p := range workspace {
			if p.Status == StatusOffline {
				continue
			}

			// Check if user is still connected via SSE
			isConnected := m.hub != nil && m.hub.IsUserConnected(workspaceID, userID)

			if !isConnected {
				// User disconnected - mark offline after timeout
				if now.Sub(p.LastSeenAt) > OfflineTimeout {
					p.Status = StatusOffline
					m.persistPresence(workspaceID, userID, StatusOffline, now)
					m.broadcastPresenceChange(workspaceID, userID, StatusOffline)
				}
			}
		}
	}
}

func (m *Manager) persistPresence(workspaceID, userID, status string, lastSeen time.Time) {
	if m.db == nil {
		return
	}

	id := ulid.Make().String()
	_, _ = m.db.Exec(`
		INSERT INTO user_presence (id, user_id, workspace_id, status, last_seen_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, workspace_id) DO UPDATE SET status = excluded.status, last_seen_at = excluded.last_seen_at
	`, id, userID, workspaceID, status, lastSeen.Format(time.RFC3339))
}

func (m *Manager) broadcastPresenceChange(workspaceID, userID, status string) {
	if m.hub == nil {
		return
	}

	m.hub.BroadcastToWorkspace(workspaceID, sse.Event{
		Type: sse.EventPresenceChanged,
		Data: map[string]string{
			"user_id": userID,
			"status":  status,
		},
	})
}
