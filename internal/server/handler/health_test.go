package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_Returns200(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealth_ReturnsJSON(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestHealth_CorrectBody(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{"status":"ok"}`, string(body))
}

func TestHealth_ValidJSONStructure(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	var result map[string]string
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "ok", result["status"])
}

func TestHealth_NoDatabase(t *testing.T) {
	// Handler with nil Database should still work for health check
	h := &Handler{
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
	h := &Handler{}

	// Health endpoint should be idempotent
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		h.Health(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
	}
}
