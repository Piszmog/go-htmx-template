package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server represents an HTTP server.
type Server struct {
	srv    *http.Server
	logger *slog.Logger
	errCh  chan error
}

// New creates a new server with the given logger, address and options.
func New(logger *slog.Logger, addr string, opts ...Option) *Server {
	srv := &http.Server{
		Addr:              addr,
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	server := &Server{srv: srv, logger: logger, errCh: make(chan error, 1)}
	for _, opt := range opts {
		opt(server)
	}

	return server
}

// Option represents a server option.
type Option func(*Server)

// WithWriteTimeout sets the write timeout.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.WriteTimeout = timeout
	}
}

// WithReadTimeout sets the read timeout.
func WithReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.ReadTimeout = timeout
	}
}

// WithIdleTimeout sets the idle timeout.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.IdleTimeout = timeout
	}
}

// WithReadHeaderTimeout sets the read header timeout.
func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.srv.ReadHeaderTimeout = timeout
	}
}

// WithMaxHeaderBytes sets the maximum header bytes.
func WithMaxHeaderBytes(bytes int) Option {
	return func(s *Server) {
		s.srv.MaxHeaderBytes = bytes
	}
}

// WithRouter sets the handler.
func WithRouter(handler http.Handler) Option {
	return func(s *Server) {
		s.srv.Handler = handler
	}
}

// StartAndWait starts the server and waits for a signal to shut down.
func (s *Server) StartAndWait() {
	s.Start()
	s.GracefulShutdown()
}

// Start starts the server.
func (s *Server) Start() {
	go func() {
		s.logger.Info("starting server", "port", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.errCh <- err
		}
	}()
}

// GracefulShutdown shuts down the server gracefully.
func (s *Server) GracefulShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a shutdown signal or a fatal server error.
	select {
	case <-sig:
	case err := <-s.errCh:
		s.logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Error("server shutdown error", "error", err)
	}

	s.logger.Info("server stopped")
}
