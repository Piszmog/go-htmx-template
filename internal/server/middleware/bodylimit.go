package middleware

import "net/http"

// BodyLimit caps the size of request bodies. Handlers that read a body larger
// than maxBytes receive an *http.MaxBytesError from Read.
func BodyLimit(maxBytes int64) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
