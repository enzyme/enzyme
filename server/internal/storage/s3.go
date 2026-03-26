package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/enzyme/server/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 implements Storage using an S3-compatible object store.
type S3 struct {
	client *minio.Client
	bucket string
}

// NewS3 creates a new S3 storage backend. Call CheckConnectivity after
// creation to verify the bucket exists and credentials are valid.
func NewS3(cfg config.S3Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
		BucketLookup: func() minio.BucketLookupType {
			if cfg.PathStyle {
				return minio.BucketLookupPath
			}
			return minio.BucketLookupAuto
		}(),
	})
	if err != nil {
		return nil, fmt.Errorf("creating S3 client: %w", err)
	}
	return &S3{client: client, bucket: cfg.Bucket}, nil
}

// CheckConnectivity verifies the bucket exists and credentials work.
func (s *S3) CheckConnectivity(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("checking S3 bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("S3 bucket %q does not exist", s.bucket)
	}
	return nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{ContentType: contentType}
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, opts)
	if err != nil {
		return fmt.Errorf("putting object %q: %w", key, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting object %q: %w", key, err)
	}
	// Stat to surface 404s immediately — minio returns a non-nil object even
	// for missing keys; the error only appears on Read/Stat.
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("getting object %q: %w", key, err)
	}
	return obj, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting object %q: %w", key, err)
	}
	return nil
}

func (s *S3) Serve(w http.ResponseWriter, r *http.Request, key string) {
	url, err := s.SignedURL(r.Context(), key, time.Hour)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *S3) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("presigning %q: %w", key, err)
	}
	return u.String(), nil
}
