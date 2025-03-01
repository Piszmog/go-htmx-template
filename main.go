package main

import (
	"context"
	"errors"
	"go-htmx-template/db"
	"go-htmx-template/log"
	"go-htmx-template/otel"
	"go-htmx-template/server"
	"go-htmx-template/server/router"
	"os"

	"github.com/golang-migrate/migrate/v4"
)

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	shutdown, err := otel.Init(context.Background(), otel.Config{Insecure: true, Endpoint: otelEndpoint, ServiceName: "my-app"})
	if err != nil {
		logger.Error("Failed to initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}
	defer shutdown(context.Background())

	database, err := db.New(logger, "./db.sqlite3")
	if err != nil {
		logger.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err = db.Migrate(database); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	svr := server.New(
		logger,
		":"+port,
		server.WithRouter(router.New(logger, database)),
	)

	svr.StartAndWait()
}
