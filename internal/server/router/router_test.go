package router_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/dist"
	"go-htmx-template/internal/server/router"
)

// TestNew_RateLimitScope verifies that static assets and the health endpoint do
// not consume the per-IP rate limit budget, while dynamic routes still do.
func TestNew_RateLimitScope(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	handler := router.New(ctx, slog.New(slog.DiscardHandler), nil, 2, false)

	do := func(method, path string) int {
		t.Helper()
		req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
		req.RemoteAddr = "198.51.100.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	htmxPath, err := dist.AssetPath("js/htmx@*.min.js")
	require.NoError(t, err)

	for range 5 {
		require.Equal(t, http.StatusOK, do(http.MethodGet, "/health"))
	}
	for range 5 {
		require.Equal(t, http.StatusOK, do(http.MethodGet, "/assets/"+htmxPath))
	}

	assert.Equal(t, http.StatusOK, do(http.MethodGet, "/"))
	assert.Equal(t, http.StatusOK, do(http.MethodGet, "/"))
	assert.Equal(t, http.StatusTooManyRequests, do(http.MethodGet, "/"))
}

// TestNew_HTMXNotFoundIsHTML verifies the dynamic fallback renders an error
// fragment for htmx requests instead of the net/http plain-text 404.
func TestNew_HTMXNotFoundIsHTML(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	handler := router.New(ctx, slog.New(slog.DiscardHandler), nil, 100, false)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/does-not-exist", nil)
	req.RemoteAddr = "198.51.100.1:12345"
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, rec.Body.String(), `id="error-region"`)
}
