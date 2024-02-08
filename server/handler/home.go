package handler

import (
	"go-htmx-template/components/core"
	"go-htmx-template/components/home"
	"net/http"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.html(r.Context(), w, http.StatusOK, core.HTML("Example Site", home.Home()))
}
