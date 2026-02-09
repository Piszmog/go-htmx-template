package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-htmx-template/internal/server/middleware"
	"go-htmx-template/internal/version"
)

const devVersion = "dev"

// TestGetClientIP_DevMode verifies that in dev mode (no reverse proxy),
// proxy headers are ignored and only RemoteAddr is used.
// These tests cannot be parallel because they modify the global version.Value.
func TestGetClientIP_DevMode(t *testing.T) { //nolint:paralleltest // modifies global version.Value
	origVersion := version.Value
	version.Value = devVersion
	t.Cleanup(func() { version.Value = origVersion })

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr IPv6 with port",
			remoteAddr: "[::1]:8080",
			expected:   "::1",
		},
		{
			name:       "ignores X-Forwarded-For",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			expected:   "127.0.0.1",
		},
		{
			name:       "ignores X-Real-IP",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "203.0.113.1"},
			expected:   "127.0.0.1",
		},
		{
			name:       "ignores X-Forwarded-For chain",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 70.41.3.18"},
			expected:   "127.0.0.1",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // modifies global version.Value
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := middleware.GetClientIP(req)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestGetClientIP_ProductionMode verifies that in production mode (behind a
// reverse proxy like Caddy), proxy headers are parsed and validated.
// These tests cannot be parallel because they modify the global version.Value.
func TestGetClientIP_ProductionMode(t *testing.T) { //nolint:paralleltest // modifies global version.Value
	origVersion := version.Value
	version.Value = "1.0.0"
	t.Cleanup(func() { version.Value = origVersion })

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "uses X-Forwarded-For single IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			expected:   "203.0.113.1",
		},
		{
			name:       "uses first IP from X-Forwarded-For chain",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1, 70.41.3.18, 198.51.100.178"},
			expected:   "203.0.113.1",
		},
		{
			name:       "trims whitespace from X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "  203.0.113.1  , 70.41.3.18"},
			expected:   "203.0.113.1",
		},
		{
			name:       "strips port from X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1:8080"},
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For IPv6",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "2001:db8::1"},
			expected:   "2001:db8::1",
		},
		{
			name:       "X-Forwarded-For IPv6 with port",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "[2001:db8::1]:8080"},
			expected:   "2001:db8::1",
		},
		{
			name:       "invalid X-Forwarded-For falls back to X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "not-an-ip",
				"X-Real-IP":       "203.0.113.1",
			},
			expected: "203.0.113.1",
		},
		{
			name:       "invalid X-Forwarded-For and X-Real-IP falls back to RemoteAddr",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "garbage",
				"X-Real-IP":       "also-garbage",
			},
			expected: "10.0.0.1",
		},
		{
			name:       "uses X-Real-IP when no X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "203.0.113.1"},
			expected:   "203.0.113.1",
		},
		{
			name:       "strips port from X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "203.0.113.1:9090"},
			expected:   "203.0.113.1",
		},
		{
			name:       "invalid X-Real-IP falls back to RemoteAddr",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "not-valid"},
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
				"X-Real-IP":       "198.51.100.1",
			},
			expected: "203.0.113.1",
		},
		{
			name:       "falls back to RemoteAddr when no proxy headers",
			remoteAddr: "10.0.0.1:12345",
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // modifies global version.Value
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := middleware.GetClientIP(req)
			assert.Equal(t, tt.expected, got)
		})
	}
}
