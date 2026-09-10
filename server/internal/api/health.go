package api

import (
	"log/slog"
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type readyResponse struct {
	Status string `json:"status"`
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: Version,
	})
}

func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	if err := h.db.Ping(ctx); err != nil {
		h.logger.Error("ready check failed: database", slog.Any("error", err))
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "Database is not ready")
		return
	}

	if err := h.storage.Ping(ctx); err != nil {
		h.logger.Error("ready check failed: storage", slog.Any("error", err))
		writeError(w, http.StatusServiceUnavailable, "storage_unavailable", "Storage is not ready")
		return
	}

	writeJSON(w, http.StatusOK, readyResponse{Status: "ready"})
}
