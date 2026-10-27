# Task runner entry point. Run `just --list` to see recipes.

set shell := ["bash", "-euo", "pipefail", "-c"]
set minimum-version := "1.55.0"

# Version stamped into the generated CSS filename.
VERSION := "dev"
# Tailwind CSS version go-tw downloads. go-tw defaults to "latest", which would
# let the generated bundle change with no commit behind it; pin it and bump by hand.
TAILWIND_VERSION := "v4.3.3"
# Non-empty and not "0" minifies the generated CSS.
MINIFY := ""

# Regenerate everything a build needs (default recipe).
default: generate

# Regenerate templ components, sqlc queries, and Tailwind CSS.
generate: generate-go generate-css

# Generate Go code from templ files.
generate-templ:
    go tool templ generate -path ./internal/components

# Generate Go code from SQL queries.
generate-sqlc:
    go tool sqlc generate

# Generate Go code from templ files and SQL queries.
generate-go: generate-templ generate-sqlc

# Build the Tailwind CSS bundle for VERSION.
generate-css:
    rm -f ./internal/dist/assets/css/output@*.css
    go tool go-tw -i ./styles/input.css \
        -o ./internal/dist/assets/css/output@{{ VERSION }}.css \
        -version {{ TAILWIND_VERSION }} \
        {{ if MINIFY == "" { "" } else if MINIFY == "0" { "" } else { "--minify" } }}

# Build the server binary into ./tmp/main.
build: generate
    go build -o ./tmp/main ./cmd/server

# Run the generated binary.
run: build
    ./tmp/main

# Run unit tests with the race detector.
test: generate-go
    go test -race ./...

# Run the Playwright end-to-end tests.
test-e2e: generate
    go test ./e2e -tags=e2e

# Lint Go and SQL.
lint: generate-go
    golangci-lint run
    go tool sqlc vet

# Apply pending database migrations.
migrate: generate-go
    go run ./cmd/server migrate
