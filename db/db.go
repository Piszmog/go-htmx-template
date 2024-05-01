package db

import (
	"database/sql"
	"embed"
	"fmt"
	"go-htmx-template/db/queries"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Database interface {
	DB() *sql.DB
	Queries() *queries.Queries
	Logger() *slog.Logger
	Close() error
}

func New(logger *slog.Logger, url string) (Database, error) {
	db, err := newLocalDB(logger, url)
	if err != nil {
		return nil, err
	}
	if err = db.db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// Migrate runs the migrations on the database. Assumes the database is SQLite.
func Migrate(db Database) error {
	driver, err := sqlite3.WithInstance(db.DB(), &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	iofsDriver, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs: %w", err)
	}
	defer iofsDriver.Close()

	m, err := migrate.NewWithInstance("iofs", iofsDriver, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	return m.Up()
}
