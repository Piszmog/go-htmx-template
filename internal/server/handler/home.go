package handler

import (
	"go-htmx-template/internal/components/core"
	"go-htmx-template/internal/components/home"
	"net/http"
	"sync/atomic"
)

var counter atomic.Int64

// Home handles the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.html(r.Context(), w, http.StatusOK, core.HTML("Example Site", home.Page(int(counter.Load()))))
}

// Count increments the counter and returns the updated Counter fragment.
func (h *Handler) Count(w http.ResponseWriter, r *http.Request) {
	newCount := counter.Add(1)
	h.html(r.Context(), w, http.StatusOK, home.Counter(int(newCount)))
}
