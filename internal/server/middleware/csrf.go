package middleware

import (
	"log/slog"
	"net/http"
)

// CSRF returns a middleware that provides CSRF protection using Go's native
// http.CrossOriginProtection (available in Go 1.24+).
//
// This protects against Cross-Site Request Forgery attacks by checking:
// 1. Sec-Fetch-Site header (modern browsers, universally available since 2023)
// 2. Origin header vs Host header comparison (fallback)
//
// Key features:
// - NO CSRF tokens required in forms (protection is transparent)
// - GET, HEAD, OPTIONS are always allowed (safe methods)
// - POST, PUT, DELETE, PATCH are checked for cross-origin requests
// - Stateless operation (no session storage needed)
//
// Browser support: All modern browsers (Chrome 76+, Firefox 90+, Safari 15.5+)
func CSRF(logger *slog.Logger) Handler {
	cop := http.NewCrossOriginProtection()

	// Set custom deny handler to log rejected requests
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Warn("CSRF protection rejected cross-origin request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", r.RemoteAddr),
			slog.String("origin", r.Header.Get("Origin")),
			slog.String("host", r.Host),
			slog.String("sec_fetch_site", r.Header.Get("Sec-Fetch-Site")),
		)
		http.Error(w, "Cross-origin request forbidden", http.StatusForbidden)
	}))

	return func(next http.Handler) http.Handler {
		return cop.Handler(next)
	}
}
