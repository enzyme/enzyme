package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
)

const (
	SessionKeyUserID = "user_id"
)

type SessionManager struct {
	*scs.SessionManager
}

func NewSessionManager(db *sql.DB, lifetime time.Duration, secureCookies bool) *SessionManager {
	sm := scs.New()
	sm.Store = NewSQLiteStore(db)
	sm.Lifetime = lifetime
	sm.Cookie.Secure = secureCookies
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Name = "feather_session"

	return &SessionManager{sm}
}

func (sm *SessionManager) SetUserID(r *http.Request, userID string) {
	sm.Put(r.Context(), SessionKeyUserID, userID)
}

func (sm *SessionManager) GetUserID(r *http.Request) string {
	return sm.GetString(r.Context(), SessionKeyUserID)
}

func (sm *SessionManager) ClearUserID(r *http.Request) {
	sm.Remove(r.Context(), SessionKeyUserID)
}

// SQLiteStore implements scs.Store for SQLite
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Delete(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (s *SQLiteStore) Find(token string) ([]byte, bool, error) {
	var data []byte
	var expiryStr string

	row := s.db.QueryRow("SELECT data, expiry FROM sessions WHERE token = ?", token)
	err := row.Scan(&data, &expiryStr)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return nil, false, err
	}

	if time.Now().After(expiry) {
		return nil, false, nil
	}

	return data, true, nil
}

func (s *SQLiteStore) Commit(token string, data []byte, expiry time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (token, data, expiry)
		VALUES (?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET data = excluded.data, expiry = excluded.expiry
	`, token, data, expiry.Format(time.RFC3339))
	return err
}

func (s *SQLiteStore) All() (map[string][]byte, error) {
	rows, err := s.db.Query("SELECT token, data FROM sessions WHERE expiry > ?", time.Now().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make(map[string][]byte)
	for rows.Next() {
		var token string
		var data []byte
		if err := rows.Scan(&token, &data); err != nil {
			return nil, err
		}
		sessions[token] = data
	}
	return sessions, rows.Err()
}
