//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeaders(t *testing.T) {
	// Check security headers via HTTP request
	resp, err := http.Get(baseUrL.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify all security headers are present and correct
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"),
		"X-Frame-Options should prevent clickjacking")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"),
		"X-Content-Type-Options should prevent MIME-sniffing")
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"),
		"Referrer-Policy should be set")
	assert.Contains(t, resp.Header.Get("Permissions-Policy"), "geolocation=()",
		"Permissions-Policy should restrict browser features")

	// Note: HSTS header only set for HTTPS requests (not in test environment)
}

func TestCSRFProtection(t *testing.T) {
	t.Run("same-origin GET allowed", func(t *testing.T) {
		// Same-origin GET requests should always be allowed (safe method)
		beforeEach(t)

		_, err := page.Goto(baseUrL.String())
		require.NoError(t, err)

		// Verify page loaded successfully
		title, err := page.Title()
		require.NoError(t, err)
		assert.NotEmpty(t, title, "Page should load without CSRF blocking GET")
	})

	t.Run("cross-origin POST blocked", func(t *testing.T) {
		// Cross-origin POST requests should be blocked by CSRF protection
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", baseUrL.String()+"/health", nil)
		require.NoError(t, err)

		// Simulate cross-origin request
		req.Header.Set("Origin", "https://evil.com")
		req.Header.Set("Sec-Fetch-Site", "cross-site")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be rejected with 403 Forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode,
			"Cross-origin POST should be blocked by CSRF protection")
	})

	t.Run("same-site POST allowed", func(t *testing.T) {
		// Same-site POST requests should be allowed
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", baseUrL.String()+"/health", nil)
		require.NoError(t, err)

		// Simulate same-site request (what a real browser would send)
		req.Header.Set("Sec-Fetch-Site", "same-origin")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should be allowed (would return 405 Method Not Allowed for /health endpoint)
		// The important thing is it's NOT 403 Forbidden
		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode,
			"Same-site POST should not be blocked by CSRF")
	})
}

func TestRateLimiting(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Make rapid requests to trigger rate limit (50 req/min = ~1 per second)
	// We'll make 60 requests as fast as possible to ensure we hit the limit
	rateLimitHit := false
	var lastStatusCode int

	for i := 0; i < 60; i++ {
		resp, err := client.Get(baseUrL.String() + "/health")
		require.NoError(t, err)
		lastStatusCode = resp.StatusCode

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitHit = true

			// Verify Retry-After header is present
			retryAfter := resp.Header.Get("Retry-After")
			assert.NotEmpty(t, retryAfter, "Retry-After header should be set on 429 response")
			assert.Equal(t, "60", retryAfter, "Retry-After should suggest 60 seconds")

			resp.Body.Close()
			break
		}
		resp.Body.Close()

		// Small delay to avoid overwhelming test server
		time.Sleep(5 * time.Millisecond)
	}

	assert.True(t, rateLimitHit,
		"Rate limit should be triggered after rapid requests (last status: %d)", lastStatusCode)
}

func TestServerTimeouts(t *testing.T) {
	// Wait a bit for rate limiter to reset from previous tests
	time.Sleep(2 * time.Second)

	// Verify server responds normally (timeout configurations don't break normal requests)
	resp, err := http.Get(baseUrL.String() + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"Server should respond normally with timeout configurations")

	// Note: Testing actual timeout enforcement (slowloris, etc.) is complex
	// and beyond the scope of basic E2E tests. The important thing is that
	// the timeouts are configured and don't break normal operation.
}
