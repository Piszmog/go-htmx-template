package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

// nonceBytes is the size of the random CSP nonce.
const nonceBytes = 16

// CSP replaces the baseline policy from SecurityHeaders with one that trusts a
// per-request nonce, and stores that nonce in the request context for templ.
// Apply it only to routes that render templ components: static assets and JSON
// endpoints keep the cheaper baseline policy and never mint a nonce.
func CSP(logger *slog.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nonce, err := generateNonce()
			if err != nil {
				logger.ErrorContext(r.Context(), "failed to generate nonce", "error", err)
				WriteError(logger, w, r, http.StatusInternalServerError, "500 Internal Server Error")
				return
			}

			w.Header().Set("Content-Security-Policy", contentSecurityPolicy(fmt.Sprintf("'self' 'nonce-%s'", nonce)))
			next.ServeHTTP(w, r.WithContext(templ.WithNonce(r.Context(), nonce)))
		})
	}
}

func generateNonce() (string, error) {
	b := make([]byte, nonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
