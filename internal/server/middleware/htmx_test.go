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

func TestWriteError_HTMXRequestReturnsOutOfBandFragment(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	middleware.WriteError(discardLogger(), rec, req, http.StatusTooManyRequests, "Rate limit exceeded.")

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	body := rec.Body.String()
	// The fragment must target the layout placeholder out of band, otherwise htmx
	// swaps the error into the element that issued the request. The live region
	// (role="alert") lives in the layout and must not be replaced, or screen
	// readers have nothing to announce into.
	assert.Contains(t, body, `id="error-message"`)
	assert.Contains(t, body, `hx-swap-oob="true"`)
	assert.NotContains(t, body, `role="alert"`)
	assert.Contains(t, body, "Rate limit exceeded.")
}

func TestWriteError_NonHTMXRequestReturnsHTMLPage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	middleware.WriteError(discardLogger(), rec, req, http.StatusInternalServerError, "500 Internal Server Error")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	body := rec.Body.String()
	assert.Contains(t, body, "<!doctype html>")
	assert.Contains(t, body, "500 Internal Server Error")
	assert.NotContains(t, body, "<script")
}

func TestWriteError_NonHTMXRequestEscapesMessage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	middleware.WriteError(discardLogger(), rec, req, http.StatusBadRequest, `<script>alert("xss")</script>`)

	assert.NotContains(t, rec.Body.String(), "<script>")
	assert.Contains(t, rec.Body.String(), "&lt;script&gt;")
}

func TestWriteError_EscapesMessage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	middleware.WriteError(discardLogger(), rec, req, http.StatusBadRequest, `<script>alert("xss")</script>`)

	assert.NotContains(t, rec.Body.String(), "<script>")
	assert.Contains(t, rec.Body.String(), "&lt;script&gt;")
}
