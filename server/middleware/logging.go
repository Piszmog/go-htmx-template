package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware represents a logging middleware.
type LoggingMiddleware struct {
	Logger *slog.Logger
}

func (m *LoggingMiddleware) ServeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		m.Logger.Debug(
			"Handled request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
