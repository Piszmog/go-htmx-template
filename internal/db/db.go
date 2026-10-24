package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go-htmx-template/internal/db/queries"
)

const pingTimeout = 5 * time.Second

// Database provides access to the underlying SQL database and generated queries.
type Database interface {
	DB() *sql.DB
	Queries() *queries.Queries
	Close() error
}

// New opens the database at url and verifies it is reachable.
func New(url string) (Database, error) {
	database, err := newLocalDB(url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := database.DB().PingContext(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("pinging database: %w", err), database.Close())
	}

	return database, nil
}
