package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecovery_PanicCaught(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic, should handle gracefully
	assert.NotPanics(t, func() {
		middleware.ServeHTTP(rec, req)
	})
}

func TestRecovery_Returns500(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "500 Internal Server Error")
}

func TestRecovery_LogsPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	panicMessage := "something went wrong"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(panicMessage)
	})

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "panic recovered")
	assert.Contains(t, logOutput, panicMessage)
	assert.Contains(t, logOutput, "method=POST")
	assert.Contains(t, logOutput, "path=/api/test")
	assert.Contains(t, logOutput, "stack=") // Stack trace logged
}

func TestRecovery_NoPanic(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("all good"))
	})

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "all good", rec.Body.String())

	logOutput := logBuffer.String()
	assert.NotContains(t, logOutput, "panic recovered")
}

func TestRecovery_ChainsCorrectly(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Handler that increments a counter in middleware chain
	var middlewareCalled bool
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain: Recovery -> TestMiddleware -> Handler
	middleware := Recovery(logger)(testMiddleware(handler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "middleware chain should execute")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRecovery_PanicWithNilValue(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	})

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should handle nil panic gracefully
	assert.NotPanics(t, func() {
		middleware.ServeHTTP(rec, req)
	})

	// Note: panic(nil) doesn't actually trigger defer recover in Go
	// But we test it doesn't cause issues
}

func TestRecovery_PanicWithError(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	testError := assert.AnError
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(testError)
	})

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "panic recovered")
	assert.Contains(t, logOutput, testError.Error())
}
