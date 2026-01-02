# AGENTS.md - Development Guidelines

This file contains comprehensive guidelines for AI coding agents working in this Go + HTMX template repository.

## Build/Test/Lint Commands

### Development
- **Run with live reload**: `air` (auto-generates templ, sqlc, tailwind CSS on file changes)
- **Manual build**: `go build -o ./tmp/main ./cmd/server`
- **Run without air**: `go run ./cmd/server`

### Testing
- **All tests**: `go test -v ./...`
- **With race detector**: `go test -race ./...`
- **Single package**: `go test -v ./internal/server/handler`
- **Single test**: `go test -v ./internal/server/handler -run TestHome`
- **E2E tests only**: `go test -v ./... -tags=e2e`
- **E2E single test**: `go test -v ./e2e -tags=e2e -run TestHomePage`
- **Headful E2E** (see browser): `HEADFUL=1 go test -v ./e2e -tags=e2e`
- **Different browser**: `BROWSER=firefox go test -v ./e2e -tags=e2e` (chromium, firefox, webkit)

### Linting
- **Lint all**: `golangci-lint run`
- **Lint with fixes**: `golangci-lint run --fix`
- **SQL lint**: `go tool sqlc vet`

### Code Generation
- **Generate all**: `go tool templ generate -path ./internal/components && go tool sqlc generate`
- **Templ only**: `go tool templ generate -path ./internal/components`
- **SQLC only**: `go tool sqlc generate`
- **Tailwind CSS**: `go tool go-tw -i ./styles/input.css -o ./internal/dist/assets/css/output@dev.css`

### Database Migrations
- **Create migration**: `migrate create -ext sql -dir internal/db/migrations <name>`
- **Migration files**: Creates two files: `<timestamp>_<name>.up.sql` and `<timestamp>_<name>.down.sql`
- **Up migrations**: Write schema changes in `.up.sql` (e.g., `CREATE TABLE`, `ALTER TABLE`)
- **Down migrations**: Write rollback logic in `.down.sql` (e.g., `DROP TABLE`)
- **Auto-run**: Migrations run automatically on app startup via `db.Migrate()` in `cmd/server/main.go`
- **Manual run**: Not typically needed (app handles it), but available via golang-migrate CLI if needed
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

## Templ Syntax
- **Components**: `templ ComponentName(params) { <html>content</html> }`
- **Expressions**: Use `{ variable }` for interpolation, `{ function() }` for function calls
- **Composition**: Call other components with `@ComponentName(args)`
- **All tags must be closed**: Use `<div></div>` or `<br/>` (self-closing)
- **Parameters**: Accept Go types as parameters: `templ Button(text string, disabled bool)`
- **File structure**: Package declaration, imports, then templ components
- **Generated files**: `*.go` files are auto-generated, edit only `*.templ` files

## HTMX Patterns
- **Basic requests**: `hx-get="/path"`, `hx-post="/path"`, `hx-put="/path"`, `hx-delete="/path"`
- **Triggers**: `hx-trigger="click"` (default), `hx-trigger="change"`, `hx-trigger="keyup delay:500ms"`
- **Targets**: `hx-target="#result"`, `hx-target="closest tr"`, `hx-target="next .error"`
- **Swapping**: `hx-swap="innerHTML"` (default), `hx-swap="outerHTML"`, `hx-swap="afterend"`
- **Indicators**: Add `class="htmx-indicator"` to show/hide loading states
- **Forms**: Include form values automatically, use `hx-include` for additional inputs
- **Boosting**: `hx-boost="true"` converts links/forms to AJAX requests

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
- **Server**: Uses standard library `http.ServeMux` with graceful shutdown (SIGINT handling)
- **Middleware**: Chain pattern with logging, caching, and custom middleware support
- **Handlers**: Struct-based handlers with dependency injection (logger, database)
- **Database**: Interface-based design (`db.Database`) for easy testing/mocking
- **Logging**: Structured logging with `slog`, configurable via `LOG_LEVEL`/`LOG_OUTPUT` env vars
- **Context**: Always pass `context.Context` as first parameter to functions that need it
- **Error handling**: Use `fmt.Errorf` with `%w` for error wrapping, log errors with context

## Environment Variables
- **PORT**: Server port (default: 8080)
- **LOG_LEVEL**: debug, info, warn, error (default: info)
- **LOG_OUTPUT**: text, json (default: text)
- **DB_URL**: Database file path (default: ./db.sqlite3)

## Testing
- **E2E**: Playwright tests with `//go:build e2e` tag, run with `go test -tags=e2e`
- **Test setup**: Automatic app startup, database seeding, random port allocation
- **Browser support**: Chromium (default), Firefox, WebKit via `BROWSER` env var

## Project Structure
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
- Only `e2e/` (tests) and `styles/` (build inputs) stay at root