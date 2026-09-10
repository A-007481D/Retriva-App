package api

import (
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

func (h *Handler) handleReady(w http.ResponseWriter, _ *http.Request) {
	// In later phases this will check DB ping and storage availability.
	writeJSON(w, http.StatusOK, readyResponse{Status: "ready"})
}
