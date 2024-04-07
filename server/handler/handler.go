package handler

import (
	"context"
	"github.com/a-h/templ"
	"go-htmx-template/db"
	"log/slog"
	"net/http"
)

// Handler handles requests.
type Handler struct {
	Logger   *slog.Logger
	Database db.Database
}

func (h *Handler) html(ctx context.Context, w http.ResponseWriter, status int, t templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := t.Render(ctx, w); err != nil {
		h.Logger.Error("Failed to render component", "error", err)
	}
}
