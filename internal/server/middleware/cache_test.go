package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-htmx-template/internal/server/middleware"
	"go-htmx-template/internal/version"
)

//nolint:paralleltest // mutates package-level version.Value; parallelism would cause a data race
func TestCacheMiddleware_DevMode(t *testing.T) {
	original := version.Value
	version.Value = "dev"
	defer func() { version.Value = original }()

	handler := middleware.CacheMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
}

//nolint:paralleltest // mutates package-level version.Value; parallelism would cause a data race
func TestCacheMiddleware_ProdMode(t *testing.T) {
	original := version.Value
	version.Value = "1.2.3"
	defer func() { version.Value = original }()

	handler := middleware.CacheMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "public, max-age=31536000", rec.Header().Get("Cache-Control"))
}
