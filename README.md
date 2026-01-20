# Go + HTMX Template

This is an _opinionated_ template repository that comes with everything you need to build a Web Application using Go (with templ) and HTMX. 

The template comes with a basic structure of using a SQL DB (`sqlc`), E2E testing (playwright), and styling (tailwindcss).

## Getting Started

In the top right, select the dropdown __Use this template__ and select __Create a new repository__.

Once cloned, run the `update_module.sh` script to change the module to your module name.

```shell
./update_module "github.com/me/my-new-module"
```

Once ran, delete the script.

Then you can upgrade your module dependencies with

```shell
go mod tidy
go get -u ./...
go get -u tool
# To upgrade test dependencies, run
go get -u all
```

Finally, you can proceed to generate sqlc and templ files

```shell
go tool sqlc generate
go tool templ generate -path ./internal/components
```

## Run

`air` is the primary way to run the applications for local development. It watches for file changes. When a file changes, it will rebuild and re-run the application.

When the application is running, go to http://localhost:8080/

## Environment Variables

The application can be configured using environment variables. For local development, copy `.env.example` to `.env` and customize as needed.

### Available Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `LOG_OUTPUT` | `text` | Log format: `text` or `json` |
| `DB_URL` | `./db.sqlite3` | Path to SQLite database file |
| `RATE_LIMIT` | `50` | Requests per minute per IP address |

### Example

```bash
# .env
PORT=3000
LOG_LEVEL=debug
DB_URL=/data/myapp.db
```

## Endpoints

### Health Check

The application provides a basic health check endpoint:

- **GET /health** - Returns `200 OK` with `{"version":"dev"}`

This endpoint is suitable for basic liveness checks from load balancers or monitoring systems.

### Prerequisites

