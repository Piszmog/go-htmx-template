# Generated code and assets are gitignored, so a fresh checkout has to run the
# generate target before it can build. CI, air, and the Docker build use it too.
VERSION ?= dev
MINIFY  ?=

.PHONY: generate generate-go generate-css build run test test-e2e lint migrate

# Default target: regenerate everything a build needs.
generate: generate-go generate-css

generate-go:
	go tool templ generate -path ./internal/components
	go tool sqlc generate

# The version is part of the filename and every file under the css directory is
# embedded in the binary, so drop stale builds before writing a new one.
generate-css:
	rm -f ./internal/dist/assets/css/output@*.css
	go tool go-tw -i ./styles/input.css \
		-o ./internal/dist/assets/css/output@$(VERSION).css \
		$(if $(filter-out 0,$(MINIFY)),--minify)

build: generate
	go build -o ./tmp/main ./cmd/server

run: build
	./tmp/main

test: generate-go
	go test -race ./...

test-e2e: generate
	go test ./e2e -tags=e2e

lint: generate-go
	golangci-lint run
	go tool sqlc vet

migrate: generate-go
	go run ./cmd/server migrate
