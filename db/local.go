package db

import (
	"database/sql"
	"go-htmx-template/db/queries"
	"log/slog"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

type LocalDB struct {
	logger  *slog.Logger
	db      *sql.DB
	queries *queries.Queries
}

var _ Database = (*LocalDB)(nil)

func (d *LocalDB) DB() *sql.DB {
	return d.db
}

func (d *LocalDB) Queries() *queries.Queries {
	return d.queries
}

func (d *LocalDB) Logger() *slog.Logger {
	return d.logger
}

func (d *LocalDB) Close() error {
	return d.db.Close()
}

func newLocalDB(logger *slog.Logger, path string) (*LocalDB, error) {
	db, err := sql.Open("libsql", "file:"+path)
	if err != nil {
		return nil, err
	}
	return &LocalDB{logger: logger, db: db, queries: queries.New(db)}, nil
}
