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
