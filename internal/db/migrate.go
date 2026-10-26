package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies all pending embedded migrations. It is idempotent and
// returns nil when the schema is already current.
func Migrate(ctx context.Context, sqlDB *sql.DB) error {
	runner, err := newRunner(ctx, sqlDB)
	if err != nil {
		return err
	}

	if err := runner.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

// MigrateDown rolls back the most recent steps. A step count of zero or less
// rolls back every applied migration.
func MigrateDown(ctx context.Context, sqlDB *sql.DB, steps int) error {
	runner, err := newRunner(ctx, sqlDB)
	if err != nil {
		return err
	}

	var downErr error
	if steps <= 0 {
		downErr = runner.Down()
	} else {
		downErr = runner.Steps(-steps)
	}
	if downErr != nil && !errors.Is(downErr, migrate.ErrNoChange) {
		return fmt.Errorf("rolling back migrations: %w", downErr)
	}

	return nil
}

// newRunner builds a migration runner from the embedded files. The driver wraps
// the caller's *sql.DB and must not be closed, since closing it would close the
// shared connection pool.
func newRunner(ctx context.Context, sqlDB *sql.DB) (*migrate.Migrate, error) {
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("creating migration source: %w", err)
	}

	driver, err := sqlite.WithInstance(sqlDB, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("creating migration driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("creating migration runner: %w", err)
	}

	return runner, nil
}
