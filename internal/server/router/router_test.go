package router_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/dist"
	"go-htmx-template/internal/server/router"
)

// newTestHandler builds a router with a nil database and the given per-minute
// rate limit. The cancel func is registered with t.Cleanup.
func newTestHandler(t *testing.T, rateLimit int) http.Handler {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return router.New(ctx, slog.New(slog.DiscardHandler), nil, rateLimit, false)
}

// serve issues one request from a fixed client IP so rate-limit tests share a
// single budget. headers are key/value pairs.
func serve(t *testing.T, handler http.Handler, method, path string, headers ...string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	req.RemoteAddr = "198.51.100.1:12345"
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// TestNew_RateLimitScope verifies that static assets and the health endpoint do
// not consume the per-IP rate limit budget, while dynamic routes still do.
func TestNew_RateLimitScope(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, 2)

	htmxPath, err := dist.AssetPath("js/htmx@*.min.js")
	require.NoError(t, err)

	for range 5 {
		require.Equal(t, http.StatusOK, serve(t, handler, http.MethodGet, "/health").Code)
	}
	for range 5 {
		require.Equal(t, http.StatusOK, serve(t, handler, http.MethodGet, "/assets/"+htmxPath).Code)
	}

	assert.Equal(t, http.StatusOK, serve(t, handler, http.MethodGet, "/").Code)
	assert.Equal(t, http.StatusOK, serve(t, handler, http.MethodGet, "/").Code)
	assert.Equal(t, http.StatusTooManyRequests, serve(t, handler, http.MethodGet, "/").Code)
}

// TestNew_AssetsDoNotListDirectories verifies the embedded asset server cannot
// be used to enumerate the asset tree.
func TestNew_AssetsDoNotListDirectories(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, 100)

	for _, path := range []string{"/assets/", "/assets/js/", "/assets/css/"} {
		require.Equal(t, http.StatusNotFound, serve(t, handler, http.MethodGet, path).Code, "asset directory %s", path)
	}

	require.Equal(t, http.StatusOK, serve(t, handler, http.MethodGet, "/assets/favicon.svg").Code)
}

// TestNew_ReadyzRoute verifies the readiness endpoint is wired outside the
// rate-limited dynamic routes and reports an unavailable database.
func TestNew_ReadyzRoute(t *testing.T) {
	t.Parallel()

	rec := serve(t, newTestHandler(t, 100), http.MethodGet, "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

// TestNew_HTMXNotFoundIsHTML verifies the dynamic fallback renders an error
// fragment for htmx requests instead of the net/http plain-text 404.
func TestNew_HTMXNotFoundIsHTML(t *testing.T) {
	t.Parallel()

	rec := serve(t, newTestHandler(t, 100), http.MethodGet, "/does-not-exist", "HX-Request", "true")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rec.Body.String(), `id="error-region"`)
}

// TestNew_MethodNotAllowed verifies that a known path rejects an unsupported
// method with 405 and an Allow header instead of falling through to the 404
// catch-all.
func TestNew_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, 1000)

	tests := []struct {
		name   string
		method string
		path   string
		allow  string
	}{
		{name: "readiness probe", method: http.MethodPut, path: "/readyz", allow: "GET, HEAD"},
		{name: "health probe", method: http.MethodPost, path: "/health", allow: "GET, HEAD"},
		{name: "counter", method: http.MethodGet, path: "/count", allow: "POST"},
		{name: "static asset", method: http.MethodPost, path: "/assets/favicon.svg", allow: "GET, HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := serve(t, handler, tt.method, tt.path)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			assert.Equal(t, tt.allow, rec.Header().Get("Allow"))
		})
	}

	assert.Equal(t, http.StatusNotFound, serve(t, handler, http.MethodGet, "/does-not-exist").Code,
		"unknown paths must still 404")
	assert.Equal(t, http.StatusOK, serve(t, handler, http.MethodHead, "/health").Code,
		"HEAD must be served by GET routes")
}

// TestNew_SmallResponsesAreNotCompressed pins the interaction between the
// handlers and the compression middleware: a JSON probe answers with a couple
// of dozen bytes, so it needs no gzip frame and must keep its Content-Length.
func TestNew_SmallResponsesAreNotCompressed(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(t, 50)

	for _, path := range []string{"/health", "/readyz"} {
		rec := serve(t, handler, http.MethodGet, path, "Accept-Encoding", "gzip")

		assert.Empty(t, rec.Header().Get("Content-Encoding"), "%s must not be compressed", path)
		assert.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"), path)
	}
}
