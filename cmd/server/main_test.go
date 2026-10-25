package main

import (
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/db"
)

func TestParseMigrateOnStart(t *testing.T) {
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

			got, err := parseMigrateOnStart(tt.raw)
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

	logger := slog.New(slog.DiscardHandler)
	require.NoError(t, runMigrate(logger))

	database, err := db.New(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	var count int
	err = database.DB().QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'authors'",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
