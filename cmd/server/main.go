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

const defaultRateLimit = 50

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
	database, err := openDatabase()
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

	port := envOrDefault("PORT", "8080")
	rateLimit, err := parseRateLimit()
	if err != nil {
		return err
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

func openDatabase() (db.Database, error) {
	url := envOrDefault("DB_URL", "./db.sqlite3")
	return db.New(url)
}

func parseRateLimit() (int, error) {
	rateLimitStr := os.Getenv("RATE_LIMIT")
	if rateLimitStr == "" {
		return defaultRateLimit, nil
	}
	parsed, err := strconv.Atoi(rateLimitStr)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%w: %s", errInvalidRateLimit, rateLimitStr)
	}
	return parsed, nil
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
