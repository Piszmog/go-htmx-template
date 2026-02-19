package middleware

import "net/http"

// Security returns a middleware that sets security headers.
func Security(ipCfg IPConfig) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			// 'unsafe-inline' is required for templ's onclick attribute handlers
			// (e.g., onclick="...") and inline styles. HTMX v2 does not need it. This is a
			// deliberate trade-off: inline event handler support at the cost of weakened
			// CSP XSS protection.
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'unsafe-inline'; "+
					"style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; "+
					"connect-src 'self'; "+
					"font-src 'self'; "+
					"object-src 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'; "+
					"frame-ancestors 'none'",
			)

			isHTTPS := r.TLS != nil
			if !isHTTPS && ipCfg.TrustProxyHeaders {
				isHTTPS = r.Header.Get("X-Forwarded-Proto") == "https"
			}
			if isHTTPS {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}
