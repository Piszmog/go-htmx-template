package main

import (
	"context"
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

	database, err := db.NewSQLite("./db.sqlite3")
	if err != nil {
		logger.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	err = db.InitSchema(context.Background(), database)
	if err != nil {
		logger.Error("Failed to initialize schema", "error", err)
		os.Exit(1)
	}

	queries := db.New(database)

	svr := server.New(
		logger,
		":8080",
		server.WithRouter(router.New(logger, queries)),
	)

	svr.StartAndWait()
}
