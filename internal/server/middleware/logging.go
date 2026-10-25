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

			// Deferred so a panic that unwinds through this middleware still
			// produces an access log entry once Recovery writes the 500.
			defer func() {
				logger.Info(
					"Handled request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote", GetClientIP(r, ipCfg)),
					slog.Int("status", rw.statusCode),
					slog.Int("bytes", rw.bytesWritten),
					slog.Duration("duration", time.Since(start)),
				)
			}()

			next.ServeHTTP(rw, r)
		})
	}
}
