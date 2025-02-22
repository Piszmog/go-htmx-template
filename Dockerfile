## Build
FROM golang:1.24 AS build

ARG VERSION='dev'

RUN apt-get update && apt-get install -y curl

WORKDIR /app

COPY ./ /app

RUN go mod download \
    && go tool templ generate -path ./components \
    && go tool go-tw -i ./styles/input.css -o ./dist/assets/css/output@${VERSION}.css --minify \
    && go tool sqlc generate

RUN go build -ldflags="-s -w -X version.Value=${VERSION}" -o my-app

## Deploy
FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=build /app/my-app /my-app

EXPOSE 8080

CMD ["/my-app"]
