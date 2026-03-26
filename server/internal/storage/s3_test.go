package storage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/enzyme/server/internal/config"
)

// TestS3_ImplementsStorage is a compile-time check that S3 satisfies the Storage interface.
func TestS3_ImplementsStorage(t *testing.T) {
	var _ Storage = (*S3)(nil)
}

func TestS3_Serve_Redirects(t *testing.T) {
	// Create a fake S3 server that handles presign requests
	fakeS3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeS3.Close()

	// Strip the scheme for minio endpoint
	endpoint := fakeS3.Listener.Addr().String()

	s3, err := NewS3(config.S3Config{
		Endpoint:  endpoint,
		Bucket:    "test-bucket",
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		Region:    "us-east-1",
		PathStyle: true,
		UseSSL:    false,
	})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-key.txt", nil)
	s3.Serve(rr, req, "test-key.txt")

	if rr.Code != http.StatusFound {
		t.Fatalf("Serve status = %d, want %d (302 redirect)", rr.Code, http.StatusFound)
	}

	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("Serve did not set Location header")
	}
}
