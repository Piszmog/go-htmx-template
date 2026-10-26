package main

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/version"
)

func TestParseBool(t *testing.T) {
	t.Parallel()

	parsers := map[string]struct {
		parse    func(string) (bool, error)
		sentinel error
	}{
		"MIGRATE_ON_START":    {parse: parseMigrateOnStart, sentinel: errInvalidMigrateOnStart},
		"TRUST_PROXY_HEADERS": {parse: parseTrustProxyHeaders, sentinel: errInvalidTrustProxyHeaders},
	}

	for name, parser := range parsers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name    string
				raw     string
				want    bool
				wantErr bool
			}{
				{name: "unset", raw: "", want: false},
				{name: "true", raw: "true", want: true},
				{name: "false", raw: "false", want: false},
				{name: "invalid", raw: "sometimes", wantErr: true},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					got, err := parser.parse(tt.raw)
					if tt.wantErr {
						require.ErrorIs(t, err, parser.sentinel)
						return
					}
					require.NoError(t, err)
					assert.Equal(t, tt.want, got)
				})
			}
		})
	}
}

func TestParseRateLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "unset uses the default", raw: "", want: defaultRateLimit},
		{name: "valid", raw: "120", want: 120},
		{name: "zero", raw: "0", wantErr: true},
		{name: "negative", raw: "-5", wantErr: true},
		{name: "not a number", raw: "fast", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseRateLimit(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunMigrate_CreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.sqlite3")
	t.Setenv("DB_URL", path)

	require.NoError(t, runMigrate(slog.New(slog.DiscardHandler), nil))
	assert.Equal(t, 1, authorsTableCount(t, path))
}

func TestExitCode_Help(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}, {"serve", "--help"}, {"migrate", "-h"}} {
		arg := strings.Join(args, " ")

		var out bytes.Buffer
		code := exitCode(slog.New(slog.DiscardHandler), args, &out)

		assert.Equal(t, 0, code, "%s", arg)
		assert.Contains(t, out.String(), "Usage: my-app", "%s", arg)
		assert.Contains(t, out.String(), "migrate", "%s", arg)
	}
}

func TestExitCode_Version(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"version"}, {"-v"}, {"--version"}, {"serve", "-v"}} {
		arg := strings.Join(args, " ")

		var out bytes.Buffer
		code := exitCode(slog.New(slog.DiscardHandler), args, &out)

		assert.Equal(t, 0, code, "%s", arg)
		assert.Equal(t, version.Value, strings.TrimSpace(out.String()), "%s", arg)
	}
}

func TestExitCode_RejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code := exitCode(slog.New(slog.DiscardHandler), []string{"migrat"}, &out)

	assert.Equal(t, 2, code, "a typo must not silently start the server")
	assert.Contains(t, out.String(), `unknown command "migrat"`)
	assert.Contains(t, out.String(), "Usage: my-app")
}

func TestExitCode_RejectsUnexpectedArgument(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	code := exitCode(slog.New(slog.DiscardHandler), []string{"serve", "--prt", "9090"}, &out)

	assert.Equal(t, 2, code, "a stray argument must not silently start the server")
	assert.Contains(t, out.String(), `unexpected argument "--prt 9090"`)
	assert.Contains(t, out.String(), "Usage: my-app")
}

func TestExitCode_MigrateFailure(t *testing.T) {
	dbURL := filepath.Join(t.TempDir(), "missing", "db.sqlite3")
	t.Setenv("DB_URL", dbURL)

	var out bytes.Buffer
	code := exitCode(slog.New(slog.DiscardHandler), []string{"migrate"}, &out)

	assert.Equal(t, 1, code, "a failed migration must not report success")
}

func TestRunMigrate_DownRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "down.sqlite3")
	t.Setenv("DB_URL", path)

	logger := slog.New(slog.DiscardHandler)
	require.NoError(t, runMigrate(logger, nil))
	assert.Equal(t, 1, authorsTableCount(t, path))

	require.NoError(t, runMigrate(logger, []string{"down", "1"}))
	assert.Equal(t, 0, authorsTableCount(t, path))
}

func TestRunMigrate_DownWithoutStepsRollsBackEverything(t *testing.T) {
	path := filepath.Join(t.TempDir(), "downall.sqlite3")
	t.Setenv("DB_URL", path)

	logger := slog.New(slog.DiscardHandler)
	require.NoError(t, runMigrate(logger, nil))
	assert.Equal(t, 1, authorsTableCount(t, path))

	require.NoError(t, runMigrate(logger, []string{"down"}))
	assert.Equal(t, 0, authorsTableCount(t, path))

	// Rolling back an already clean schema is a no-op, not an error.
	require.NoError(t, runMigrate(logger, []string{"down"}))
}

func TestRunMigrate_RejectsInvalidArguments(t *testing.T) {
	t.Setenv("DB_URL", filepath.Join(t.TempDir(), "invalid.sqlite3"))

	logger := slog.New(slog.DiscardHandler)
	for _, args := range [][]string{
		{"sideways"},
		{"down", "many"},
		{"down", "0"},
		{"down", "1", "2"},
		{"up", "3"},
	} {
		require.Error(t, runMigrate(logger, args), "%v", args)
	}
}

// authorsTableCount reports how many tables named "authors" exist, which is how
// these tests observe whether the up and down migrations ran.
func authorsTableCount(t *testing.T, path string) int {
	t.Helper()

	database, err := db.New(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	var count int
	err = database.DB().QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'authors'",
	).Scan(&count)
	require.NoError(t, err)
	return count
}
