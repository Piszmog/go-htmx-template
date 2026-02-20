package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

// Security returns a middleware that sets security headers.
func Security(ipCfg IPConfig) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := generateNonce()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			// 'unsafe-inline' is kept for style-src because HTMX v2 injects inline
			// styles for request indicators. script-src uses a per-request nonce so
			// that templ script components work without 'unsafe-inline'.
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					fmt.Sprintf("script-src 'self' 'nonce-%s'; ", nonce)+
				"script-src-attr 'unsafe-inline'; "+
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
				// No preload: preload requires submission to the HSTS preload list and
				// is irreversible for the lifetime of the entry.
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r.WithContext(templ.WithNonce(r.Context(), nonce)))
		})
	}
}

func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
