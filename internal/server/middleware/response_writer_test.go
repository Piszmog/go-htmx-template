package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-htmx-template/internal/server/middleware"
)

// loggingRig wires up the Logging middleware (which wraps responseWriter
// internally) and returns the log buffer so tests can inspect captured fields.
func loggingRig(t *testing.T) (middleware.Handler, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return middleware.Logging(logger, middleware.IPConfig{}), &buf
}

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	t.Parallel()
	lm, logBuf := loggingRig(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	lm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// write nothing — responseWriter default is 200
	})).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, logBuf.String(), "status=200")
}

func TestResponseWriter_WriteHeaderCapturesCode(t *testing.T) {
	t.Parallel()
	lm, logBuf := loggingRig(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	lm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, logBuf.String(), "status=404")
}

func TestResponseWriter_WriteHeaderIsIdempotent(t *testing.T) {
	t.Parallel()
	lm, logBuf := loggingRig(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	lm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.WriteHeader(http.StatusInternalServerError) // must be ignored
	})).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, logBuf.String(), "status=404")
	assert.NotContains(t, logBuf.String(), "status=500")
}

func TestResponseWriter_WriteCountsBytesAcrossMultipleCalls(t *testing.T) {
	t.Parallel()
	lm, logBuf := loggingRig(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	lm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Hello"))
		assert.NoError(t, err)
		_, err = w.Write([]byte(" World"))
		assert.NoError(t, err)
	})).ServeHTTP(rec, req)

	assert.Equal(t, "Hello World", rec.Body.String())
	assert.Contains(t, logBuf.String(), "bytes=11")
}

func TestResponseWriter_WriteImplicitlyCallsWriteHeader(t *testing.T) {
	t.Parallel()
	lm, logBuf := loggingRig(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	lm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("body")) // no explicit WriteHeader call
		assert.NoError(t, err)
	})).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, logBuf.String(), "status=200")
}

func TestResponseWriter_UnwrapReturnsUnderlying(t *testing.T) {
	t.Parallel()
	lm, _ := loggingRig(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	lm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.NewResponseController calls Unwrap() to find Flush on the
		// underlying *httptest.ResponseRecorder.
		err := http.NewResponseController(w).Flush()
		assert.NoError(t, err, "Flush via ResponseController requires Unwrap to work")
	})).ServeHTTP(rec, req)

	assert.True(t, rec.Flushed, "underlying recorder should have been flushed")
}
