package router

import (
	"go-htmx-template/db"
	"go-htmx-template/dist"
	"go-htmx-template/server/handler"
	"go-htmx-template/server/middleware"
	"log/slog"
	"net/http"
)

func New(logger *slog.Logger, database db.Database) http.Handler {
	h := &handler.Handler{
		Logger:   logger,
		Database: database,
	}

	mux := http.NewServeMux()

	mux.Handle("/assets/", middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))
	mux.HandleFunc("/", h.Home)

	return middleware.NewLoggingMiddleware(logger, mux)
}
