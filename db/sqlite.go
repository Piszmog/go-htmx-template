package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// NewSQLite creates a new SQLite database.
func NewSQLite(dataSourceName string) (*sql.DB, error) {
	return sql.Open("sqlite", dataSourceName)
}
