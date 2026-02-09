package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-htmx-template/internal/server/middleware"
)

func TestLogging_Middleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		handler           http.HandlerFunc
		method            string
		path              string
		remoteAddr        string
		expectedStatus    int
		expectedBody      string
		expectedLogFields []string
	}{
		{
			name: "captures custom status code",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte("created"))
			},
			method:            http.MethodPost,
			path:              "/test",
			remoteAddr:        "",
			expectedStatus:    http.StatusCreated,
			expectedBody:      "created",
			expectedLogFields: []string{"status=201"},
		},
		{
			name: "captures bytes written",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("test response body"))
			},
			method:            http.MethodGet,
			path:              "/test",
			remoteAddr:        "",
			expectedStatus:    http.StatusOK,
			expectedBody:      "test response body",
			expectedLogFields: []string{"bytes=18"},
		},
		{
			name: "defaults to status 200 when not set",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Don't call WriteHeader, should default to 200
				_, _ = w.Write([]byte("ok"))
			},
			method:            http.MethodGet,
			path:              "/test",
			remoteAddr:        "",
			expectedStatus:    http.StatusOK,
			expectedBody:      "ok",
			expectedLogFields: []string{"status=200"},
		},
		{
			name: "accumulates bytes from multiple writes",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("Hello "))
				_, _ = w.Write([]byte("World"))
			},
			method:            http.MethodGet,
			path:              "/test",
			remoteAddr:        "",
			expectedStatus:    http.StatusOK,
			expectedBody:      "Hello World",
			expectedLogFields: []string{"bytes=11"},
		},
		{
			name: "logs all request fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("response"))
			},
			method:         http.MethodPost,
			path:           "/api/users",
			remoteAddr:     "192.168.1.1:12345",
			expectedStatus: http.StatusOK,
			expectedBody:   "response",
			expectedLogFields: []string{
				"Handled request",
				"method=POST",
				"path=/api/users",
				"remote=192.168.1.1",
				"status=200",
				"bytes=8",
				"duration=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			mw := middleware.Logging(logger)(tt.handler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}
			rec := httptest.NewRecorder()

			mw.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			assert.Equal(t, tt.expectedBody, rec.Body.String())

			logOutput := logBuffer.String()
			for _, field := range tt.expectedLogFields {
				assert.Contains(t, logOutput, field)
			}
		})
	}
}

func TestLogging_NoLogWhenNotDebugLevel(t *testing.T) {
	t.Parallel()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Set to INFO, not DEBUG
	}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mw := middleware.Logging(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	logOutput := logBuffer.String()
	// Should not log at debug level when logger is set to INFO
	assert.Empty(t, strings.TrimSpace(logOutput))
}
