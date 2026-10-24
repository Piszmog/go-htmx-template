package middleware

import (
	"log/slog"
	"net/http"

	"go-htmx-template/internal/components/core"
)

// htmxRequestHeader is always "true" on requests issued by htmx.
const htmxRequestHeader = "HX-Request"

// WriteError writes an error response. For htmx requests the body is an HTML
// fragment that htmx swaps out-of-band into the layout's #error-region, leaving
// the element that issued the request untouched. Every other client receives a
// plain-text response.
func WriteError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, status int, message string) {
	if r.Header.Get(htmxRequestHeader) != "true" {
		http.Error(w, message, status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := core.ErrorRegion(message).Render(r.Context(), w); err != nil {
		// The status line is already written, so the response cannot be corrected.
		logger.ErrorContext(r.Context(), "failed to render error fragment", "error", err)
	}
}
