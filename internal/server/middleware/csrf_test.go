package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/server/middleware"
)

// TestCSRF_HTMXRequestGetsOutOfBandError ensures the deny handler goes through
// middleware.WriteError: a rejected htmx request must not swap its error text
// into the element that issued it.
func TestCSRF_HTMXRequestGetsOutOfBandError(t *testing.T) {
	t.Parallel()

	called := false
	mw := middleware.CSRF(discardLogger(), middleware.IPConfig{TrustProxyHeaders: false})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.False(t, called, "cross-origin request must not reach the handler")
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rec.Body.String(), `id="error-region"`)
	assert.Contains(t, rec.Body.String(), `hx-swap-oob="true"`)
}
