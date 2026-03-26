package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLocal_PutGetDelete(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)
	ctx := context.Background()

	data := []byte("hello world")
	key := "a/b/test.txt"

	// Put
	if err := s.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Verify file exists on disk
	if _, err := os.Stat(filepath.Join(dir, "a", "b", "test.txt")); err != nil {
		t.Fatalf("file not on disk: %v", err)
	}

	// Get
	rc, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, data) {
		t.Fatalf("Get returned %q, want %q", got, data)
	}

	// Delete
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get after delete should fail
	if _, err := s.Get(ctx, key); err == nil {
		t.Fatal("expected error after delete")
	}

	// Delete again (not-found is not an error)
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete not-found: %v", err)
	}
}

func TestLocal_Serve(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)
	ctx := context.Background()

	data := []byte("serve me")
	key := "serve.txt"
	if err := s.Put(ctx, key, bytes.NewReader(data), int64(len(data)), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Serve existing file
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/serve.txt", nil)
	s.Serve(rr, req, key)
	if rr.Code != http.StatusOK {
		t.Fatalf("Serve status = %d, want 200", rr.Code)
	}
	if !bytes.Equal(rr.Body.Bytes(), data) {
		t.Fatalf("Serve body = %q, want %q", rr.Body.String(), data)
	}

	// Serve missing file
	rr = httptest.NewRecorder()
	s.Serve(rr, req, "missing.txt")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Serve missing status = %d, want 404", rr.Code)
	}
}

func TestLocal_SignedURL(t *testing.T) {
	s := NewLocal(t.TempDir())
	url, err := s.SignedURL(context.Background(), "anything", 0)
	if err != nil {
		t.Fatalf("SignedURL: %v", err)
	}
	if url != "" {
		t.Fatalf("expected empty URL, got %q", url)
	}
}

func TestLocal_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	s := NewLocal(dir)
	ctx := context.Background()

	// Write a file at the base path for the attacker to try to reach
	secretFile := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("writing secret file: %v", err)
	}
	defer os.Remove(secretFile)

	// Attempt directory traversal — should resolve to _invalid path inside basePath
	_, err := s.Get(ctx, "../secret.txt")
	if err == nil {
		t.Fatal("expected error when traversing outside basePath")
	}

	// Put with traversal key should not write outside basePath
	if err := s.Put(ctx, "../escape.txt", bytes.NewReader([]byte("pwned")), 5, "text/plain"); err != nil {
		// The write should either error or land inside basePath
		return
	}
	// If no error, verify the file was NOT written outside basePath
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.txt")); err == nil {
		t.Fatal("Put wrote file outside basePath via path traversal")
	}
}
