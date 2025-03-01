package router

import (
	"go-htmx-template/db"
	"go-htmx-template/dist"
	"go-htmx-template/server/handler"
	"go-htmx-template/server/middleware"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func New(logger *slog.Logger, database db.Database) http.Handler {
	h := &handler.Handler{
		Logger:   logger,
		Database: database,
	}

	mux := http.NewServeMux()

	mux.Handle(newPath(http.MethodGet, "/assets/"), middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))
	mux.HandleFunc(newPath(http.MethodGet, "/"), h.Home)

	loggingMiddleware := middleware.LoggingMiddleware{Logger: logger}
	return middleware.Chain(
		loggingMiddleware.ServeHTTP,
		otelhttp.NewMiddleware("http-server"),
	)(mux)
}

func newPath(method string, path string) string {
	return method + " " + path
}
