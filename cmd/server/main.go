package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
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
	errInvalidMigrateSteps      = errors.New("invalid migrate step count")
	errUnexpectedMigrateArg     = errors.New("unexpected migrate argument")
	errUnknownMigrateCommand    = errors.New("unknown migrate command")
)

const usage = `Usage: my-app [command]

Commands:
  serve                      start the HTTP server (default)
  migrate [up]               apply pending migrations and exit
  migrate down [steps]       roll back the most recent migrations (all when omitted)

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
		// A trailing argument is a typo like `serve --prt 9090`; starting the
		// server anyway would hide it.
		if len(args) > 1 {
			return usageError(stdout, "unexpected argument", args[1:])
		}
		if err := run(logger); err != nil {
			logger.Error("server error", "error", err)
			return 1
		}
		return 0
	case "migrate":
		if err := runMigrate(logger, args[1:]); err != nil {
			logger.Error("migration error", "error", err)
			return 1
		}
		return 0
	default:
		return usageError(stdout, "unknown command", args)
	}
}

// usageError reports a bad command line and returns the usage exit code. The
// usage text is the message, so logging it would print the same line twice.
func usageError(stdout io.Writer, problem string, args []string) int {
	_, _ = fmt.Fprintf(stdout, "%s %q\n\n%s", problem, strings.Join(args, " "), usage)
	return exitUsage
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

// runMigrate applies or rolls back the embedded migrations. args is everything
// after the "migrate" command.
func runMigrate(logger *slog.Logger, args []string) error {
	options, err := parseMigrateOptions(args)
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

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	if options.down {
		return db.MigrateDown(ctx, database.DB(), options.steps)
	}

	return db.Migrate(ctx, database.DB())
}

const (
	migrateUp   = "up"
	migrateDown = "down"
)

// migrateOptions is the parsed form of the arguments after "migrate".
type migrateOptions struct {
	down  bool
	steps int
}

func parseMigrateOptions(args []string) (migrateOptions, error) {
	direction := migrateUp
	var rest []string
	if len(args) > 0 {
		direction, rest = args[0], args[1:]
	}

	switch direction {
	case migrateUp:
		if len(rest) > 0 {
			return migrateOptions{}, unexpectedArg(rest)
		}
		return migrateOptions{}, nil
	case migrateDown:
		if len(rest) > 1 {
			return migrateOptions{}, unexpectedArg(rest[1:])
		}
		options := migrateOptions{down: true}
		if len(rest) == 1 {
			steps, err := strconv.Atoi(rest[0])
			// Zero would silently mean "roll back everything", so reject it.
			if err != nil || steps < 1 {
				return migrateOptions{}, fmt.Errorf("%w: %s", errInvalidMigrateSteps, rest[0])
			}
			options.steps = steps
		}
		return options, nil
	default:
		return migrateOptions{}, fmt.Errorf("%w: %s", errUnknownMigrateCommand, direction)
	}
}

// unexpectedArg reports the arguments left over after a valid migrate command,
// so a typo like `migrate up 3` fails instead of silently doing nothing.
func unexpectedArg(rest []string) error {
	return fmt.Errorf("%w: %s", errUnexpectedMigrateArg, strings.Join(rest, " "))
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
