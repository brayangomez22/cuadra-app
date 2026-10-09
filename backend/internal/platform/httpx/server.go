// Package httpx provides the HTTP server, shared middleware and response helpers.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Server is an HTTP server with sane timeouts and graceful shutdown.
type Server struct {
	srv             *http.Server
	shutdownTimeout time.Duration
}

// NewServer builds a Server listening on addr.
func NewServer(addr string, handler http.Handler, shutdownTimeout time.Duration) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		shutdownTimeout: shutdownTimeout,
	}
}

// Run listens on the configured address and serves until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.srv.Addr, err)
	}
	return s.Serve(ctx, ln)
}

// Serve serves on ln until ctx is canceled, then shuts down gracefully: it
// stops accepting connections and waits for in-flight requests up to the
// shutdown timeout.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	// ctx is already canceled; the shutdown deadline must not inherit it.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()
	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		_ = s.srv.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
