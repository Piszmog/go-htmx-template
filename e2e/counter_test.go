//go:build e2e

package e2e_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounter_RendersWithInitialValue(t *testing.T) {
	_, page := newPage(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	// The counter is process-global, so assert the rendered shape rather than a
	// specific value. This keeps the test independent of execution order.
	text, err := page.Locator("#counter p").InnerText()
	require.NoError(t, err)
	assert.Regexp(t, `^Count: [0-9]+$`, text)
}

func TestCounter_IncrementViaHTMX(t *testing.T) {
	_, page := newPage(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	before := counterValue(t, page)

	err = page.Locator(`button:has-text("Increment")`).Click()
	require.NoError(t, err)

	// Wait for HTMX swap and verify counter increased by 1.
	expectedText := "Count: " + strconv.Itoa(before+1)
	require.NoError(t, expect.Locator(page.GetByText(expectedText)).ToBeVisible())
}

func TestCounter_IncrementsMultipleTimes(t *testing.T) {
	_, page := newPage(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	before := counterValue(t, page)

	// Click one at a time and wait for each swap. htmx 4 dropped extra clicks
	// under its default "queue first" strategy, so firing three clicks as fast as
	// possible asserts something htmx does not guarantee.
	incrementBtn := page.Locator(`button:has-text("Increment")`)
	for i := range 3 {
		require.NoError(t, incrementBtn.Click())

		expectedText := "Count: " + strconv.Itoa(before+i+1)
		require.NoError(t, expect.Locator(page.GetByText(expectedText)).ToBeVisible())
	}
}

// TestCounter_PreservesButtonNodeAfterSwap proves the outerMorph swap updates
// the existing DOM instead of replacing it: a property set on the button before
// the request is still there afterwards. An outerHTML swap would drop it.
func TestCounter_PreservesButtonNodeAfterSwap(t *testing.T) {
	_, page := newPage(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	before := counterValue(t, page)

	_, err = page.Evaluate(`document.getElementById("counter-button").swapMarker = "original-node"`)
	require.NoError(t, err)

	require.NoError(t, page.Locator("#counter-button").Click())

	expectedText := "Count: " + strconv.Itoa(before+1)
	require.NoError(t, expect.Locator(page.GetByText(expectedText)).ToBeVisible())

	marker, err := page.Evaluate(`document.getElementById("counter-button").swapMarker`)
	require.NoError(t, err)
	assert.Equal(t, "original-node", marker, "outerMorph should preserve the button node")
}

// counterValue reads the current counter value from the rendered page.
func counterValue(t *testing.T, page playwright.Page) int {
	t.Helper()
	text, err := page.Locator(`#counter p`).InnerText()
	require.NoError(t, err, "could not read counter element text")
	parts := strings.SplitN(text, ": ", 2)
	require.Len(t, parts, 2, "unexpected counter text format: %s", text)
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	require.NoError(t, err, "could not parse counter value from: %s", text)
	return n
}
