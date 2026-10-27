package db

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// sqliteDSN builds a SQLite URI from a file path or file URL. Paths are
// escaped so characters such as '?' and '#' are treated as part of the file
// name, and driver pragmas are added without clobbering an existing query.
func sqliteDSN(path string) (string, error) {
	var u *url.URL

	if strings.HasPrefix(path, "file:") {
		parsed, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("parsing database URL: %w", err)
		}
		u = parsed
	} else {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolving database path: %w", err)
		}
		u = &url.URL{Scheme: "file", Path: fileURLPath(abs)}
	}

	q := u.Query()
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// fileURLPath renders an absolute filesystem path as the path component of a
// file: URL. The path must be absolute, or url.URL writes it into the authority
// slot ("file://./db.sqlite3"), which the driver rejects as an invalid URI
// authority. Windows drive letters need a leading slash for the same reason.
func fileURLPath(abs string) string {
	path := filepath.ToSlash(abs)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
