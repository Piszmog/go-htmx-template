package handler_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/server/handler"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		validate func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "returns 200 status code",
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "returns JSON content type",
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			},
		},
		{
			name: "returns correct JSON body",
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				body, err := io.ReadAll(rec.Body)
				require.NoError(t, err)
				assert.JSONEq(t, `{"version":"dev"}`, string(body))
			},
		},
		{
			name: "sets an accurate content length",
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := handler.New(slog.New(slog.DiscardHandler), nil)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			h.Health(rec, req)

			tt.validate(t, rec)
		})
	}
}

func TestHealth_NoStore(t *testing.T) {
	t.Parallel()

	h := handler.New(slog.New(slog.DiscardHandler), nil)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

func TestReady_Healthy(t *testing.T) {
	t.Parallel()

	database, err := db.New(filepath.Join(t.TempDir(), "ready.sqlite3"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	h := handler.New(slog.New(slog.DiscardHandler), database)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestReady_Unhealthy(t *testing.T) {
	t.Parallel()

	database, err := db.New(filepath.Join(t.TempDir(), "ready.sqlite3"))
	require.NoError(t, err)
	require.NoError(t, database.Close())

	h := handler.New(slog.New(slog.DiscardHandler), database)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
	assert.JSONEq(t, `{"status":"unavailable"}`, rec.Body.String())
}

func TestReady_NilDatabase(t *testing.T) {
	t.Parallel()

	h := handler.New(slog.New(slog.DiscardHandler), nil)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHealth_NoDatabase(t *testing.T) {
	t.Parallel()
	// Handler with nil Database should still work for health check
	h := handler.New(nil, nil)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Should not panic or error even without database
	assert.NotPanics(t, func() {
		h.Health(rec, req)
	})

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealth_MultipleRequests(t *testing.T) {
	t.Parallel()
	h := handler.New(slog.New(slog.DiscardHandler), nil)

	// Health endpoint should be idempotent
	for range 5 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.Health(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"version":"dev"}`, rec.Body.String())
	}
}
