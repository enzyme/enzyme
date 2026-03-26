package signing

import (
	"strings"
	"testing"
	"time"
)

func TestSignVerifyRoundtrip(t *testing.T) {
	s := NewSigner("test-secret-key")
	fileID := "file123"
	userID := "user456"
	expires := time.Now().Add(time.Hour)

	sig := s.Sign(fileID, userID, expires)
	if err := s.Verify(fileID, userID, expires.Unix(), sig); err != nil {
		t.Fatalf("valid signature should verify: %v", err)
	}
}

func TestVerifyExpired(t *testing.T) {
	s := NewSigner("test-secret-key")
	fileID := "file123"
	userID := "user456"
	expires := time.Now().Add(-time.Hour) // already expired

	sig := s.Sign(fileID, userID, expires)
	err := s.Verify(fileID, userID, expires.Unix(), sig)
	if err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestVerifyTamperedSignature(t *testing.T) {
	s := NewSigner("test-secret-key")
	fileID := "file123"
	userID := "user456"
	expires := time.Now().Add(time.Hour)

	err := s.Verify(fileID, userID, expires.Unix(), "tampered-signature")
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyWrongUser(t *testing.T) {
	s := NewSigner("test-secret-key")
	fileID := "file123"
	expires := time.Now().Add(time.Hour)

	sig := s.Sign(fileID, "user456", expires)
	err := s.Verify(fileID, "wrong-user", expires.Unix(), sig)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for wrong user, got %v", err)
	}
}

func TestVerifyWrongFileID(t *testing.T) {
	s := NewSigner("test-secret-key")
	userID := "user456"
	expires := time.Now().Add(time.Hour)

	sig := s.Sign("file123", userID, expires)
	err := s.Verify("wrong-file", userID, expires.Unix(), sig)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for wrong file, got %v", err)
	}
}

func TestSignedURL(t *testing.T) {
	s := NewSigner("test-secret-key")
	fileID := "file123"
	userID := "user456"

	url, expiresAt, err := s.SignedURL("http://localhost:8080/api/files/file123/download", fileID, userID, time.Hour)
	if err != nil {
		t.Fatalf("SignedURL: %v", err)
	}

	if url == "" {
		t.Fatal("URL should not be empty")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expiresAt should be in the future")
	}

	// Verify the URL contains the expected query params
	if !strings.Contains(url, "expires=") || !strings.Contains(url, "uid=user456") || !strings.Contains(url, "sig=") {
		t.Fatalf("URL missing expected params: %s", url)
	}
}

func TestSignedURLInvalidBaseURL(t *testing.T) {
	s := NewSigner("test-secret-key")

	_, _, err := s.SignedURL("://invalid", "file123", "user456", time.Hour)
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}
