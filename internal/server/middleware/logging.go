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

	// Wrap the response writer to capture status code and bytes written
	rw := newResponseWriter(w)

	// Call next handler with wrapped writer
	l.handler.ServeHTTP(rw, r)

	// Log with captured status and bytes
	l.logger.Debug(
		"Handled request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("remote", r.RemoteAddr),
		slog.Int("status", rw.statusCode),
		slog.Int("bytes", rw.bytesWritten),
		slog.Duration("duration", time.Since(start)),
	)
}

// Logging returns a middleware handler that logs requests.
func Logging(logger *slog.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return &LoggingMiddleware{
			logger:  logger,
			handler: next,
		}
	}
}
