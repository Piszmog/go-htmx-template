package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogging_CapturesStatusCode(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "status=201")
}

func TestLogging_CapturesBytesWritten(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	testBody := "test response body"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(testBody))
	})

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, testBody, rec.Body.String())
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "bytes=18")
}

func TestLogging_DefaultStatus200(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't call WriteHeader, should default to 200
		_, _ = w.Write([]byte("ok"))
	})

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "status=200")
}

func TestLogging_MultipleWrites(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello "))
		_, _ = w.Write([]byte("World"))
	})

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, "Hello World", rec.Body.String())
	logOutput := logBuffer.String()
	// Total bytes: 6 + 5 = 11
	assert.Contains(t, logOutput, "bytes=11")
}

func TestLogging_AllFields(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	})

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := logBuffer.String()

	// Verify all expected log fields are present
	assert.Contains(t, logOutput, "Handled request")
	assert.Contains(t, logOutput, "method=POST")
	assert.Contains(t, logOutput, "path=/api/users")
	assert.Contains(t, logOutput, "remote=192.168.1.1:12345")
	assert.Contains(t, logOutput, "status=200")
	assert.Contains(t, logOutput, "bytes=8")
	assert.Contains(t, logOutput, "duration=")
}

func TestResponseWriter_Unwrap(t *testing.T) {
	originalWriter := httptest.NewRecorder()
	wrapped := newResponseWriter(originalWriter)

	unwrapped := wrapped.Unwrap()
	require.NotNil(t, unwrapped)
	assert.Equal(t, originalWriter, unwrapped)
}

func TestResponseWriter_WriteHeaderOnlyOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusBadRequest) // Should be ignored

	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestLogging_NoLogWhenNotDebugLevel(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Set to INFO, not DEBUG
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := logBuffer.String()
	// Should not log at debug level when logger is set to INFO
	assert.Empty(t, strings.TrimSpace(logOutput))
}
