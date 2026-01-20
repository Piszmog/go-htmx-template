package middleware

import (
	"log/slog"
	"net/http"
)

// Security returns a middleware that sets security headers.
func Security(logger *slog.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			logger.Debug("security headers set", slog.String("path", r.URL.Path))
			next.ServeHTTP(w, r)
		})
	}
}
