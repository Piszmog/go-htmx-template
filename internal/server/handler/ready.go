package handler

import (
	"context"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

type readinessResponse struct {
	Status string `json:"status"`
}

// Ready reports whether the application can serve traffic. Unlike Health, it
// checks the database and is intended for readiness probes.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	response := readinessResponse{Status: "ok"}

	if h.database == nil {
		status = http.StatusServiceUnavailable
		response.Status = "unavailable"
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := h.database.DB().PingContext(ctx); err != nil {
			h.logger.ErrorContext(r.Context(), "readiness check failed", "error", err)
			status = http.StatusServiceUnavailable
			response.Status = "unavailable"
		}
	}

	h.writeJSON(w, r, status, response)
}
