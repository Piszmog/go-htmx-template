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
	resp, err := http.Get(baseUrL.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.Contains(t, resp.Header.Get("Permissions-Policy"), "geolocation=()")
}

func TestCSRFProtection(t *testing.T) {
	t.Run("same-origin GET allowed", func(t *testing.T) {
		beforeEach(t)

		_, err := page.Goto(baseUrL.String())
		require.NoError(t, err)

		title, err := page.Title()
		require.NoError(t, err)
		assert.NotEmpty(t, title)
	})

	t.Run("cross-origin POST blocked", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", baseUrL.String()+"/health", nil)
		require.NoError(t, err)

		req.Header.Set("Origin", "https://evil.com")
		req.Header.Set("Sec-Fetch-Site", "cross-site")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("same-site POST allowed", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", baseUrL.String()+"/health", nil)
		require.NoError(t, err)

		req.Header.Set("Sec-Fetch-Site", "same-origin")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestRateLimiting(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	rateLimitHit := false
	var lastStatusCode int

	for i := 0; i < 60; i++ {
		resp, err := client.Get(baseUrL.String() + "/health")
		require.NoError(t, err)
		lastStatusCode = resp.StatusCode

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitHit = true

			retryAfter := resp.Header.Get("Retry-After")
			assert.NotEmpty(t, retryAfter)
			assert.Equal(t, "60", retryAfter)

			resp.Body.Close()
			break
		}
		resp.Body.Close()

		time.Sleep(5 * time.Millisecond)
	}

	assert.True(t, rateLimitHit,
		"Rate limit should be triggered after rapid requests (last status: %d)", lastStatusCode)
}

func TestServerTimeouts(t *testing.T) {
	time.Sleep(2 * time.Second)

	resp, err := http.Get(baseUrL.String() + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
