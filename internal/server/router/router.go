package router

import (
	"context"
	"log/slog"
	"net/http"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/dist"
	"go-htmx-template/internal/server/handler"
	"go-htmx-template/internal/server/middleware"
)

// New creates a new router with the given logger, database, and context.
// The context is used for graceful shutdown of background goroutines (e.g.,
// rate limiter cleanup).
func New(ctx context.Context, logger *slog.Logger, database db.Database) http.Handler {
	h := &handler.Handler{
		Logger:   logger,
		Database: database,
	}

	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc(newPath(http.MethodGet, "/health"), h.Health)
	mux.Handle(newPath(http.MethodGet, "/assets/"), middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))
	mux.HandleFunc(newPath(http.MethodGet, "/"), h.Home)

	// Middleware chain
	handler := http.Handler(mux)
	handler = middleware.Chain(
		middleware.Recovery(logger),
		middleware.Logging(logger),
		middleware.Security(logger),
		middleware.RateLimit(ctx, logger, 50),
		middleware.CSRF(logger),
	)(handler)

	return handler
}

func newPath(method string, path string) string {
	return method + " " + path
}
