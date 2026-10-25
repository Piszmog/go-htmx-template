package db_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/db/queries"
)

// TestNew_QueriesRoundTrip proves the constructor and generated queries work
// together against a real SQLite file, including the DSN pragmas.
func TestNew_QueriesRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "test.sqlite3")

	database, err := db.New(path)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	ctx := context.Background()

	migration, err := os.ReadFile(filepath.Join("migrations", "20240407203525_init.up.sql"))
	require.NoError(t, err)
	_, err = database.DB().ExecContext(ctx, string(migration))
	require.NoError(t, err)

	var foreignKeys int
	require.NoError(t, database.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys))
	assert.Equal(t, 1, foreignKeys)

	var journalMode string
	require.NoError(t, database.DB().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
	assert.Equal(t, "wal", strings.ToLower(journalMode))
	created, err := database.Queries().CreateAuthor(ctx, queries.CreateAuthorParams{
		Name: "Ada Lovelace",
		Bio:  sql.NullString{String: "Mathematician", Valid: true},
	})
	require.NoError(t, err)

	authors, err := database.Queries().ListAuthors(ctx)
	require.NoError(t, err)
	require.Len(t, authors, 1)
	assert.Equal(t, created.ID, authors[0].ID)
	assert.Equal(t, "Ada Lovelace", authors[0].Name)
}

func TestNew_InvalidPath(t *testing.T) {
	t.Parallel()

	_, err := db.New(filepath.Join(t.TempDir(), "missing", "db.sqlite3"))
	require.Error(t, err)
}
