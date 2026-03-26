package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

var (
	ErrExpired          = errors.New("signed URL has expired")
	ErrInvalidSignature = errors.New("invalid signature")
)

// Signer creates and verifies HMAC-SHA256 signed URLs for file downloads.
type Signer struct {
	secret []byte
}

// NewSigner creates a new Signer with the given secret.
func NewSigner(secret string) *Signer {
	return &Signer{secret: []byte(secret)}
}

// Sign computes an HMAC-SHA256 signature for the given file ID, user ID, and expiry time.
func (s *Signer) Sign(fileID, userID string, expires time.Time) string {
	msg := fmt.Sprintf("%s:%s:%d", fileID, userID, expires.Unix())
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks that the signature is valid and not expired.
func (s *Signer) Verify(fileID, userID string, expiresUnix int64, sig string) error {
	if time.Now().Unix() > expiresUnix {
		return ErrExpired
	}

	expected := s.Sign(fileID, userID, time.Unix(expiresUnix, 0))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return ErrInvalidSignature
	}
	return nil
}

// SignedURL builds a full signed download URL with expiry, user ID, and signature query params.
func (s *Signer) SignedURL(baseURL, fileID, userID string, ttl time.Duration) (string, time.Time, error) {
	expires := time.Now().Add(ttl)
	sig := s.Sign(fileID, userID, expires)

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parsing base URL: %w", err)
	}
	q := u.Query()
	q.Set("expires", strconv.FormatInt(expires.Unix(), 10))
	q.Set("uid", userID)
	q.Set("sig", sig)
	u.RawQuery = q.Encode()

	return u.String(), expires, nil
}
