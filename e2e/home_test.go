//go:build e2e

package e2e_test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHome(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	require.NoError(t, expect.Locator(page.GetByText("Welcome!")).ToBeVisible())
}
