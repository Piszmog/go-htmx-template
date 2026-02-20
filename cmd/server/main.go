package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/log"
	"go-htmx-template/internal/server"
	"go-htmx-template/internal/server/router"
)

var errInvalidRateLimit = errors.New("invalid RATE_LIMIT value")

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	if err := run(logger); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "./db.sqlite3"
	}
	database, err := db.New(logger, dbURL)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := database.Close(); cerr != nil {
			logger.Error("failed to close the database", "error", cerr)
		}
	}()

	if err = db.Migrate(database); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	rateLimit := 50
	if rateLimitStr := os.Getenv("RATE_LIMIT"); rateLimitStr != "" {
		parsed, err := strconv.Atoi(rateLimitStr)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("%w: %s", errInvalidRateLimit, rateLimitStr)
		}
		rateLimit = parsed
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svr := server.New(
		logger,
		":"+port,
		server.WithRouter(router.New(ctx, logger, database, rateLimit)),
	)

	return svr.StartAndWait()
}
