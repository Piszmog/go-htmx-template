package middleware

import (
	"go-htmx-template/version"
	"net/http"
)

// CacheMiddleware sets the Cache-Control header based on the version.
func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if version.Value == "dev" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
