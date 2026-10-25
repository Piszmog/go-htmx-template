package middleware

import (
	"log/slog"
	"net/http"

	"go-htmx-template/internal/components/core"
)

// htmxRequestHeader is always "true" on requests issued by htmx.
const htmxRequestHeader = "HX-Request"

// WriteError writes an error response. For htmx requests the body is an HTML
// fragment that htmx swaps out-of-band into the layout's #error-message, leaving
// the element that issued the request untouched. Every other client receives a
// full error page.
func WriteError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	if r.Header.Get(htmxRequestHeader) == "true" {
		if err := core.ErrorRegion(message).Render(r.Context(), w); err != nil {
			// The status line is already written, so the response cannot be corrected.
			logger.ErrorContext(r.Context(), "failed to render error fragment", "error", err)
		}
		return
	}

	if err := core.ErrorPage(status, message).Render(r.Context(), w); err != nil {
		logger.ErrorContext(r.Context(), "failed to render error page", "error", err)
	}
}
