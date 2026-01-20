package router

import (
	"go-htmx-template/internal/db"
	"go-htmx-template/internal/dist"
	"go-htmx-template/internal/server/handler"
	"go-htmx-template/internal/server/middleware"
	"log/slog"
	"net/http"
)

func New(logger *slog.Logger, database db.Database) http.Handler {
	h := &handler.Handler{
		Logger:   logger,
		Database: database,
	}

	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc(newPath(http.MethodGet, "/health"), h.Health)
	mux.Handle(newPath(http.MethodGet, "/assets/"), middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))
	mux.HandleFunc(newPath(http.MethodGet, "/"), h.Home)

	// Middleware chain (order matters: first in chain = outermost wrapper)
	handler := http.Handler(mux)
	handler = middleware.Chain(
		middleware.Recovery(logger),      // 1. Catch panics (outermost)
		middleware.Logging(logger),       // 2. Log all requests
		middleware.Security(logger),      // 3. Set security headers
		middleware.RateLimit(logger, 50), // 4. Rate limiting (50 req/min)
		middleware.CSRF(logger),          // 5. CSRF protection (native Go 1.24+)
	)(handler)

	return handler
}

func newPath(method string, path string) string {
	return method + " " + path
}
