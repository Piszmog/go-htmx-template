//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTMXVersion guards the vendored bundle version and asserts the template
// does not reintroduce htmx 2 error-swap behavior through config: htmx 4 swaps
// every response except 204/304, and errors are returned as HTML fragments.
func TestHTMXVersion(t *testing.T) {
	t.Parallel()

	_, page := newPage(t)
	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	version, err := page.Evaluate("htmx.version")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(version.(string), "4."), "expected htmx 4.x, got %v", version)

	noSwap, err := page.Evaluate("htmx.config.noSwap.join(',')")
	require.NoError(t, err)
	assert.Equal(t, "204,304", noSwap, "error responses must swap in htmx 4")
}

// TestHTMXNoCSPViolations proves the CSP is compatible with htmx 4: indicator
// CSS is injected through adoptedStyleSheets, so style-src does not need
// 'unsafe-inline', and no other directive is violated during a swap.
func TestHTMXNoCSPViolations(t *testing.T) {
	t.Parallel()

	_, page := newPage(t)
	require.NoError(t, page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			window.__cspViolations = [];
			window.__swapped = false;
			document.addEventListener("securitypolicyviolation", function (e) {
				window.__cspViolations.push(e.violatedDirective + " " + e.blockedURI);
			});
			document.addEventListener("htmx:after:swap", function () {
				window.__swapped = true;
			});
		`),
	}))

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	require.NoError(t, page.Locator("#counter-button").Click())
	_, err = page.WaitForFunction("() => window.__swapped === true", nil)
	require.NoError(t, err, "expected an htmx swap to complete")

	violations, err := page.Evaluate("window.__cspViolations")
	require.NoError(t, err)
	assert.Empty(t, violations, "htmx must not trigger CSP violations")
}

// TestHTMXErrorRendersOutOfBand trips the rate limiter on the dedicated
// rate-limit app and asserts the 429 body is swapped into #error-region while
// the counter is left untouched. This is the htmx 4 error model: servers return
// HTML for error statuses, and out-of-band content keeps the main swap empty.
func TestHTMXErrorRendersOutOfBand(t *testing.T) {
	// Shares the rate-limit app (RATE_LIMIT=30) with the rate-limit tests, so the
	// token bucket must be manipulated serially.
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	_, page := newPage(t)
	_, err := page.Goto(rateLimitURL + "/")
	require.NoError(t, err)
	require.NoError(t, expect.Locator(page.Locator("#counter-button")).ToBeVisible())

	// Drain the per-IP token bucket so the next htmx request is rejected.
	_, err = page.Evaluate(`(async () => {
		for (let i = 0; i < 40; i++) {
			await fetch("/count", { method: "POST" });
		}
	})()`)
	require.NoError(t, err)

	before := counterValue(t, page)

	require.NoError(t, page.Locator("#counter-button").Click())

	require.NoError(t, expect.Locator(page.Locator("#error-region")).ToContainText("Rate limit exceeded"))

	// The main swap is suppressed because the response contains only out-of-band
	// content, so the component that issued the request survives untouched.
	assert.Equal(t, before, counterValue(t, page))
	require.NoError(t, expect.Locator(page.Locator("#counter-button")).ToBeVisible())
}
