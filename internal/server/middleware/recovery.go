package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns a middleware that recovers from panics and delegates to onPanic.
func Recovery(logger *slog.Logger, onPanic http.Handler) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error(
						"panic recovered",
						slog.Any("panic", err),
						slog.String("stack", string(debug.Stack())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)

					onPanic.ServeHTTP(w, r)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
