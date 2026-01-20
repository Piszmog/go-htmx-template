package middleware

import (
	"log/slog"
	"net/http"
)

// Security returns a middleware that sets security headers.
// These headers provide defense-in-depth against common web attacks.
func Security(logger *slog.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent clickjacking attacks - page cannot be embedded in iframe
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent MIME-sniffing attacks - browser must respect Content-Type
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Control referrer information leakage
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Restrict browser features (geolocation, microphone, camera)
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Enforce HTTPS (even though Caddy handles TLS, this adds defense-in-depth)
			// Only set if request is HTTPS (check X-Forwarded-Proto from Caddy or direct TLS)
			if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
				// max-age=31536000 = 1 year, includeSubDomains applies to all subdomains
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			logger.Debug("security headers set", slog.String("path", r.URL.Path))
			next.ServeHTTP(w, r)
		})
	}
}
