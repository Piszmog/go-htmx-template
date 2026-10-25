package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/log"
	"go-htmx-template/internal/server"
	"go-htmx-template/internal/server/router"
)

const (
	defaultRateLimit = 50
	migrateTimeout   = 30 * time.Second
)

var (
	errInvalidRateLimit         = errors.New("invalid RATE_LIMIT value")
	errInvalidTrustProxyHeaders = errors.New("invalid TRUST_PROXY_HEADERS value")
	errInvalidMigrateOnStart    = errors.New("invalid MIGRATE_ON_START value")
)

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)
	slog.SetDefault(logger)

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := runMigrate(logger); err != nil {
			logger.Error("migration error", "error", err)
			os.Exit(1)
		}
		return
	}

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

	migrateOnStart, err := parseMigrateOnStart(os.Getenv("MIGRATE_ON_START"))
	if err != nil {
		return err
	}
	if migrateOnStart {
		ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
		defer cancel()
		if err := db.Migrate(ctx, database.DB()); err != nil {
			return fmt.Errorf("running migrations on start: %w", err)
		}
	}

	port := envOrDefault("PORT", "8080")
	rateLimit, err := parseRateLimit()
	if err != nil {
		return err
	}

	trustProxyHeaders, err := parseTrustProxyHeaders()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svr := server.New(
		logger,
		":"+port,
		server.WithRouter(router.New(ctx, logger, database, rateLimit, trustProxyHeaders)),
	)

	return svr.StartAndWait()
}

func openDatabase() (db.Database, error) {
	url := envOrDefault("DB_URL", "./db.sqlite3")
	return db.New(url)
}

func runMigrate(logger *slog.Logger) error {
	database, err := openDatabase()
	if err != nil {
		return err
	}
	defer func() {
		if cerr := database.Close(); cerr != nil {
			logger.Error("failed to close the database", "error", cerr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()
	if err := db.Migrate(ctx, database.DB()); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

func parseMigrateOnStart(raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%w: %s", errInvalidMigrateOnStart, raw)
	}
	return value, nil
}

func parseTrustProxyHeaders() (bool, error) {
	raw := os.Getenv("TRUST_PROXY_HEADERS")
	if raw == "" {
		return false, nil
	}
	trusted, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%w: %s", errInvalidTrustProxyHeaders, raw)
	}
	return trusted, nil
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
