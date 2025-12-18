package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// Server represents an HTTP server.
type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

// New creates a new server with the given logger, address and options.
func New(logger *slog.Logger, addr string, opts ...Option) *Server {
	srv := &http.Server{
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	server := &Server{srv: srv, logger: logger}
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
		if err := s.srv.ListenAndServe(); err != nil {
			s.logger.Warn("failed to start server", "error", err)
		}
	}()
}

// GracefulShutdown shuts down the server gracefully.
func (s *Server) GracefulShutdown() {
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = s.srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	s.logger.Info("shutting down")
}
