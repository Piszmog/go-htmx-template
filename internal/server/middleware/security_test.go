package middleware_test

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/server/middleware"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSecurity_StaticHeaders(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Contains(t, rec.Header().Get("Permissions-Policy"), "geolocation=()")
}

func TestSecurity_CSPContainsNonce(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "script-src 'self' 'nonce-")
	assert.NotEmpty(t, extractSecurityNonce(csp), "nonce value should not be empty")
}

func TestSecurity_NonceIsUniqueAcrossRequests(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	nonces := make([]string, 3)
	for i := range 3 {
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		csp := rec.Header().Get("Content-Security-Policy")
		nonces[i] = extractSecurityNonce(csp)
		require.NotEmpty(t, nonces[i], "nonce[%d] should not be empty", i)
	}

	assert.NotEqual(t, nonces[0], nonces[1], "nonces should differ between requests")
	assert.NotEqual(t, nonces[1], nonces[2], "nonces should differ between requests")
	assert.NotEqual(t, nonces[0], nonces[2], "nonces should differ between requests")
}

func TestSecurity_NonceStoredInTemplContext(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	var contextNonce string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextNonce = templ.GetNonce(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cspNonce := extractSecurityNonce(rec.Header().Get("Content-Security-Policy"))
	require.NotEmpty(t, contextNonce, "nonce should be in templ context")
	require.NotEmpty(t, cspNonce, "nonce should be in CSP header")
	assert.Equal(t, cspNonce, contextNonce, "CSP nonce and context nonce should match")
}

func TestSecurity_HSTSNotSetForPlainHTTP(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"), "HSTS should not be set for plain HTTP")
}

func TestSecurity_HSTSSetWhenTLS(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set for TLS requests")
	assert.Contains(t, hsts, "max-age=31536000")
}

func TestSecurity_HSTSSetWhenForwardedProtoAndTrustProxies(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: true}
	mw := middleware.Security(discardLogger(), cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set when X-Forwarded-Proto is https and proxy headers are trusted")
	assert.Contains(t, hsts, "max-age=31536000")
}

func TestSecurity_HSTSNotSetWhenForwardedProtoButNoTrustProxies(t *testing.T) {
	t.Parallel()

	cfg := middleware.IPConfig{TrustProxyHeaders: false}
	mw := middleware.Security(discardLogger(), cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"),
		"HSTS should not be set when X-Forwarded-Proto is https but proxy headers are not trusted")
}

// extractSecurityNonce extracts the nonce value from a CSP header.
func extractSecurityNonce(csp string) string {
	const prefix = "'nonce-"
	idx := strings.Index(csp, prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(csp[start:], "'")
	if end < 0 {
		return ""
	}
	return csp[start : start+end]
}
