## Build
FROM golang:1.26 AS build

ARG VERSION='dev'

WORKDIR /app

COPY ./ /app

RUN go mod download \
    && go tool templ generate -path ./internal/components \
    && go tool go-tw -i ./styles/input.css -o ./internal/dist/assets/css/output@${VERSION}.css --minify \
    && go tool sqlc generate \
    && go build -ldflags="-s -w -X go-htmx-template/internal/version.Value=${VERSION}" -o my-app ./cmd/server

## Deploy
FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=build /app/my-app /my-app

EXPOSE 8080

CMD ["/my-app"]
