package db

import (
	"fmt"
	"net/url"
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
		u = &url.URL{Scheme: "file", Path: path}
	}

	q := u.Query()
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()

	return u.String(), nil
}
