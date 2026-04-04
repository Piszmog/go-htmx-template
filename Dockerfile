## Build
FROM golang:1.26 AS build

ARG VERSION='dev'

WORKDIR /app

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

# Copy source and generate artifacts
COPY ./ .
RUN go tool templ generate -path ./internal/components \
    && go tool go-tw -i ./styles/input.css \
         -o ./internal/dist/assets/css/output@${VERSION}.css --minify \
    && go tool sqlc generate

# Build static binary
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X go-htmx-template/internal/version.Value=${VERSION}" \
    -o my-app ./cmd/server

## Deploy
FROM gcr.io/distroless/static-debian13

WORKDIR /

COPY --from=build /app/my-app /my-app

EXPOSE 8080

CMD ["/my-app"]
