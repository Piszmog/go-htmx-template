package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"go-htmx-template/internal/db"
	"go-htmx-template/internal/server/middleware"
)

// Handler handles requests.
type Handler struct {
	logger   *slog.Logger
	database db.Database
}

// New creates a new Handler.
func New(logger *slog.Logger, database db.Database) *Handler {
	return &Handler{logger: logger, database: database}
}

// html renders t into a buffer before writing the status and headers, so a
// render failure still produces a proper error response instead of a truncated
// 200. The buffer also gives the response an accurate Content-Length.
func (h *Handler) html(w http.ResponseWriter, r *http.Request, status int, t templ.Component) {
	var buf bytes.Buffer
	// Use WithoutCancel so a client disconnect doesn't truncate the render.
	if err := t.Render(context.WithoutCancel(r.Context()), &buf); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to render component", "error", err)
		middleware.WriteError(h.logger, w, r, http.StatusInternalServerError, "500 Internal Server Error")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to write response", "error", err)
	}
}

// writeJSON buffers the payload so the response carries an accurate
// Content-Length. That keeps callers from re-encoding the body to find its
// size, and lets intermediaries and the compression middleware see how large
// the response really is before it is written.
func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, status int, payload any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to encode JSON response", "error", err)
		// Writing nothing here would leave net/http to send an empty 200.
		middleware.WriteError(h.logger, w, r, http.StatusInternalServerError, "500 Internal Server Error")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to write JSON response", "error", err)
	}
}
