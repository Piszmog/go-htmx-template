package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"go-htmx-template/internal/db/queries"
)

const maxOpenConns = 4

type localDB struct {
	db      *sql.DB
	queries *queries.Queries
}

var _ Database = (*localDB)(nil)

func (d *localDB) DB() *sql.DB {
	return d.db
}

func (d *localDB) Queries() *queries.Queries {
	return d.queries
}

func (d *localDB) Close() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("closing database: %w", err)
	}
	return nil
}

func newLocalDB(path string) (*localDB, error) {
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	return &localDB{db: db, queries: queries.New(db)}, nil
}
