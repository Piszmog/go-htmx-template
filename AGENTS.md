# AGENTS.md - Development Guidelines

## Build/Test Commands
- **Development**: `air` (live reload with templ/sqlc/tailwind generation)
- **Build**: `go build -o ./tmp/main ./cmd/server`
- **Test all**: `go test -v ./...`
- **E2E tests**: `go test -v ./... -tags=e2e`
- **Single test**: `go test -v ./path/to/package -run TestName`
- **Generate**: `go tool templ generate -path ./internal/components && go tool sqlc generate`
- **CSS**: `go tool go-tw -i ./styles/input.css -o ./internal/dist/assets/css/output@dev.css`

## Code Style
- **Imports**: Standard library first, then third-party, then local packages
- **Naming**: Use Go conventions (PascalCase for exported, camelCase for unexported)
- **Error handling**: Always check errors, use `fmt.Errorf` with `%w` for wrapping
- **Logging**: Use structured logging with `slog.Logger`, include context in error messages
- **Interfaces**: Keep small and focused (e.g., `Database` interface in `internal/db/db.go`)
- **Comments**: Document exported functions/types, use `//` for single line comments

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