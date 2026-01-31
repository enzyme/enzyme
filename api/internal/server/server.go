package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	httpServer *http.Server
	addr       string
}

func New(host string, port int, handler http.Handler) *Server {
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	return &Server{
		addr: addr,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.addr
}
