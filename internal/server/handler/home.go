package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/a-h/templ"

	"go-htmx-template/internal/components/core"
	"go-htmx-template/internal/components/home"
	"go-htmx-template/internal/server/middleware"
)

const maxCounterValue = 1_000_000_000

// Home handles the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.html(w, r, http.StatusOK, core.HTML("Example Site", home.Page(0)))
}

// Count increments the submitted counter and returns the updated Counter fragment.
// The value is round-tripped through the form so the demo does not need global
// state; it is not a security boundary or a replacement for persistence.
func (h *Handler) Count(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			middleware.WriteError(h.logger, w, r, http.StatusRequestEntityTooLarge, "Request body too large")
			return
		}
		middleware.WriteError(h.logger, w, r, http.StatusBadRequest, "Invalid form")
		return
	}

	count := 0
	if raw := r.PostForm.Get("count"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed >= maxCounterValue {
			middleware.WriteError(h.logger, w, r, http.StatusBadRequest, "Invalid count")
			return
		}
		count = parsed
	}

	h.html(w, r, http.StatusOK, templ.Join(home.Counter(count+1), core.ErrorClear()))
}
