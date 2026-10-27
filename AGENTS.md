# AGENTS.md - Development Guidelines

This file contains comprehensive guidelines for AI coding agents working in this Go + HTMX template repository.

## Build/Test/Lint Commands

### Task runner
- **Entry point**: `just` — the `justfile` wraps the common commands. Run `just --list` for the recipe list
- **Recipes**: `generate` (default), `generate-go`, `generate-css`, `build`, `run`, `test`, `test-e2e`, `lint`, `migrate`
- **Overrides**: `just VERSION=v2026.10.0 MINIFY=1 generate` — flags first, then overrides, then the recipe
- **Formatting**: run `just --fmt` after editing the `justfile`; CI enforces `just --fmt --check`
- **Version**: `just` >= 1.55 (`set minimum-version`); CI and the Docker build pin 1.58.0

### Development
- **Run with live reload**: `air` (auto-generates templ, sqlc, tailwind CSS on file changes; needs `just` on `PATH`)
- **Manual build**: `just build` (`go build -o ./tmp/main ./cmd/server`)
- **Run without air**: `just run` (`go run ./cmd/server`)

### Testing
- **All tests**: `just test` — regenerates generated Go, then runs `go test -race ./...`
- **All tests without the race detector**: `go test -v ./...`
- **Single package**: `go test -v ./internal/server/handler`
- **Single test**: `go test -v ./internal/server/handler -run TestHome`
- **E2E tests only**: `just test-e2e` (`go test ./e2e -tags=e2e`)
- **E2E single test**: `go test -v ./e2e -tags=e2e -run TestHomePage`
- **Headful E2E** (see browser): `HEADFUL=1 go test -v ./e2e -tags=e2e`
- **Different browser**: `BROWSER=firefox go test -v ./e2e -tags=e2e` (chromium, firefox, webkit)

### Security Testing
- **All security tests**: `go test -v ./e2e -tags=e2e -run "TestSecurity|TestCSRF|TestRate|TestServer"`
- **Security headers**: `go test -v ./e2e -tags=e2e -run TestSecurityHeaders`
- **CSRF protection**: `go test -v ./e2e -tags=e2e -run TestCSRFProtection`
- **Rate limiting**: `go test -v ./e2e -tags=e2e -run TestRateLimiting`
- **Server timeouts**: `go test -v ./e2e -tags=e2e -run TestServerTimeouts`
- **Vulnerability scan**: `govulncheck ./...` (install: `go install golang.org/x/vuln/cmd/govulncheck@latest`)

### Linting
- **Lint all**: `just lint` (regenerates generated Go, `golangci-lint run`, `go tool sqlc vet`)
- **Lint Go only**: `golangci-lint run`
- **Lint with fixes**: `golangci-lint run --fix`
- **SQL lint**: `go tool sqlc vet`

### Code Generation
- **Generate all**: `just generate` (templ components, sqlc queries, Tailwind CSS)
- **Generated Go only**: `just generate-go`
- **Tailwind CSS**: `just generate-css`; add `VERSION=<v> MINIFY=1` before the recipe for a release build
- **Underlying commands** (what the recipes run): `go tool templ generate -path ./internal/components`,
  `go tool sqlc generate`, `go tool go-tw -i ./styles/input.css -o ./internal/dist/assets/css/output@dev.css`

### Database Migrations
- **Create migration**: `go run -tags sqlite github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 create -ext sql -dir internal/db/migrations <name>`
- **Migration files**: Creates two files: `<timestamp>_<name>.up.sql` and `<timestamp>_<name>.down.sql`
- **Up migrations**: Write schema changes in `.up.sql` (e.g., `CREATE TABLE`, `ALTER TABLE`)
- **Down migrations**: Write rollback logic in `.down.sql` (e.g., `DROP TABLE`)
- **Auto-run**: `air` runs `go run ./cmd/server migrate` automatically on each rebuild (see `.air.toml` `pre_cmd`). The server applies the embedded migrations on startup only when `MIGRATE_ON_START=true` (the Docker image sets it).
- **Manual run**: `just migrate` (up) or `go run ./cmd/server migrate down 1` (rollback); `migrate down` with no step count rolls back everything
- **Migration naming**: Use descriptive names (e.g., `add_users_table`, `add_email_to_authors`)

