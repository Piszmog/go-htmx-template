package middleware

import (
	"log/slog"
	"net/http"
)

// CSRF returns a middleware that provides CSRF protection using Go's native
// http.CrossOriginProtection (Go 1.25+). No tokens required in forms.
func CSRF(logger *slog.Logger) Handler {
	cop := http.NewCrossOriginProtection()

	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Warn("CSRF protection rejected cross-origin request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote", GetClientIP(r)),
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
