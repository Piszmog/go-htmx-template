package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware represents a logging middleware.
type LoggingMiddleware struct {
	logger  *slog.Logger
	handler http.Handler
}

// NewLoggingMiddleware creates a new logging middleware with the given logger and handler.
func NewLoggingMiddleware(logger *slog.Logger, handler http.Handler) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger:  logger,
		handler: handler,
	}
}

// ServeHTTP logs the request and calls the next handler.
func (l *LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	l.handler.ServeHTTP(w, r)
	l.logger.Debug(
		"Handled request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("remote", r.RemoteAddr),
		slog.Duration("duration", time.Since(start)),
	)
}
