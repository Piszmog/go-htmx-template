# AGENTS.md - Development Guidelines

## Build/Test Commands
- **Development**: `air` (live reload with templ/sqlc/tailwind generation)
- **Build**: `go build -o ./tmp/main .`
- **Test all**: `go test -v ./...`
- **E2E tests**: `go test -v ./... -tags=e2e`
- **Single test**: `go test -v ./path/to/package -run TestName`
- **Generate**: `go tool templ generate -path ./components && go tool sqlc generate`
- **CSS**: `go tool go-tw -i ./styles/input.css -o ./dist/assets/css/output@dev.css`

## Code Style
- **Imports**: Standard library first, then third-party, then local packages
- **Naming**: Use Go conventions (PascalCase for exported, camelCase for unexported)
- **Error handling**: Always check errors, use `fmt.Errorf` with `%w` for wrapping
- **Logging**: Use structured logging with `slog.Logger`, include context in error messages
- **Interfaces**: Keep small and focused (e.g., `Database` interface in `db/db.go`)
- **Comments**: Document exported functions/types, use `//` for single line comments

## Project Structure
- `components/`: templ files (*.go files auto-generated, ignored by git)
- `db/`: sqlc generated code and migrations
- `server/`: HTTP handlers, middleware, routing
- `e2e/`: End-to-end tests using Playwright
- `dist/assets/`: Static assets (CSS auto-generated, ignored by git)