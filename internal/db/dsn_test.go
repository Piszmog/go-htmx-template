//nolint:testpackage // asserts unexported sqliteDSN behaviour
package db

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDSN(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		path string
	}{
		{name: "dot relative", path: "./db.sqlite3"},
		{name: "bare relative", path: "db.sqlite3"},
		{name: "parent relative", path: "../db.sqlite3"},
		{name: "absolute", path: "/tmp/app.sqlite3"},
		{name: "file url", path: "file:/tmp/app.sqlite3"},
		{name: "file url with query", path: "file:/tmp/app.sqlite3?mode=rwc"},
		{name: "path with separators", path: "name?with#separators.db"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dsn, err := sqliteDSN(tt.path)
			require.NoError(t, err)

			parsed, err := url.Parse(dsn)
			require.NoError(t, err)

			// A relative path rendered into the authority slot is what the
			// driver rejects with "invalid uri authority: .".
			assert.Empty(t, parsed.Host, "authority must stay empty")
			assert.Equal(t, expectedPath(t, tt.path), parsed.Path)
			assert.Len(t, parsed.Query()["_pragma"], 3)
			if strings.Contains(tt.path, "mode=rwc") {
				assert.Equal(t, "rwc", parsed.Query().Get("mode"))
			}
		})
	}
}

func TestFileURLPath(t *testing.T) {
	t.Parallel()

	t.Run("absolute path keeps its leading slash", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "/tmp/app.sqlite3", fileURLPath(filepath.FromSlash("/tmp/app.sqlite3")))
	})

	t.Run("drive letter gets a leading slash", func(t *testing.T) {
		t.Parallel()
		if runtime.GOOS != "windows" {
			t.Skip("drive-letter paths only exist on Windows")
		}
		assert.Equal(t, "/C:/tmp/app.sqlite3", fileURLPath(`C:\tmp\app.sqlite3`))
	})
}

func expectedPath(t *testing.T, path string) string {
	t.Helper()

	// Only a file: URL treats "?" as a query separator; elsewhere it is part
	// of the file name and must be escaped into the path.
	if trimmed, ok := strings.CutPrefix(path, "file:"); ok {
		path = trimmed
		if i := strings.Index(path, "?"); i >= 0 {
			path = path[:i]
		}
	}

	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return abs
}