### Dependencies
- **Update all**: `go get -u ./...`
- **Tidy**: `go mod tidy`
- **Update tools**: `go get -u tool`

## Code Style & Conventions

### Imports
- **Order**: Standard library first, blank line, third-party packages, blank line, local packages
- **Example**:
  ```go
  import (
      "context"
      "fmt"
      "net/http"

      "github.com/a-h/templ"

      "go-htmx-template/internal/db"
      "go-htmx-template/internal/log"
  )
  ```

### Naming
- **Exported**: PascalCase (e.g., `Handler`, `Database`, `NewLogger`)
- **Unexported**: camelCase (e.g., `defaultHandler`, `getPort`)
- **Acronyms**: Use uppercase for exported (e.g., `HTML`, `DB`, `URL`), lowercase for unexported (e.g., `db`, `url`)
- **Interfaces**: Name after what they do (e.g., `Database`) or add `-er` suffix (e.g., `Handler`)

### Error Handling
- **Always check errors**: Never ignore error return values
- **Wrap errors**: Use `fmt.Errorf` with `%w` for error context: `fmt.Errorf("failed to query: %w", err)`
- **Join errors**: Use `errors.Join(err1, err2)` for multiple errors (see `internal/db/db.go:51`)
- **Log then return**: Log errors with context before returning: `h.Logger.Error("msg", "error", err)`

### Logging
- **Use structured logging**: `slog.Logger` with key-value pairs
- **Example**: `logger.Info("server started", "port", port, "env", env)`
- **Error logs**: `logger.Error("operation failed", "error", err, "context", value)`
- **Inject logger**: Pass `*slog.Logger` to structs via dependency injection (see `internal/server/handler/handler.go:13`)

### Comments
- **Document exports**: All exported functions, types, methods need doc comments
- **Format**: Start with the name: `// Handler handles requests.`
- **Single line**: Use `//` for all comments (avoid `/* */` except for package docs)

### Interface Implementation
- **Compile-time checks**: Always add compile-time interface checks for structs that implement interfaces
- **Format**: `var _ InterfaceName = (*StructName)(nil)` or `var _ InterfaceName = StructName{}` (for value receivers)
- **Placement**: Place the check immediately after the struct definition, before constructor functions
- **Example**:
  ```go
  type responseWriter struct {
      http.ResponseWriter
      statusCode int
  }
  
  // Compile-time check to ensure responseWriter implements http.ResponseWriter.
  var _ http.ResponseWriter = (*responseWriter)(nil)
  
  func newResponseWriter(w http.ResponseWriter) *responseWriter {
      // ...
  }
  ```
- **Purpose**: Catches interface compliance issues at build time rather than runtime

## Templ Syntax
- **Components**: `templ ComponentName(params) { <html>content</html> }`
- **Expressions**: Use `{ variable }` for interpolation, `{ function() }` for function calls
- **Composition**: Call other components with `@ComponentName(args)`
- **All tags must be closed**: Use `<div></div>` or `<br/>` (self-closing)
- **Parameters**: Accept Go types as parameters: `templ Button(text string, disabled bool)`
- **File structure**: Package declaration, imports, then templ components
- **Generated files**: `*.go` files are auto-generated, edit only `*.templ` files

## HTMX Patterns (htmx 4)
- **Basic requests**: `hx-get="/path"`, `hx-post="/path"`, `hx-put="/path"`, `hx-delete="/path"`
- **Triggers**: `hx-trigger="click"` (default), `hx-trigger="change"`, `hx-trigger="keyup delay:500ms"`
- **Targets**: `hx-target="#result"`, `hx-target="closest tr"`, `hx-target="next .error"`
- **Swapping**: `hx-swap="innerHTML"` (default), `hx-swap="outerHTML"`, `hx-swap="afterend"`
  - Prefer `innerMorph`/`outerMorph` over `innerHTML`/`outerHTML` when DOM state (focus, video, scroll,
    third-party widgets) must survive the swap. Morphing matches elements by `id`, so give elements you want
    to keep stable an `id`
