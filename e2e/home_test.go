//go:build e2e

package e2e_test

import (
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHome(t *testing.T) {
	t.Parallel()
	_, page := newPage(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	require.NoError(t, expect.Locator(page.GetByText("Welcome!")).ToBeVisible())
}

// TestHome_GreetButton verifies the Greet button fires an alert under the app's
// nonce-based CSP. A CSP violation would silence the handler and the dialog
// would never arrive, causing the test to time out.
func TestHome_GreetButton(t *testing.T) {
	t.Parallel()
	_, page := newPage(t)
	_, err := page.Goto(getFullPath("/"))
	require.NoError(t, err)

	dialogCh := make(chan string, 1)
	page.OnDialog(func(dialog playwright.Dialog) {
		dialogCh <- dialog.Message()
		_ = dialog.Dismiss()
	})

	err = page.Locator("#greet-button").Click()
	require.NoError(t, err)

	select {
	case msg := <-dialogCh:
		assert.Equal(t, "Greet clicked", msg)
	case <-time.After(3 * time.Second):
		t.Fatal("expected alert dialog but none appeared — possible CSP violation blocking the click handler")
	}
}
