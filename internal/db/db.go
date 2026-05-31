package db

import (
	"context"
	"database/sql"
	"fmt"
	"go-htmx-template/internal/db/queries"
)

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
