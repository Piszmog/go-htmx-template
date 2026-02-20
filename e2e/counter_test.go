//go:build e2e

package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCounter_RendersWithInitialValue(t *testing.T) {
	beforeEach(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	require.NoError(t, expect.Locator(page.GetByText("Count: 0")).ToBeVisible())
}

func TestCounter_IncrementViaHTMX(t *testing.T) {
	beforeEach(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	err = page.GetByRole("button", nil).Filter(nil).Nth(0).Click()
	if err != nil {
		// Try locating by text content
		err = page.Locator(`button:has-text("Increment")`).Click()
	}
	require.NoError(t, err)

	require.NoError(t, expect.Locator(page.GetByText("Count: 1")).ToBeVisible())
}

func TestCounter_IncrementsMultipleTimes(t *testing.T) {
	beforeEach(t)

	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	incrementBtn := page.Locator(`button:has-text("Increment")`)
	for range 3 {
		require.NoError(t, incrementBtn.Click())
	}

	require.NoError(t, expect.Locator(page.GetByText("Count: 3")).ToBeVisible())
}
