package middleware

import (
	"net/http"
)

// contentSecurityPolicy builds the policy for the given script-src value. The
// baseline uses 'self'; routes that render templ components replace it with a
// per-request nonce via CSP.
func contentSecurityPolicy(scriptSrc string) string {
	// htmx 4 injects its request-indicator CSS through document.adoptedStyleSheets,
	// which CSP style-src does not apply to, so no 'unsafe-inline' is needed.
	return "default-src 'self'; " +
		"script-src " + scriptSrc + "; " +
		"style-src 'self'; " +
		"img-src 'self' data:; " +
		"connect-src 'self'; " +
		"font-src 'self'; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"frame-ancestors 'none'"
}

// SecurityHeaders sets the headers that every response carries. It deliberately
// does not mint a CSP nonce: static assets, JSON endpoints, and error pages do
// not execute scripts, so only the routes that render templ components need one
// (see CSP).
func SecurityHeaders(ipCfg IPConfig) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			h.Set("X-Permitted-Cross-Domain-Policies", "none")
			h.Set("Content-Security-Policy", contentSecurityPolicy("'self'"))

			if isHTTPS(r, ipCfg) {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isHTTPS reports whether the request reached the client over TLS, either
// directly or through a trusted proxy that terminated it.
func isHTTPS(r *http.Request, ipCfg IPConfig) bool {
	if r.TLS != nil {
		return true
	}
	return ipCfg.TrustProxyHeaders && r.Header.Get("X-Forwarded-Proto") == "https"
}
