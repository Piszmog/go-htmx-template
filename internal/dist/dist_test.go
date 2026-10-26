package dist_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/dist"
)

func TestAssetPath_HTMX(t *testing.T) {
	t.Parallel()

	path, err := dist.AssetPath("js/htmx@*.min.js")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, "js/htmx@v4."), "path = %q", path)
	assert.True(t, strings.HasSuffix(path, ".min.js"), "path = %q", path)
}

func TestAssetPath_NoMatch(t *testing.T) {
	t.Parallel()

	_, err := dist.AssetPath("js/missing@*.js")
	require.Error(t, err)
}

// TestAssetPath_MultipleMatches documents the contract callers rely on when
// they resolve a versioned asset: a pattern must identify exactly one file, so
// a stale vendored bundle is a hard error rather than a silent pick.
func TestAssetPath_MultipleMatches(t *testing.T) {
	t.Parallel()

	// Matches the favicon and the js directory, plus the css directory when it
	// has been generated, so it is ambiguous either way.
	_, err := dist.AssetPath("*")
	require.ErrorContains(t, err, "multiple")
}
