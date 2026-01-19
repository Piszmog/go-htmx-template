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

func TestRecovery_PanicHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		panicValue           interface{}
		method               string
		path                 string
		expectedStatus       int
		expectedBodyContains string
		expectedLogContains  []string
	}{
		{
			name:                 "recovers from string panic",
			panicValue:           "test panic",
			method:               http.MethodGet,
			path:                 "/test",
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: "500 Internal Server Error",
			expectedLogContains:  []string{"panic recovered", "test panic"},
		},
		{
			name:                 "returns 500 status on panic",
			panicValue:           "test panic",
			method:               http.MethodGet,
			path:                 "/test",
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: "500 Internal Server Error",
			expectedLogContains:  []string{"panic recovered"},
		},
		{
			name:                 "logs panic details with stack trace",
			panicValue:           "something went wrong",
			method:               http.MethodPost,
			path:                 "/api/test",
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: "500 Internal Server Error",
			expectedLogContains: []string{
				"panic recovered",
				"something went wrong",
				"method=POST",
				"path=/api/test",
				"stack=",
			},
		},
		{
			name:                 "recovers from error panic",
			panicValue:           assert.AnError,
			method:               http.MethodGet,
			path:                 "/test",
			expectedStatus:       http.StatusInternalServerError,
			expectedBodyContains: "500 Internal Server Error",
			expectedLogContains:  []string{"panic recovered", assert.AnError.Error()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panicValue)
			})

			mw := middleware.Recovery(logger)(handler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			// Should not panic, should handle gracefully
			assert.NotPanics(t, func() {
				mw.ServeHTTP(rec, req)
			})

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.expectedBodyContains)

			logOutput := logBuffer.String()
			for _, logField := range tt.expectedLogContains {
				assert.Contains(t, logOutput, logField)
			}
		})
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	t.Parallel()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("all good"))
	})

	mw := middleware.Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "all good", rec.Body.String())

	logOutput := logBuffer.String()
	assert.NotContains(t, logOutput, "panic recovered")
}

func TestRecovery_ChainsCorrectly(t *testing.T) {
	t.Parallel()
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
	mw := middleware.Recovery(logger)(testMiddleware(handler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	assert.True(t, middlewareCalled, "middleware chain should execute")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRecovery_PanicWithNilValue(t *testing.T) {
	t.Parallel()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	})

	mw := middleware.Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should handle nil panic gracefully
	assert.NotPanics(t, func() {
		mw.ServeHTTP(rec, req)
	})

	// Note: panic(nil) doesn't actually trigger defer recover in Go
	// But we test it doesn't cause issues
}
