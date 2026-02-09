package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging returns a middleware handler that logs requests.
func Logging(logger *slog.Logger, ipCfg IPConfig) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)
			logger.Debug(
				"Handled request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote", GetClientIP(r, ipCfg)),
				slog.Int("status", rw.statusCode),
				slog.Int("bytes", rw.bytesWritten),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}
