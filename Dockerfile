## Build
FROM golang:1.24 AS build

ARG VERSION='dev'

RUN apt-get update && apt-get install -y curl

RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    && chmod +x tailwindcss-linux-x64 \
    && mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss

WORKDIR /app

COPY ./ /app

RUN go mod download \
    && go tool templ generate -path ./components \
    && tailwindcss -i ./styles/input.css -o ./dist/assets/css/output@${VERSION}.css --minify \
    && go tool sqlc generate

RUN go build -ldflags="-s -w -X version.Value=${VERSION}" -o my-app

## Deploy
FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=build /app/my-app /my-app

EXPOSE 8080

CMD ["/my-app"]
