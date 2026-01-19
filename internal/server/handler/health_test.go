package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			name: "returns valid JSON structure with version field",
			validate: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				var result map[string]string
				err := json.NewDecoder(rec.Body).Decode(&result)
				require.NoError(t, err)
				assert.Equal(t, "dev", result["version"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := &handler.Handler{}
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			h.Health(rec, req)

			tt.validate(t, rec)
		})
	}
}

func TestHealth_NoDatabase(t *testing.T) {
	t.Parallel()
	// Handler with nil Database should still work for health check
	h := &handler.Handler{
		Logger:   nil,
		Database: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Should not panic or error even without database
	assert.NotPanics(t, func() {
		h.Health(rec, req)
	})

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealth_MultipleRequests(t *testing.T) {
	t.Parallel()
	h := &handler.Handler{}

	// Health endpoint should be idempotent
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.Health(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"version":"dev"}`, rec.Body.String())
	}
}
