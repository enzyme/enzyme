package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionStore struct {
	db       *sql.DB
	lifetime time.Duration
}

func NewSessionStore(db *sql.DB, lifetime time.Duration) *SessionStore {
	return &SessionStore{db: db, lifetime: lifetime}
}

// Create inserts a new session and returns the plaintext token.
// Only the SHA-256 hash is stored in the database.
func (s *SessionStore) Create(userID string) (string, error) {
	token := generateSessionToken()
	expiry := time.Now().Add(s.lifetime).UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id, expiry) VALUES (?, ?, ?)",
		hashToken(token), userID, expiry,
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

// Validate looks up a session by its hashed token and returns the user ID if valid.
func (s *SessionStore) Validate(token string) (string, error) {
	hashed := hashToken(token)
	var userID, expiryStr string
	err := s.db.QueryRow(
		"SELECT user_id, expiry FROM sessions WHERE token = ?", hashed,
	).Scan(&userID, &expiryStr)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", err
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return "", err
	}
	if time.Now().After(expiry) {
		// Clean up expired session
		_, _ = s.db.Exec("DELETE FROM sessions WHERE token = ?", hashed)
		return "", ErrSessionNotFound
	}

	return userID, nil
}

// Delete removes a session by its hashed token.
func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", hashToken(token))
	return err
}

// DeleteExpired removes all expired sessions.
func (s *SessionStore) DeleteExpired() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expiry < ?", time.Now().UTC().Format(time.RFC3339))
	return err
}

func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashToken returns the hex-encoded SHA-256 hash of a plaintext token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