- **Inheritance is explicit**: `hx-confirm:inherited="Are you sure?"` inherits down the tree; `:append` adds to the
  inherited value. htmx 2's implicit inheritance is off (`htmx.config.implicitInheritance`), so attributes on a
  parent do not apply to children unless marked
- **Indicators**: Add `class="htmx-indicator"` to show/hide loading states — htmx 4 ships the CSS, and it is injected
  via `document.adoptedStyleSheets`, so no `style-src 'unsafe-inline'` is needed
- **Forms**: Include form values automatically, use `hx-include` for additional inputs. `hx-get` and `hx-delete` do
  not include enclosing form inputs in htmx 4; add `hx-include="closest form"` when needed
- **Boosting**: `hx-boost="true"` converts links/forms to AJAX requests
- **Error responses swap**: htmx 4 swaps every response except `204`/`304`. Servers must return HTML for error
  statuses. Middleware uses `middleware.WriteError`, which renders `core.ErrorRegion` as an out-of-band swap into
  `#error-region`; because the response then contains only out-of-band content, htmx skips the main swap and the
  element that issued the request survives. Use `hx-status:422="target:#field-errors"` for per-element control
- **Per-request options**: `hx-config="timeout:5s"` (also `credentials`, `cache`, `redirect`, `referrer`,
  `integrity`, `validate`). The default timeout is 60s, up from "no timeout" in htmx 2
- **History**: htmx 4 stamps every history entry and, for apps using `hx-boost`/`hx-push-url`, restores back/forward
  by refetching the page and swapping it into `<body>` (no localStorage cache; an entry replaced by a full page load
  still navigates natively). `htmx.config.inlineScriptNonce` in `core/head` keeps inline scripts in swapped
  fragments executable under the document's original CSP nonce
- **Renamed in htmx 4**: `hx-disable` → `hx-ignore`, `hx-disabled-elt` → `hx-disable`, `hx-vars` → `hx-vals`,
  `hx-ext` is gone (include extension scripts directly), `hx-request` → `hx-config`. Removed outright: `hx-params`,
  `hx-disinherit`, `hx-inherit`, `hx-history`, and the `queue` modifier on `hx-trigger` (use `hx-sync`)
- **Request headers**: `HX-Trigger` is now `HX-Source` (`tag#id`), `HX-Trigger-Name` is gone, and `HX-Request-Type`
  (`partial` or `full`) is new
- **Event names**: `htmx:after:swap`, `htmx:after:settle`, `htmx:before:request`, `htmx:response:error`,
  `htmx:error` (htmx 2 used `htmx:afterSwap`, `htmx:responseError`, …)
- **Upgrading**: run `./upgrade_htmx.sh v<version>` (it pulls `dist/htmx.min.js` from npm via jsdelivr and validates
  the download), then check `internal/components/**/*.templ` against the rename lists above

## SQLC Usage
- **Query annotations**: `-- name: FunctionName :one|:many|:exec` (required for all queries)
- **Return types**: `:one` (single row), `:many` (slice), `:exec` (error only), `:execresult` (sql.Result)
- **Parameters**: Use `?` for SQLite placeholders in queries
- **Generated code**: Run `go tool sqlc generate` to create Go functions from SQL
- **File structure**: Queries in `internal/db/queries/`, migrations in `internal/db/migrations/`, generated code in `internal/db/queries/`
- **Usage pattern**: `queries := db.New(sqlDB); result, err := queries.FunctionName(ctx, params)`

## Tailwind CSS
- **Utility-first**: Use small, single-purpose classes like `text-center`, `bg-blue-500`, `p-4`
- **Responsive**: Prefix utilities with breakpoints: `sm:text-left`, `md:flex`, `lg:grid-cols-3`
- **States**: Use state prefixes: `hover:bg-blue-700`, `focus:ring-2`, `disabled:opacity-50`
- **Spacing**: Use consistent scale: `p-4` (padding), `m-2` (margin), `gap-6` (gap)
- **Colors**: Use semantic names: `bg-red-500`, `text-gray-700`, `border-blue-200`
- **Layout**: Common patterns: `flex items-center justify-between`, `grid grid-cols-2 gap-4`
- **Typography**: Size and weight: `text-xl font-bold`, `text-sm text-gray-600`

