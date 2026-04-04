package db

import (
	"database/sql"
	"fmt"
	"go-htmx-template/internal/db/queries"

	_ "modernc.org/sqlite"
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
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	return &localDB{db: db, queries: queries.New(db)}, nil
}

// NewFromRawDB creates a Database from an existing *sql.DB. Useful for testing.
func NewFromRawDB(rawDB *sql.DB) Database {
	rawDB.SetMaxOpenConns(maxOpenConns)
	return &localDB{db: rawDB, queries: queries.New(rawDB)}
}
