# Go + HTMX Template

This is an _opinionated_ template repository that comes with everything you need to build a Web Application using Go (with templ) and HTMX. 

The template comes with a basic structure of using a SQL DB (`sqlc`), E2E testing (playwright), and styling (tailwindcss).

## Getting Started

In the top right, select the dropdown __Use this template__ and select __Create a new repository__.

Once cloned, run the `update_module.sh` script to change the module to your module name.

```shell
./update_module.sh "github.com/me/my-new-module"
```

Once ran, delete the script.

Then you can upgrade your module dependencies with

```shell
go mod tidy
go get -u ./...
go get -u tool
```

Finally, generate the templ components, sqlc queries, and Tailwind CSS:

```shell
make generate
```

## Run

`air` is the primary way to run the applications for local development. It watches for file changes. When a file changes, it will rebuild and re-run the application. On each rebuild, `air` also applies any pending DB migrations by running the `migrate` subcommand.

If running the binary directly (not via `air`), generate the artifacts and run migrations first:

```shell
make generate
go run ./cmd/server migrate
```

`make generate` regenerates the templ components, the sqlc queries, and the Tailwind CSS. It accepts `VERSION` (default `dev`, used in the CSS filename) and `MINIFY=1` for production builds.

The migrations are embedded in the binary, so `go run ./cmd/server migrate down <steps>` rolls back
without any external tooling; omit the step count to roll back everything.

When the application is running, go to http://localhost:8080/

### Prerequisites

