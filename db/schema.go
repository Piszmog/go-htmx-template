package db

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed schema.sql
var Schema string

// InitSchema initializes the database schema.
func InitSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, Schema)
	return err
}
