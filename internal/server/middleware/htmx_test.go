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

	body := rec.Body.String()
	// The fragment must target the layout placeholder out of band, otherwise htmx
	// swaps the error into the element that issued the request.
	assert.Contains(t, body, `id="error-region"`)
	assert.Contains(t, body, `hx-swap-oob="true"`)
	assert.Contains(t, body, `role="alert"`)
	assert.Contains(t, body, "Rate limit exceeded.")
}

func TestWriteError_NonHTMXRequestReturnsPlainText(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/count", nil)
	rec := httptest.NewRecorder()

	middleware.WriteError(discardLogger(), rec, req, http.StatusInternalServerError, "500 Internal Server Error")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
	assert.Equal(t, "500 Internal Server Error\n", rec.Body.String())
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
