package server

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type TLSOptions struct {
	Mode     string // "off", "auto", "manual"
	CertFile string // manual mode
	KeyFile  string // manual mode
	Domain   string // auto mode
	Email    string // auto mode
	CacheDir string // auto mode
}

type Server struct {
	httpServer     *http.Server
	addr           string
	tlsOpts        TLSOptions
	certManager    *autocert.Manager
	redirectServer *http.Server
}

func New(host string, port int, handler http.Handler, tlsOpts TLSOptions, readTimeout, writeTimeout, idleTimeout time.Duration) *Server {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	s := &Server{
		addr:    addr,
		tlsOpts: tlsOpts,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}

	if tlsOpts.Mode == "auto" {
		s.certManager = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(tlsOpts.Domain),
			Cache:      autocert.DirCache(tlsOpts.CacheDir),
			Email:      tlsOpts.Email,
		}
		s.httpServer.TLSConfig = &tls.Config{
			GetCertificate: s.certManager.GetCertificate,
			MinVersion:     tls.VersionTLS12,
		}
		s.redirectServer = &http.Server{
			Addr:         ":80",
			Handler:      s.certManager.HTTPHandler(nil),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
	}

	return s
}

func (s *Server) Start() error {
	switch s.tlsOpts.Mode {
	case "auto":
		slog.Info("starting https server", "addr", s.addr, "tls", "auto", "domain", s.tlsOpts.Domain)
		go func() {
			slog.Info("starting http redirect server", "addr", ":80")
			if err := s.redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("http redirect server error", "error", err)
			}
		}()
		return s.httpServer.ListenAndServeTLS("", "")
	case "manual":
		slog.Info("starting https server", "addr", s.addr, "tls", "manual")
		return s.httpServer.ListenAndServeTLS(s.tlsOpts.CertFile, s.tlsOpts.KeyFile)
	default:
		slog.Info("starting server", "addr", s.addr)
		return s.httpServer.ListenAndServe()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")
	if s.redirectServer != nil {
		if err := s.redirectServer.Shutdown(ctx); err != nil {
			slog.Error("http redirect server shutdown error", "error", err)
		}
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) TLSMode() string {
	if s.tlsOpts.Mode == "" {
		return "off"
	}
	return s.tlsOpts.Mode
}
