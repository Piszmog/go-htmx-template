package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-htmx-template/internal/db"
)

func TestMigrate_CreatesSchemaAndIsIdempotent(t *testing.T) {
	t.Parallel()

	database, err := db.New(filepath.Join(t.TempDir(), "migrate.sqlite3"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	ctx := context.Background()

	require.NoError(t, db.Migrate(ctx, database.DB()))
	assert.True(t, tableExists(t, database.DB(), "authors"))
	assert.True(t, tableExists(t, database.DB(), "schema_migrations"))

	// A second run must be a no-op, not an error.
	require.NoError(t, db.Migrate(ctx, database.DB()))
}

func TestMigrate_ClosedDatabaseReturnsError(t *testing.T) {
	t.Parallel()

	database, err := db.New(filepath.Join(t.TempDir(), "closed.sqlite3"))
	require.NoError(t, err)
	require.NoError(t, database.Close())

	require.Error(t, db.Migrate(context.Background(), database.DB()))
}

func tableExists(t *testing.T, sqlDB *sql.DB, name string) bool {
	t.Helper()

	var count int
	err := sqlDB.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		name,
	).Scan(&count)
	require.NoError(t, err)
	return count == 1
}
