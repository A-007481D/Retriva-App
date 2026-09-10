package api

import (
	"net/http"
	"strconv"

	"github.com/A-007481D/retriva/server/internal/history"
)

func (h *Handler) handleListHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	filter := history.ListFilter{
		OwnerID: "local",
		Limit:   limit,
		Offset:  offset,
	}

	records, err := h.historyRepo.List(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list history", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to list history")
		return
	}

	if records == nil {
		records = make([]*history.Record, 0)
	}

	total, err := h.historyRepo.Count(ctx, "local")
	if err != nil {
		h.logger.Error("failed to count history", "error", err)
		total = 0
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"history": records,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}
