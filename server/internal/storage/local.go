package storage

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Local implements Storage using the local filesystem.
type Local struct {
	basePath string
	absBase  string
}

// NewLocal creates a local filesystem storage rooted at basePath.
func NewLocal(basePath string) *Local {
	abs, err := filepath.Abs(basePath)
	if err != nil {
		abs = basePath
	}
	return &Local{basePath: basePath, absBase: abs}
}

func (l *Local) fullPath(key string) string {
	p := filepath.Join(l.basePath, filepath.FromSlash(key))
	// Defense-in-depth: verify the resolved path stays within basePath.
	absPath, err := filepath.Abs(p)
	if err != nil {
		return filepath.Join(l.basePath, "_invalid")
	}
	if !strings.HasPrefix(absPath, l.absBase+string(filepath.Separator)) && absPath != l.absBase {
		return filepath.Join(l.basePath, "_invalid")
	}
	return absPath
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p := l.fullPath(key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(p)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(p)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(p)
		return err
	}
	return nil
}

func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(l.fullPath(key))
}

func (l *Local) Delete(_ context.Context, key string) error {
	err := os.Remove(l.fullPath(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *Local) Serve(w http.ResponseWriter, r *http.Request, key string) {
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, l.fullPath(key))
}

func (l *Local) SignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
