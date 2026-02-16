package main

import (
	"context"
	"errors"
	"os"

	"github.com/golang-migrate/migrate/v4"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/log"
	"go-htmx-template/internal/server"
	"go-htmx-template/internal/server/router"
)

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "./db.sqlite3"
	}
	database, err := db.New(logger, dbURL)
	if err != nil {
		logger.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := database.Close(); cerr != nil {
			logger.Error("failed to close the database", "error", cerr)
		}
	}()

	if err = db.Migrate(database); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Context for graceful shutdown of background goroutines (e.g., rate
	// limiter cleanup). Cancelled when the server shuts down.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svr := server.New(
		logger,
		":"+port,
		server.WithRouter(router.New(ctx, logger, database)),
	)

	if err := svr.StartAndWait(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