- Install [air](https://github.com/air-verse/air#installation)

`templ`, `sqlc`, and `tailwindcss` (via [`go-tw`](https://github.com/Piszmog/go-tw)) are included as `go tool` directives. When running
the application for the first time, it may take a little time as `templ`, `sqlc` and `go-tw` are being downloaded and installed.

The `Makefile` wraps the common commands: `make generate`, `make build`, `make test`, `make test-e2e`, `make lint`, and `make migrate`.

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

## Environment Variables

The application reads environment variables from the process; it does **not** load `.env` automatically. Export them in your shell or use a tool such as [direnv](https://direnv.net/). The example file documents the available variables.

### Available Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |
| `LOG_OUTPUT` | `text` | Log format: `text` or `json` |
| `DB_URL` | `./db.sqlite3` | Path or `file:` URL for the SQLite database |
| `MIGRATE_ON_START` | `false` | Apply embedded migrations before serving |
| `RATE_LIMIT` | `50` | Requests per minute per IP address |
| `TRUST_PROXY_HEADERS` | `false` | Trust `X-Forwarded-For` / `X-Real-IP` from a single reverse proxy |

### Example

```bash
export PORT=3000
export LOG_LEVEL=debug
export DB_URL=/data/myapp.db
```

## Endpoints

### Health and Readiness

- **GET /health** - Liveness. Returns `200 OK` with the build version in a JSON body
  (`{"version":"dev"}` when built without ldflags) and never touches the database.
- **GET /readyz** - Readiness. Pings the database and returns `200 OK` with `{"status":"ok"}`, or
  `503 Service Unavailable` with `{"status":"unavailable"}` when the database cannot be reached.

Use `/health` for liveness probes and `/readyz` for readiness probes. Neither endpoint consumes the
rate-limit budget.

## Deploy

The Docker image embeds the migrations and runs them on startup:

```shell
docker build -t go-htmx-template .
docker run --rm -p 8080:8080 -v go-htmx-template-data:/home/nonroot go-htmx-template
```

- **Explicit migrations:** `./my-app migrate` applies the embedded migrations and exits. Release archives include the same command. In Docker, run `docker run --rm -v go-htmx-template-data:/home/nonroot go-htmx-template migrate`.
- **Automatic migrations:** `MIGRATE_ON_START=true` runs migrations before the server starts. The image sets this by default.
- **Multi-instance deployments:** run migrations once as a separate job or init container instead of enabling `MIGRATE_ON_START` on every replica.
- **Persistence:** mount a volume at `/home/nonroot` (or point `DB_URL` at a mounted path). Without a volume, the SQLite database is lost when the container is removed.

## Technologies

A few different technologies are configured to help getting off the ground easier.

- [sqlc](https://sqlc.dev/) for database layer
  - Stubbed to use SQLite
  - This can be easily swapped with [sqlx](https://jmoiron.github.io/sqlx/)
- [Tailwind CSS](https://tailwindcss.com/) for styling
  - Output is generated with the [CLI](https://tailwindcss.com/docs/installation/tailwind-cli)
- [templ](https://templ.guide/) for creating HTML
- [HTMX](https://htmx.org/) for HTML interaction
  - Vendored at `internal/dist/assets/js/htmx@v4.0.0.min.js`
  - The script `upgrade_htmx.sh` is available to make upgrading easier; it pulls the bundle from npm (via jsdelivr) and validates it before replacing anything
- [air](https://github.com/air-verse/air) for live reloading of the application.
- [golang migrate](https://github.com/golang-migrate/migrate) for DB migrations. The migrations are embedded and applied by the `migrate` subcommand or `MIGRATE_ON_START`.
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
- **Scope:** Dynamic routes only; `/assets/`, `/health`, and `/readyz` do not consume the budget
- **Algorithm:** Token bucket with in-memory storage
- **Per-instance:** Each replica enforces its own budget; use a shared store for multi-instance deployments
- **Auto-cleanup:** Inactive IP limiters are cleaned up every 10 minutes
- **Configuration:** Set `RATE_LIMIT` environment variable to customize
- **Headers:** Returns `Retry-After` computed from the configured rate when the limit is exceeded
- **Implementation:** See `internal/server/middleware/ratelimit.go`

### Security Headers

The following security headers are automatically set on all responses:

- `X-Frame-Options: DENY` - Prevents clickjacking attacks
- `X-Content-Type-Options: nosniff` - Prevents MIME-type sniffing
- `Referrer-Policy: strict-origin-when-cross-origin` - Controls referrer information
- `Permissions-Policy: geolocation=(), microphone=(), camera=()` - Restricts browser features
- `Content-Security-Policy` - Nonce-based `script-src`, no `'unsafe-inline'` for scripts or styles
  - htmx 4 injects its indicator CSS through constructable stylesheets, which `style-src` does not govern
  - Error responses are rendered as HTML fragments and swapped out of band into `#error-region`
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
├── cmd/server/              # application entrypoint
├── internal/
│   ├── components/          # templ components (generated *.go is gitignored)
│   ├── db/                  # sqlc queries and migrations
│   ├── dist/                # embedded assets (CSS is generated)
│   ├── log/                 # slog setup
│   ├── server/              # handlers, middleware, router, HTTP server
│   └── version/             # build-time version
├── e2e/                     # Playwright end-to-end tests
├── styles/input.css         # Tailwind source
├── Dockerfile
├── Makefile                 # generate, build, test, lint, migrate targets
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

`sqlc` generates `internal/db/queries` from `internal/db/queries/query.sql` and the schema in
`internal/db/migrations`. Add queries to `query.sql`, then run `make generate`.

This project uses [golang migrate](https://github.com/golang-migrate/migrate) for DB migrations.
The migration files are embedded in the binary, so the same migrations run from the `migrate`
subcommand and from `MIGRATE_ON_START`:

```shell
go run ./cmd/server migrate          # apply pending migrations (or: make migrate)
go run ./cmd/server migrate down 1   # roll back the most recent migration
go run ./cmd/server migrate down     # roll back everything
```

`MIGRATE_ON_START=true` applies pending migrations before the server starts; the Docker image sets
it by default. `air` runs `migrate` on every rebuild. See [Deploy](#deploy) for when to prefer
each option.

To create a new migration pair:

```shell
go run -tags sqlite github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
  create -ext sql -dir internal/db/migrations <name>
```

The generated files are named `<timestamp>_<name>.up.sql` and `<timestamp>_<name>.down.sql`.

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
- **Graceful shutdown** - Handles `SIGINT` and `SIGTERM` with a 10-second grace period
- **Timeout protection** - ReadHeaderTimeout, ReadTimeout, WriteTimeout, IdleTimeout
- **Size limits** - MaxHeaderBytes prevents oversized header attacks
- **Structured errors** - `http.Server.ErrorLog` is routed through `slog`
- **Options** - `WithRouter` sets the handler; timeout values are package constants

See `internal/server/server.go` for configuration details.

#### Router

This package sets up the routing for the application, such as the `/assets/` path and `/` path.
It uses the standard library's mux for routing. You can easily swap out for other HTTP 
routers such as [gorilla/mux](https://github.com/gorilla/mux).

#### Middleware

This package contains middleware applied to all routes in a chain:

1. **Logging** - Structured request/response logging with duration and status
2. **Compress** - Gzips compressible responses for clients that accept it
3. **Recovery** - Catches panics, logs the stack trace, and renders a 500
4. **BodyLimit** - Caps request bodies at 1 MiB
5. **SecurityHeaders** - Sets the headers every response carries (COOP, CORP, HSTS, baseline CSP)
6. **CSRF** - Cross-origin request protection using Go 1.25+ native implementation

`CSP` is deliberately not in that chain: it wraps only the dynamic handler and replaces the baseline CSP
with one that trusts a fresh nonce, which templ stamps onto the inline htmx configuration script. Static
assets and JSON endpoints never pay for minting a nonce they cannot use.

`RateLimit` wraps only the dynamic handler (`/` and `/count`); `/health`, `/readyz`, and `/assets/` are exempt. `CacheMiddleware` applies only to static assets under `/assets/`.

See `internal/server/router/router.go` for the middleware chain configuration.

#### Handler

This package contains HTTP handlers for routes:

- **handler.go** - Base handler struct with logger and database dependencies
- **home.go** - Homepage handler and the `/count` counter fragment
- **health.go** - Liveness endpoint (`/health`) returning version info
- **ready.go** - Readiness endpoint (`/readyz`) that pings the database

Handlers use dependency injection for testability and follow standard `http.HandlerFunc` signature.

### Styles

This contains the `input.css` at the root level that the Tailwind CSS CLI uses to generate your output CSS. 
Update `input.css` with any custom CSS you need and it will be included in the output CSS.

### Version

This package in `internal/version/` allows you to set a version at build time. If not set, the version defaults to
`dev`. The version is also part of the generated CSS filename (`output@<version>.css`), so generate the assets with
the same version before building:

```shell
make generate VERSION=v2026.10.0
go build -o ./app -ldflags="-X go-htmx-template/internal/version.Value=v2026.10.0" ./cmd/server
```

GoReleaser takes the version from the git tag (`{{ .Tag }}`), so releases, archives, and container images all report
the same CalVer string.

## Github Workflow

The repository comes with two Github workflows. `ci.yml` lints and tests changes, dry-runs the release build, and
builds pull request images. `release.yml` runs after `ci.yml` succeeds for a push to `main`: it calculates the next
[CalVer](https://calver.org/) tag (`vYYYY.MM.MICRO`, e.g. `v2026.10.0`), tags the commit CI tested, runs
[GoReleaser](https://goreleaser.com/) to build and attach all the binaries to the GitHub Release, and publishes the
multi-arch Docker image to GHCR. Manual version bumps and semver tags are gone; every merge whose CI run succeeds
gets its own release. Micro numbers are claimed when the tag is created, so concurrent merges may not receive them in
merge order.

See release [v1.0.2](https://github.com/Piszmog/go-htmx-template/releases/tag/v1.0.2) for the format used before the
CalVer switch.

