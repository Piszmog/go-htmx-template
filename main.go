package main

import (
	"context"
	"go-htmx-template/db"
	"go-htmx-template/logger"
	"go-htmx-template/server"
	"go-htmx-template/server/router"
	"os"
)

func main() {
	log := logger.New(
		logger.GetLevel(),
		logger.GetOutput(),
	)

	database, err := db.NewSQLite("./db.sqlite3")
	if err != nil {
		log.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	err = db.InitSchema(context.Background(), database)
	if err != nil {
		log.Error("Failed to initialize schema", "error", err)
		os.Exit(1)
	}

	queries := db.New(database)

	svr := server.New(
		log,
		":8080",
		server.WithRouter(router.New(log, queries)),
	)

	svr.StartAndWait()
}