- Install [air](https://github.com/air-verse/air#installation)

`templ`, `sqlc`, and `tailwindcss` (via [`go-tw`](https://github.com/Piszmog/go-tw)) are included as `go tool` directives. When running
the application for the first time, it may take a little time as `templ`, `sqlc` and `go-tw` are being downloaded and installed.

### air

`air` has been configured with the file `.air.toml` to allow live reloading of the application 
when a file changes.

To run, install `air`

```shell
go install github.com/air-verse/air@latest
```

Then simply run the command

```shell
air
```

## Technologies

A few different technologies are configured to help getting off the ground easier.

- [sqlc](https://sqlc.dev/) for database layer
  - Stubbed to use SQLite
  - This can be easily swapped with [sqlx](https://jmoiron.github.io/sqlx/)
- [Tailwind CSS](https://tailwindcss.com/) for styling
  - Output is generated with the [CLI](https://tailwindcss.com/docs/installation/tailwind-cli)
- [templ](https://templ.guide/) for creating HTML
- [HTMX](https://htmx.org/) for HTML interaction
  - The script `upgrade_htmx.sh` is available to make upgrading easier
  - Already included in this template
- [air](https://github.com/air-verse/air) for live reloading of the application.
- [golang migrate](https://github.com/golang-migrate/migrate) for DB migrations.
- [playwright-go](https://github.com/playwright-community/playwright-go) for E2E testing.

## Security

This template comes with comprehensive security features enabled by default to help you build secure applications from the start.

### CSRF Protection

The application uses Go 1.25+'s native `http.CrossOriginProtection` for CSRF defense. This provides transparent protection without requiring CSRF tokens in your forms.

- **How it works:** Validates requests using the `Sec-Fetch-Site` header
- **What's protected:** POST, PUT, DELETE, PATCH requests
- **Configuration:** Enabled by default in the middleware chain
- **Implementation:** See `internal/server/middleware/csrf.go`

### Rate Limiting

Per-IP rate limiting prevents abuse and helps protect against DoS attacks.

- **Default limit:** 50 requests per minute per IP address
- **Algorithm:** Token bucket with in-memory storage
- **Auto-cleanup:** Inactive IP limiters are cleaned up every 10 minutes
- **Configuration:** Set `RATE_LIMIT` environment variable to customize
- **Headers:** Returns `Retry-After: 60` when limit exceeded
- **Implementation:** See `internal/server/middleware/ratelimit.go`

### Security Headers

The following security headers are automatically set on all responses:

- `X-Frame-Options: DENY` - Prevents clickjacking attacks
- `X-Content-Type-Options: nosniff` - Prevents MIME-type sniffing
- `Referrer-Policy: strict-origin-when-cross-origin` - Controls referrer information
- `Permissions-Policy: geolocation=(), microphone=(), camera=()` - Restricts browser features
- `Strict-Transport-Security` - Enforces HTTPS (when using TLS)

**Implementation:** See `internal/server/middleware/security.go`

### Server Hardening

The HTTP server is configured with timeouts to prevent slowloris and similar attacks:

- `ReadHeaderTimeout: 5s` - Maximum time to read request headers
- `ReadTimeout: 15s` - Maximum time to read entire request
- `WriteTimeout: 15s` - Maximum time to write response
- `IdleTimeout: 60s` - Maximum idle connection time
- `MaxHeaderBytes: 1MB` - Maximum header size

**Implementation:** See `internal/server/server.go`

### Panic Recovery

A recovery middleware catches panics and logs them with full stack traces, preventing the server from crashing while providing debugging information.

**Implementation:** See `internal/server/middleware/recovery.go`

### Testing

Comprehensive security tests are included in `e2e/security_test.go`:
- Security headers validation
- CSRF protection enforcement
- Rate limiting behavior
- Server timeout configurations

## Structure

```text
.
├── .air.toml
├── .github
│   └── workflows
│       ├── ci.yml
│       └── release.yml
├── .gitignore
├── .goreleaser.yaml
├── AGENTS.md
├── Dockerfile
├── cmd
│   └── server
│       └── main.go
├── internal
│   ├── components
│   │   ├── core
│   │   │   └── html.templ
│   │   └── home
│   │       └── home.templ
│   ├── db
│   │   ├── db.go
│   │   ├── local.go
│   │   ├── migrations
│   │   │   ├── 20240407203525_init.down.sql
│   │   │   └── 20240407203525_init.up.sql
│   │   └── queries
│   │       ├── db.go
│   │       ├── models.go
│   │       ├── query.sql
│   │       └── query.sql.go
│   ├── dist
│   │   ├── assets
│   │   │   ├── css
│   │   │   │   └── output@dev.css
│   │   │   └── js
│   │   │       └── htmx@v2.0.7.min.js
│   │   └── dist.go
│   ├── log
│   │   └── log.go
│   ├── server
│   │   ├── handler
│   │   │   ├── handler.go
│   │   │   ├── health.go
│   │   │   ├── health_test.go
│   │   │   └── home.go
│   │   ├── middleware
│   │   │   ├── cache.go
│   │   │   ├── csrf.go
│   │   │   ├── logging.go
│   │   │   ├── logging_test.go
│   │   │   ├── middleware.go
│   │   │   ├── ratelimit.go
│   │   │   ├── recovery.go
│   │   │   ├── recovery_test.go
│   │   │   ├── response_writer.go
│   │   │   └── security.go
│   │   ├── router
│   │   │   └── router.go
│   │   └── server.go
│   └── version
│       └── version.go
├── e2e
│   ├── e2e_test.go
│   ├── home_test.go
│   ├── security_test.go
│   └── testdata
│       └── seed.sql
├── styles
│   └── input.css
├── go.mod
├── go.sum
├── sqlc.yml
├── update_module.sh
└── upgrade_htmx.sh
```

### Agents

At the root of the project is the file `AGENTS.md`. It is designed to help agents better understand the project and help you in your development.

### cmd/

This directory contains the application entrypoints. The `server/` subdirectory contains `main.go` which starts the HTTP server. This follows Go's standard project layout for applications.

### internal/

All application implementation code lives in the `internal/` directory. This prevents external packages from importing implementation details and follows the official Go project layout for server applications as documented at [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout).

### Components

This is where `templ` files live in `internal/components/`. Anything you want to render to the user goes here. Note, all `*.go` files will be ignored by `git` (configured in `.gitignore`).

### DB

This is the directory in `internal/db/` that `sqlc` generates to. Update `queries.sql` to build 
your database operations.

This project uses [golang migrate](https://github.com/golang-migrate/migrate) for DB 
migrations. `sqlc` uses the `internal/db/migrations` directory to generate DB tables. `cmd/server/main.go` calls `db.Migrate(..)` to automatically migrate the DB. To add migration
call the following command,

```shell
migrate create -ext sql -dir internal/db/migrations <name of migration>
```

#### Example Connection to Turso

If you want to connect to a remote Database, like [Turso](https://turso.tech/), you can create a struct that implements `Database`.

```golang
package db

import (
	"database/sql"
	"log/slog"

	"go-htmx-template/internal/db/queries"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type RemoteDB struct {
	logger  *slog.Logger
	db      *sql.DB
	queries *queries.Queries
}

var _ Database = (*RemoteDB)(nil)

func (d *RemoteDB) DB() *sql.DB {
	return d.db
}

func (d *RemoteDB) Queries() *queries.Queries {
	return d.queries
}

func (d *RemoteDB) Logger() *slog.Logger {
	return d.logger
}

func (d *RemoteDB) Close() error {
	return d.db.Close()
}

func newRemoteDB(logger *slog.Logger, name string, token string) (*RemoteDB, error) {
	db, err := sql.Open("libsql", "libsql://"+name+".turso.io?authToken="+token)
	if err != nil {
		return nil, err
	}
	return &RemoteDB{logger: logger, db: db, queries: queries.New(db)}, nil
}
```

### Dist

This is where your assets live in `internal/dist/`. Any Javascript, images, or styling needs to go in the 
`internal/dist/assets` directory. The directory will be embedded into the application.

Note, the `internal/dist/assets/css` will be ignored by `git` (configured in `.gitignore`) since the 
files that are written to this directory are done by the Tailwind CSS CLI. Custom styles should
go in the `styles/input.css` file at the root level.

### E2E

To test the UI, the `e2e` directory contains the Go tests for performing End to end testing. To
run the tests, run the command

```shell
go test -v ./... -tags=e2e
```

The end to end tests, will start up the app, on a random port, seeding the database using the 
`seed.sql` file. Once the tests are complete, the app will be stopped.

The E2E tests use Playwright (Go) for better integration into the Go tooling.

### Log

This contains helper function to create a `slog.Logger`. Log level and output type can be set
with then environment variables `LOG_LEVEL` and `LOG_OUTPUT`. The logger will write to 
`stdout`.

### Server

This contains everything related to the HTTP server in `internal/server/`. 

The server is configured with:
- **Graceful shutdown** - Handles `SIGINT` with 10-second grace period
- **Timeout protection** - ReadHeaderTimeout, ReadTimeout, WriteTimeout, IdleTimeout
- **Size limits** - MaxHeaderBytes prevents oversized header attacks
- **Functional options** - Customize timeouts via `WithReadTimeout`, `WithWriteTimeout`, etc.

See `internal/server/server.go` for configuration details.

#### Router

This package sets up the routing for the application, such as the `/assets/` path and `/` path.
It uses the standard library's mux for routing. You can easily swap out for other HTTP 
routers such as [gorilla/mux](https://github.com/gorilla/mux).

#### Middleware

This package contains middleware applied to all routes in a chain:

1. **Recovery** - Catches panics and logs stack traces
2. **Logging** - Structured request/response logging with duration and status
3. **Security** - Sets security headers (X-Frame-Options, CSP, etc.)
4. **RateLimit** - Per-IP rate limiting (configurable via `RATE_LIMIT` env var)
5. **CSRF** - Cross-origin request protection using Go 1.25+ native implementation
6. **Cache** - Applied only to static assets under `/assets/`

See `internal/server/router/router.go` for the middleware chain configuration.

#### Handler

This package contains HTTP handlers for routes:

- **handler.go** - Base handler struct with logger and database dependencies
- **home.go** - Homepage handler rendering templ components
- **health.go** - Health check endpoint (`/health`) returning version info
- **health_test.go** - Unit tests for handler logic

Handlers use dependency injection for testability and follow standard `http.HandlerFunc` signature.

### Styles

This contains the `input.css` at the root level that the Tailwind CSS CLI uses to generate your output CSS. 
Update `input.css` with any custom CSS you need and it will be included in the output CSS.

### Version

This package in `internal/version/` allows you to set a version at build time. If not set, the version defaults to 
`dev`. To set the version run the following command,

```shell
go build -o ./app -ldflags="-X go-htmx-template/internal/version.Value=1.0.0" ./cmd/server
```

## Github Workflow

The repository comes with two Github workflows as well. One called `ci.yml` that lints and 
tests your code. The other called `release.yml` that creates a tag, GitHub Release, run [GoReleaser](https://goreleaser.com/) to build and 
attach all the binaries, and published the docker image. See release [v1.0.2](https://github.com/Piszmog/go-htmx-template/releases/tag/v1.0.2) as an example.

