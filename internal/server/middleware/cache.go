package middleware

import (
	"net/http"

	"go-htmx-template/internal/version"
)

// CacheMiddleware sets the Cache-Control header based on the version.
func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if version.Value == "dev" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			// Safe to cache for a year: the version either appears in the asset
			// name (output@<version>.css) or is pinned (htmx@vX.Y.Z.min.js), so a
			// release always fetches a new URL. favicon.svg is the exception — it
			// carries no version, so replacing the icon means changing its path
			// (or shortening this max-age) rather than bumping the version.
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
