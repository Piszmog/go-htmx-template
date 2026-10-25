package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/log"
	"go-htmx-template/internal/server"
	"go-htmx-template/internal/server/router"
	"go-htmx-template/internal/version"
)

const (
	defaultRateLimit = 50
	migrateTimeout   = 30 * time.Second
	exitUsage        = 2
)

var (
	errInvalidRateLimit         = errors.New("invalid RATE_LIMIT value")
	errInvalidTrustProxyHeaders = errors.New("invalid TRUST_PROXY_HEADERS value")
	errInvalidMigrateOnStart    = errors.New("invalid MIGRATE_ON_START value")
)

const usage = `Usage: my-app [command]

Commands:
  serve    start the HTTP server (default)
  migrate  apply the embedded migrations and exit

Flags:
  -h, --help     show this help
  -v, --version  print the version

Environment:
  PORT                 HTTP port (default 8080)
  DB_URL               database path or file: URL (default ./db.sqlite3)
  MIGRATE_ON_START     apply embedded migrations before serving (default false)
  LOG_LEVEL            debug, info, warn, error (default info)
  LOG_OUTPUT           text or json (default text)
  RATE_LIMIT           requests per minute per IP (default 50)
  TRUST_PROXY_HEADERS  trust X-Forwarded-* headers from one proxy (default false)
`

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)
	slog.SetDefault(logger)

	os.Exit(exitCode(logger, os.Args[1:], os.Stdout))
}

// exitCode runs the requested command and returns the process exit code.
func exitCode(logger *slog.Logger, args []string, stdout io.Writer) int {
	if printInfo(args, stdout) {
		return 0
	}

	command := "serve"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "serve":
		if err := run(logger); err != nil {
			logger.Error("server error", "error", err)
			return 1
		}
		return 0
	case "migrate":
		if err := runMigrate(logger); err != nil {
			logger.Error("migration error", "error", err)
			return 1
		}
		return 0
	default:
		// The message is the usage output itself; logging it too would print
		// the same line twice.
		_, _ = fmt.Fprintf(stdout, "unknown command %q\n\n%s", command, usage)
		return exitUsage
	}
}

// printInfo writes the usage or the version and reports whether it handled the
// call. Flags count wherever they appear, so `serve --help` and `migrate -h`
// print help instead of doing work.
func printInfo(args []string, stdout io.Writer) bool {
	for _, arg := range args {
		switch arg {
		case "help", "-h", "--help":
			_, _ = fmt.Fprint(stdout, usage)
			return true
		case "version", "-v", "--version":
			_, _ = fmt.Fprintln(stdout, version.Value)
			return true
		}
	}
	return false
}

func run(logger *slog.Logger) error {
	// Validate configuration before touching the database so a typo does not
	// create an empty database file.
	migrateOnStart, err := parseMigrateOnStart(os.Getenv("MIGRATE_ON_START"))
	if err != nil {
		return err
	}

	port := envOrDefault("PORT", "8080")

	rateLimit, err := parseRateLimit(os.Getenv("RATE_LIMIT"))
	if err != nil {
		return err
	}

	trustProxyHeaders, err := parseTrustProxyHeaders(os.Getenv("TRUST_PROXY_HEADERS"))
	if err != nil {
		return err
	}

	database, err := openDatabase()
	if err != nil {
		return err
	}
	defer func() {
		if cerr := database.Close(); cerr != nil {
			logger.Error("failed to close the database", "error", cerr)
		}
	}()

	if migrateOnStart {
		ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
		defer cancel()
		if err := db.Migrate(ctx, database.DB()); err != nil {
			return fmt.Errorf("running migrations on start: %w", err)
		}
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

// parseMigrateOnStart parses the MIGRATE_ON_START value.
func parseMigrateOnStart(raw string) (bool, error) {
	return parseBool(raw, errInvalidMigrateOnStart)
}

// parseTrustProxyHeaders parses the TRUST_PROXY_HEADERS value.
func parseTrustProxyHeaders(raw string) (bool, error) {
	return parseBool(raw, errInvalidTrustProxyHeaders)
}

// parseBool parses a boolean environment value, treating empty as unset and
// reporting invalid through the caller's sentinel.
func parseBool(raw string, invalid error) (bool, error) {
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%w: %s", invalid, raw)
	}
	return value, nil
}

// parseRateLimit parses the RATE_LIMIT value, falling back to the default.
func parseRateLimit(raw string) (int, error) {
	if raw == "" {
		return defaultRateLimit, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%w: %s", errInvalidRateLimit, raw)
	}
	return parsed, nil
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
