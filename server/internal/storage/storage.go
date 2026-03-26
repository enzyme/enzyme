package storage

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Storage abstracts file storage operations for uploads, avatars, icons, and emoji.
type Storage interface {
	// Put stores data under the given key.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Get returns a reader for the stored object.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the object at key. Not-found is not an error.
	Delete(ctx context.Context, key string) error

	// Serve writes the object to the HTTP response. For local storage this
	// uses http.ServeFile; for S3 it issues a 302 redirect to a pre-signed URL.
	Serve(w http.ResponseWriter, r *http.Request, key string)

	// SignedURL returns a pre-signed download URL valid for ttl.
	// Local storage returns ("", nil) so callers fall back to HMAC-signed server URLs.
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}
