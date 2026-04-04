package handler

import (
	"context"
	"github.com/a-h/templ"
	"go-htmx-template/internal/db"
	"log/slog"
	"net/http"
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

func (h *Handler) html(ctx context.Context, w http.ResponseWriter, status int, t templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	// Use WithoutCancel so a client disconnect doesn't truncate a partially-written response.
	if err := t.Render(context.WithoutCancel(ctx), w); err != nil {
		h.logger.Error("Failed to render component", "error", err)
	}
}
