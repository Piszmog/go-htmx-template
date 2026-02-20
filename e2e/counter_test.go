//go:build e2e

package e2e_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounter_RendersWithInitialValue(t *testing.T) {
	// This test must run first in the E2E suite (counter_test.go is
	// alphabetically first) before any test clicks the Increment button.
	beforeEach(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	require.NoError(t, expect.Locator(page.GetByText("Count: 0")).ToBeVisible())
}

func TestCounter_IncrementViaHTMX(t *testing.T) {
	beforeEach(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	before := counterValue(t)

	err = page.Locator(`button:has-text("Increment")`).Click()
	require.NoError(t, err)

	// Wait for HTMX swap and verify counter increased by 1.
	expectedText := "Count: " + strconv.Itoa(before+1)
	require.NoError(t, expect.Locator(page.GetByText(expectedText)).ToBeVisible())
}

func TestCounter_IncrementsMultipleTimes(t *testing.T) {
	beforeEach(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	before := counterValue(t)

	incrementBtn := page.Locator(`button:has-text("Increment")`)
	for range 3 {
		require.NoError(t, incrementBtn.Click())
	}

	expectedText := "Count: " + strconv.Itoa(before+3)
	require.NoError(t, expect.Locator(page.GetByText(expectedText)).ToBeVisible())
	assert.Greater(t, before+3, before, "counter should increase with each click")
}

// counterValue reads the current counter value from the rendered page.
func counterValue(t *testing.T) int {
	t.Helper()
	text, err := page.Locator(`#counter p`).InnerText()
	require.NoError(t, err, "could not read counter element text")
	parts := strings.SplitN(text, ": ", 2)
	require.Len(t, parts, 2, "unexpected counter text format: %s", text)
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	require.NoError(t, err, "could not parse counter value from: %s", text)
	return n
}
