## Build
# The builder runs on the native build platform and cross-compiles for the
# target. The generators in `make generate` (templ, sqlc, go-tw) are `go tool`
# directives, so they are compiled from source before they run; emulating that
# under QEMU made the linux/arm64 leg take longer than CI's 20 minute limit.
FROM --platform=$BUILDPLATFORM golang:1.27 AS build

ARG VERSION='dev'
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

# Copy source and generate artifacts. Generated Go is architecture independent
# and does not read VERSION, so keeping it in its own layer means a version bump
# alone does not re-run templ and sqlc.
COPY ./ .
RUN make generate-go
RUN make generate-css VERSION=${VERSION} MINIFY=1

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
