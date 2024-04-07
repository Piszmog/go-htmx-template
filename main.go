package main

import (
	"go-htmx-template/db"
	"go-htmx-template/log"
	"go-htmx-template/server"
	"go-htmx-template/server/router"
	"os"
)

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	database, err := db.New(logger, "./db.sqlite3")
	if err != nil {
		logger.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if err = db.Migrate(database); err != nil {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	svr := server.New(
		logger,
		":8080",
		server.WithRouter(router.New(logger, database)),
	)

	svr.StartAndWait()
}
