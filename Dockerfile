## Build
# The builder runs on the native build platform and cross-compiles for the
# target. The generators in `just generate` (templ, sqlc, go-tw) are `go tool`
# directives, so they are compiled from source before they run; emulating that
# under QEMU made the linux/arm64 leg take longer than CI's 20 minute limit.
FROM --platform=$BUILDPLATFORM golang:1.27 AS build

ARG VERSION='dev'
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# `just` runs the recipes in the justfile. The image holds a single static
# binary, so copying it avoids installing a toolchain. Bump the tag by hand:
# Dependabot only tracks FROM instructions, not COPY --from.
COPY --from=ghcr.io/casey/just:1.58.0 /just /usr/local/bin/just

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

# Warm the generator toolchain into its own layer, keyed on go.mod, the justfile,
# and styles/ rather than on the whole tree. `go tool` compiles templ, sqlc, and
# go-tw from source, and go-tw downloads the Tailwind binary; both caches live in
# this layer, so without this a one-line source change recompiled sqlc and
# re-downloaded Tailwind from GitHub. The warm-up runs the real generate-css
# recipe so it cannot drift from what the build below produces; its output is
# deleted and replaced by the VERSION-specific bundle at the end.
COPY justfile ./
COPY ./styles ./styles
RUN go build -o /dev/null $(go list tool) \
    && just MINIFY=1 generate-css

# Give each generator its own COPY so a change to one does not re-run the other.
# sqlc comes first because schema changes are rarer than component changes, and
# each layer below it is discarded: sqlc reads only sqlc.yml and internal/db,
# templ only internal/components.
COPY sqlc.yml ./
COPY ./internal/db ./internal/db
RUN just generate-sqlc

COPY ./internal/components ./internal/components
RUN just generate-templ

# Everything else: handlers, entrypoint, embedded assets. Generated Go is
# excluded by .dockerignore, so the generated layers above survive this copy.
COPY ./ .

# VERSION names the CSS bundle, so a version bump re-runs only this layer.
RUN just VERSION=${VERSION} MINIFY=1 generate-css

# Build static binary for the requested platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w -X go-htmx-template/internal/version.Value=${VERSION}" \
    -o my-app ./cmd/server

## Deploy
FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /home/nonroot

COPY --from=build /app/my-app /usr/local/bin/my-app

ENV LOG_OUTPUT=json \
    DB_URL=/home/nonroot/db.sqlite3 \
    MIGRATE_ON_START=true

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/my-app"]