## Architecture & Patterns
- **Server**: Uses standard library `http.ServeMux` with graceful shutdown (SIGINT and SIGTERM)
- **Middleware**: Chain pattern with logging, caching, and custom middleware support
- **Handlers**: Struct-based handlers with dependency injection (logger, database)
- **Database**: Interface-based design (`db.Database`); the concrete `*queries.Queries` is not mockable, so test the DB against a temp SQLite file. `DB_URL` is resolved to an absolute path before the driver sees it (`internal/db/dsn.go`)
- **Logging**: Structured logging with `slog`, configurable via `LOG_LEVEL`/`LOG_OUTPUT` env vars
- **Context**: Always pass `context.Context` as first parameter to functions that need it
- **Error handling**: Use `fmt.Errorf` with `%w` for error wrapping, log errors with context
- **Security**: Secure-by-default with native CSRF protection (Go 1.25+), rate limiting, and security headers
  - **CSRF**: Uses `http.CrossOriginProtection` (no tokens needed, transparent protection via Sec-Fetch-Site header)
  - **Rate Limiting**: Per-IP token bucket algorithm, 50 requests/minute default, in-memory storage with auto-cleanup
  - **Security Headers**: X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy, HSTS
  - **Server Hardening**: Timeout configurations (ReadHeaderTimeout, IdleTimeout, MaxHeaderBytes) prevent slowloris attacks

## Environment Variables
- **PORT**: Server port (default: 8080)
- **LOG_LEVEL**: debug, info, warn, error (default: info; case-insensitive)
- **LOG_OUTPUT**: text, json (default: text; case-insensitive)
- **DB_URL**: Database file path or `file:` URL; relative paths resolve against the working directory (default: ./db.sqlite3)
- **MIGRATE_ON_START**: Apply embedded migrations before serving (default: false)
- **RATE_LIMIT**: Requests per minute per IP (default: 50)
- **TRUST_PROXY_HEADERS**: Trust `X-Forwarded-For`/`X-Real-IP` from a single proxy (default: false)

## Testing
- **E2E**: Playwright tests with `//go:build e2e` tag, run with `go test -tags=e2e`
- **Test setup**: Automatic app startup, database seeding, random port allocation
- **Browser support**: Chromium (default), Firefox, WebKit via `BROWSER` env var

## Stacked PRs
- Multi-part work goes in a `gh stack`: branches follow `<area>/<slug>` and each PR is based on the layer below
- `gh stack init <bottom>` then `gh stack add <next>`; keep one commit per layer so `submit --auto` takes the PR title from the commit subject
- `gh stack submit --auto --open` opens the PRs; merge with `gh stack merge <top-pr> --yes` (`gh pr merge` cannot merge a stack)
- CI runs per PR against its own branch, so the top layer is validated on top of the layers below it

## Project Structure
- `justfile`: Task runner recipes (generate, build, test, lint, migrate)
- `cmd/server/`: Application entrypoint with main.go
- `internal/`: All implementation packages (prevents external imports)
  - `components/`: templ files (*.go files auto-generated, ignored by git)
  - `db/`: sqlc generated code and migrations
  - `server/`: HTTP handlers, middleware, routing
  - `dist/`: Embedded static assets (CSS auto-generated, ignored by git)
  - `log/`: Structured logging utilities
  - `version/`: Build-time version info (set via ldflags)
- `e2e/`: End-to-end tests using Playwright (external to app)
- `styles/`: CSS source files (input for Tailwind)

## Why `internal/` Package?
All application code lives in `internal/` following Go's official server project layout:
- Prevents external packages from importing implementation details
- Signals this is a server application, not a reusable library
- Follows [go.dev/doc/modules/layout](https://go.dev/doc/modules/layout) "Server project" pattern
- `cmd/server/` contains the application entrypoint
- Only `e2e/` (tests), `styles/` (build inputs), and the `justfile` stay at root
