package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"go-htmx-template/internal/db/queries"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Database interface {
	DB() *sql.DB
	Queries() *queries.Queries
	Close() error
}

func New(url string) (Database, error) {
	db, err := newLocalDB(url)
	if err != nil {
		return nil, err
	}
	if err = db.DB().PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return db, nil
}

// Migrate runs the migrations on the database. Assumes the database is SQLite.
func Migrate(db Database) (err error) {
	driver, err := sqlite3.WithInstance(db.DB(), &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	iofsDriver, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs: %w", err)
	}
	defer func() {
		if cerr := iofsDriver.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close driver: %w", cerr))
		}
	}()

	m, err := migrate.NewWithInstance("iofs", iofsDriver, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	if err = m.Up(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}
