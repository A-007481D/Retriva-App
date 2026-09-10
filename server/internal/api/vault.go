package api

import (
	"net/http"
	"strconv"

	"github.com/A-007481D/retriva/server/internal/media"
)

func (h *Handler) handleListVault(w http.ResponseWriter, r *http.Request) {
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

	filter := media.ListFilter{
		OwnerID: "local",
		Status:  []string{media.StatusActive},
		Limit:   limit,
		Offset:  offset,
	}

	items, err := h.mediaRepo.List(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list vault media", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to list vault media")
		return
	}

	if items == nil {
		items = make([]*media.Item, 0)
	}

	// We can reuse the filter for counting, but we need to drop Limit and Offset
	countFilter := filter
	countFilter.Limit = 0
	countFilter.Offset = 0

	total, err := h.mediaRepo.Count(ctx, countFilter)
	if err != nil {
		h.logger.Error("failed to count vault media", "error", err)
		total = 0
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":   items,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}
