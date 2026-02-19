package server

import (
	"net/http"
	"testing"
	"time"
)

func TestNew_ModeOff(t *testing.T) {
	s := New("localhost", 8080, http.NewServeMux(), TLSOptions{Mode: "off"}, 30*time.Second, 60*time.Second, 120*time.Second)

	if s.httpServer.TLSConfig != nil {
		t.Fatal("expected no TLSConfig for mode off")
	}
	if s.certManager != nil {
		t.Fatal("expected no certManager for mode off")
	}
	if s.redirectServer != nil {
		t.Fatal("expected no redirectServer for mode off")
	}
	if s.TLSMode() != "off" {
		t.Fatalf("expected TLSMode 'off', got %q", s.TLSMode())
	}
}

func TestNew_ModeAuto(t *testing.T) {
	s := New("0.0.0.0", 443, http.NewServeMux(), TLSOptions{
		Mode:     "auto",
		Domain:   "example.com",
		Email:    "admin@example.com",
		CacheDir: t.TempDir(),
	}, 30*time.Second, 60*time.Second, 120*time.Second)

	if s.httpServer.TLSConfig == nil {
		t.Fatal("expected TLSConfig to be set for mode auto")
	}
	if s.httpServer.TLSConfig.GetCertificate == nil {
		t.Fatal("expected GetCertificate to be set")
	}
	if s.certManager == nil {
		t.Fatal("expected certManager to be set for mode auto")
	}
	if s.redirectServer == nil {
		t.Fatal("expected redirectServer to be set for mode auto")
	}
	if s.redirectServer.Addr != ":80" {
		t.Fatalf("expected redirect server on :80, got %s", s.redirectServer.Addr)
	}
	if s.TLSMode() != "auto" {
		t.Fatalf("expected TLSMode 'auto', got %q", s.TLSMode())
	}
}

func TestNew_ModeManual(t *testing.T) {
	s := New("localhost", 443, http.NewServeMux(), TLSOptions{
		Mode:     "manual",
		CertFile: "/path/to/cert.pem",
		KeyFile:  "/path/to/key.pem",
	}, 30*time.Second, 60*time.Second, 120*time.Second)

	if s.httpServer.TLSConfig != nil {
		t.Fatal("expected no TLSConfig for mode manual (certs loaded from file)")
	}
	if s.certManager != nil {
		t.Fatal("expected no certManager for mode manual")
	}
	if s.redirectServer != nil {
		t.Fatal("expected no redirectServer for mode manual")
	}
	if s.TLSMode() != "manual" {
		t.Fatalf("expected TLSMode 'manual', got %q", s.TLSMode())
	}
}

func TestNew_ModeEmpty(t *testing.T) {
	s := New("localhost", 8080, http.NewServeMux(), TLSOptions{}, 30*time.Second, 60*time.Second, 120*time.Second)

	if s.httpServer.TLSConfig != nil {
		t.Fatal("expected no TLSConfig for empty mode")
	}
	if s.TLSMode() != "off" {
		t.Fatalf("expected TLSMode 'off' for empty mode, got %q", s.TLSMode())
	}
}

func TestTLSMode(t *testing.T) {
	for _, mode := range []string{"off", "auto", "manual"} {
		t.Run(mode, func(t *testing.T) {
			opts := TLSOptions{Mode: mode}
			if mode == "auto" {
				opts.Domain = "example.com"
				opts.CacheDir = t.TempDir()
			}
			s := New("localhost", 8080, http.NewServeMux(), opts, 30*time.Second, 60*time.Second, 120*time.Second)
			if s.TLSMode() != mode {
				t.Fatalf("expected TLSMode %q, got %q", mode, s.TLSMode())
			}
		})
	}
}
