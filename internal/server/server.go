package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	writeTimeout      = 15 * time.Second
	readTimeout       = 15 * time.Second
	idleTimeout       = 60 * time.Second
	readHeaderTimeout = 5 * time.Second
	maxHeaderBytes    = 1 << 20
	shutdownTimeout   = 10 * time.Second
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
		WriteTimeout:      writeTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	server := &Server{srv: srv, logger: logger, errCh: make(chan error, 1)}
	for _, opt := range opts {
		opt(server)
	}

	return server
}

// Option represents a server option.
type Option func(*Server)

// WithRouter sets the handler.
func WithRouter(handler http.Handler) Option {
	return func(s *Server) {
		s.srv.Handler = handler
	}
}

// StartAndWait starts the server and waits for a signal to shut down.
func (s *Server) StartAndWait() error {
	s.start()
	return s.gracefulShutdown()
}

func (s *Server) start() {
	go func() {
		s.logger.Info("starting server", "port", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.errCh <- err
		}
	}()
}

func (s *Server) gracefulShutdown() error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a shutdown signal or a fatal server error.
	select {
	case <-sig:
		signal.Stop(sig)
		// Drain errCh: if the server failed to start concurrently, return that error.
		select {
		case err := <-s.errCh:
			return fmt.Errorf("server failed to start: %w", err)
		default:
		}
	case err := <-s.errCh:
		return fmt.Errorf("server failed to start: %w", err)
	}

	s.logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.logger.Info("server stopped")
	return nil
}
