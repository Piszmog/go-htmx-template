package handler

import (
	"net/http"

	"go-htmx-template/internal/version"
)

// Health returns a simple health check response.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, r, http.StatusOK, healthResponse{Version: version.Value})
}

type healthResponse struct {
	Version string `json:"version"`
}
