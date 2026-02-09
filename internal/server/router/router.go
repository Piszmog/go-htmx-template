package router

import (
	"context"
	"log/slog"
	"net/http"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/dist"
	"go-htmx-template/internal/server/handler"
	"go-htmx-template/internal/server/middleware"
	"go-htmx-template/internal/version"
)

// New creates a new router with the given context, logger, and database.
func New(ctx context.Context, logger *slog.Logger, database db.Database) http.Handler {
	h := &handler.Handler{
		Logger:   logger,
		Database: database,
	}

	ipCfg := middleware.IPConfig{
		TrustProxyHeaders: version.Value != "dev",
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
		middleware.Logging(logger, ipCfg),
		middleware.Security(logger),
		middleware.RateLimit(ctx, logger, 50, ipCfg),
		middleware.CSRF(logger, ipCfg),
	)(handler)

	return handler
}

func newPath(method string, path string) string {
	return method + " " + path
}
