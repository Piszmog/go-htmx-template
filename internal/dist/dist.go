package dist

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

var (
	errNoAssetMatch         = errors.New("no embedded asset matches pattern")
	errMultipleAssetMatches = errors.New("multiple embedded assets match pattern")
)

//go:embed all:assets
var AssetsDir embed.FS

// AssetPath returns the path of the single embedded asset matching pattern,
// relative to the assets directory. Pattern is a glob such as
// "js/htmx@*.min.js".
func AssetPath(pattern string) (string, error) {
	matches, err := fs.Glob(AssetsDir, "assets/"+pattern)
	if err != nil {
		return "", fmt.Errorf("globbing assets: %w", err)
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %q", errNoAssetMatch, pattern)
	case 1:
		return strings.TrimPrefix(matches[0], "assets/"), nil
	default:
		return "", fmt.Errorf("%w: %q: %v", errMultipleAssetMatches, pattern, matches)
	}
}
