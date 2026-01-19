package handler

import (
	"encoding/json"
	"go-htmx-template/internal/version"
	"net/http"
)

// Health returns a simple health check response.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(healthResponse{Version: version.Value}); err != nil {
		h.Logger.Error("failed to encode health response", "error", err)
	}
}

type healthResponse struct {
	Version string `json:"version"`
}
