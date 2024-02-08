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
	Logger  *slog.Logger
	Queries *db.Queries
}

func (h *Handler) html(ctx context.Context, w http.ResponseWriter, status int, t templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	_ = t.Render(context.Background(), w)
}
