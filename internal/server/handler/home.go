package handler

import (
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"go-htmx-template/internal/components/core"
	"go-htmx-template/internal/components/home"
	"go-htmx-template/internal/server/middleware"
)

// Home handles the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.html(r.Context(), w, http.StatusOK, core.HTML("Example Site", home.Page(0)))
}

// Count increments the submitted counter and returns the updated Counter fragment.
// The value is round-tripped through the form so the demo does not need global
// state; it is not a security boundary or a replacement for persistence.
func (h *Handler) Count(w http.ResponseWriter, r *http.Request) {
	count := 0
	if raw := r.PostFormValue("count"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			middleware.WriteError(h.logger, w, r, http.StatusBadRequest, "Invalid count")
			return
		}
		count = parsed
	}

	h.html(r.Context(), w, http.StatusOK, templ.Join(home.Counter(count+1), core.ErrorClear()))
}
