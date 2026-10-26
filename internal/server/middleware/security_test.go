package middleware_test

import (
	"context"
	"crypto/tls"
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
	return slog.New(slog.DiscardHandler)
}

// serve runs a middleware-wrapped no-op handler and returns the recorder.
func serve(t *testing.T, mw middleware.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)
	return rec
}

func TestSecurityHeaders_StaticHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := serve(t, middleware.SecurityHeaders(middleware.IPConfig{}), req)

	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Opener-Policy"))
	assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Resource-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Contains(t, rec.Header().Get("Permissions-Policy"), "geolocation=()")
	assert.Equal(t, "none", rec.Header().Get("X-Permitted-Cross-Domain-Policies"))
}

// TestSecurityHeaders_CSPOverriddenByCSP proves the chain emits a single CSP
// header, with the nonce, when both middlewares are applied.
func TestSecurityHeaders_CSPOverriddenByCSP(t *testing.T) {
	t.Parallel()

	mw := middleware.SecurityHeaders(middleware.IPConfig{})(middleware.CSP(discardLogger())(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	cspHeaders := rec.Header().Values("Content-Security-Policy")
	require.Len(t, cspHeaders, 1, "a second CSP header would be ignored by browsers")
	assert.Contains(t, cspHeaders[0], "'nonce-")
	assert.NotEmpty(t, extractSecurityNonce(cspHeaders[0]))
}

// TestSecurityHeaders_BaselineCSPHasNoNonce proves static responses get a
// policy without the per-request nonce work.
func TestSecurityHeaders_BaselineCSPHasNoNonce(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/assets/favicon.svg", nil)
	rec := serve(t, middleware.SecurityHeaders(middleware.IPConfig{}), req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "script-src 'self';")
	assert.NotContains(t, csp, "'nonce-")
	assert.Contains(t, csp, "style-src 'self';")
	assert.NotContains(t, csp, "'unsafe-inline'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestCSP_AddsNonce(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := serve(t, middleware.CSP(discardLogger()), req)

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "script-src 'self' 'nonce-")
	assert.NotEmpty(t, extractSecurityNonce(csp), "nonce value should not be empty")
	assert.NotRegexp(t, `script-src [^;]*'unsafe-inline'`, csp)
}

func TestCSP_NonceIsUniqueAcrossRequests(t *testing.T) {
	t.Parallel()

	nonces := make([]string, 3)
	for i := range nonces {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		rec := serve(t, middleware.CSP(discardLogger()), req)
		nonces[i] = extractSecurityNonce(rec.Header().Get("Content-Security-Policy"))
		require.NotEmpty(t, nonces[i], "nonce[%d] should not be empty", i)
	}

	assert.NotEqual(t, nonces[0], nonces[1], "nonces should differ between requests")
	assert.NotEqual(t, nonces[1], nonces[2], "nonces should differ between requests")
	assert.NotEqual(t, nonces[0], nonces[2], "nonces should differ between requests")
}

func TestCSP_NonceStoredInTemplContext(t *testing.T) {
	t.Parallel()

	var contextNonce string
	handler := middleware.CSP(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextNonce = templ.GetNonce(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cspNonce := extractSecurityNonce(rec.Header().Get("Content-Security-Policy"))
	require.NotEmpty(t, contextNonce, "nonce should be in templ context")
	require.NotEmpty(t, cspNonce, "nonce should be in CSP header")
	assert.Equal(t, cspNonce, contextNonce, "CSP nonce and context nonce should match")
}

func TestSecurityHeaders_HSTSNotSetForPlainHTTP(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := serve(t, middleware.SecurityHeaders(middleware.IPConfig{}), req)

	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"), "HSTS should not be set for plain HTTP")
}

func TestSecurityHeaders_HSTSSetWhenTLS(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := serve(t, middleware.SecurityHeaders(middleware.IPConfig{}), req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set for TLS requests")
	assert.Contains(t, hsts, "max-age=31536000")
}

func TestSecurityHeaders_HSTSSetWhenForwardedProtoAndTrustProxies(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := serve(t, middleware.SecurityHeaders(middleware.IPConfig{TrustProxyHeaders: true}), req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotEmpty(t, hsts, "HSTS should be set when X-Forwarded-Proto is https and proxy headers are trusted")
	assert.Contains(t, hsts, "max-age=31536000")
}

func TestSecurityHeaders_HSTSNotSetWhenForwardedProtoButNoTrustProxies(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := serve(t, middleware.SecurityHeaders(middleware.IPConfig{}), req)

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
