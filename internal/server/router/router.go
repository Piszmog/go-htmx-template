package router

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/dist"
	"go-htmx-template/internal/server/handler"
	"go-htmx-template/internal/server/middleware"
)

const maxRequestBodyBytes = 1 << 20

// New creates a new router with the given context, logger, database, rate limit,
// and proxy header trust setting.
func New(ctx context.Context, logger *slog.Logger, database db.Database, rateLimit int, trustProxyHeaders bool) http.Handler {
	h := handler.New(logger, database)

	ipCfg := middleware.IPConfig{
		TrustProxyHeaders: trustProxyHeaders,
	}

	mux := http.NewServeMux()

	// Routes that should not consume the per-IP rate limit budget.
	mux.Handle("/health", allow(logger, http.MethodGet)(http.HandlerFunc(h.Health)))
	mux.Handle("/readyz", allow(logger, http.MethodGet)(http.HandlerFunc(h.Ready)))
	fileServer := middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir)))
	mux.Handle("/assets/", allow(logger, http.MethodGet)(assetHandler(logger, fileServer)))
	mux.Handle("/favicon.ico", allow(logger, http.MethodGet)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/assets/favicon.svg", http.StatusPermanentRedirect)
	})))

	// Dynamic routes share the per-IP budget. A page load would otherwise spend
	// several tokens on its assets and a single exhausted bucket would break it.
	dynamic := http.NewServeMux()
	dynamic.Handle("/{$}", allow(logger, http.MethodGet)(http.HandlerFunc(h.Home)))
	dynamic.Handle("/count", allow(logger, http.MethodPost)(http.HandlerFunc(h.Count)))
	dynamic.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middleware.WriteError(logger, w, r, http.StatusNotFound, "404 Not Found")
	}))
	mux.Handle("/", middleware.RateLimit(ctx, logger, rateLimit, middleware.DefaultMaxEntries, ipCfg)(middleware.CSP(logger)(dynamic)))

	// Middleware chain
	hdlr := http.Handler(mux)
	hdlr = middleware.Chain(
		middleware.Logging(logger, ipCfg),
		middleware.Compress(),
		middleware.Recovery(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middleware.WriteError(logger, w, r, http.StatusInternalServerError, "500 Internal Server Error")
		})),
		middleware.BodyLimit(maxRequestBodyBytes),
		middleware.SecurityHeaders(ipCfg),
		middleware.CSRF(logger, ipCfg),
	)(hdlr)

	return hdlr
}

// allow restricts a handler to the given methods and answers 405 with an Allow
// header otherwise. ServeMux only produces 405 when no other pattern matches
// the path, and the dynamic catch-all matches everything, so method rejection
// has to happen here. HEAD is implied by GET, matching ServeMux's behaviour.
func allow(logger *slog.Logger, methods ...string) middleware.Handler {
	allowed := make([]string, 0, len(methods)+1)
	for _, method := range methods {
		allowed = append(allowed, method)
		if method == http.MethodGet {
			allowed = append(allowed, http.MethodHead)
		}
	}

	allowHeader := strings.Join(allowed, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if slices.Contains(allowed, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Allow", allowHeader)
			middleware.WriteError(logger, w, r, http.StatusMethodNotAllowed, "405 Method Not Allowed")
		})
	}
}

// assetHandler serves embedded files and rejects directory requests so the
// asset tree cannot be enumerated.
func assetHandler(logger *slog.Logger, fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		info, err := fs.Stat(dist.AssetsDir, name)
		if err != nil || info.IsDir() {
			middleware.WriteError(logger, w, r, http.StatusNotFound, "404 Not Found")
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
