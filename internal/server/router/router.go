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

// New creates a new router with the given context, logger, database, rate limit,
// and proxy header trust setting.
func New(ctx context.Context, logger *slog.Logger, database db.Database, rateLimit int, trustProxyHeaders bool) http.Handler {
	h := handler.New(logger, database)

	ipCfg := middleware.IPConfig{
		TrustProxyHeaders: trustProxyHeaders,
	}

	mux := http.NewServeMux()

	// Routes that should not consume the per-IP rate limit budget.
	mux.HandleFunc(newPath(http.MethodGet, "/health"), h.Health)
	mux.Handle(newPath(http.MethodGet, "/assets/"), middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))

	// Dynamic routes share the per-IP budget. A page load would otherwise spend
	// several tokens on its assets and a single exhausted bucket would break it.
	dynamic := http.NewServeMux()
	dynamic.HandleFunc(newPath(http.MethodGet, "/{$}"), h.Home)
	dynamic.HandleFunc(newPath(http.MethodPost, "/count"), h.Count)
	mux.Handle("/", middleware.RateLimit(ctx, logger, rateLimit, middleware.DefaultMaxEntries, ipCfg)(dynamic))

	// Middleware chain
	hdlr := http.Handler(mux)
	hdlr = middleware.Chain(
		middleware.Logging(logger, ipCfg),
		middleware.Recovery(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middleware.WriteError(logger, w, r, http.StatusInternalServerError, "500 Internal Server Error")
		})),
		middleware.Security(logger, ipCfg),
		middleware.CSRF(logger, ipCfg),
	)(hdlr)

	return hdlr
}

func newPath(method string, path string) string {
	return method + " " + path
}
