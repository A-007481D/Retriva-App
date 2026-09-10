package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/A-007481D/retriva/server/internal/jobs"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/oklog/ulid/v2"
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

func (h *Handler) handleRecoverMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Media ID is required")
		return
	}

	ctx := r.Context()
	m, err := h.mediaRepo.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Media not found")
		return
	}

	if m.Status != media.StatusExpired {
		writeError(w, http.StatusBadRequest, "invalid_state", "Media is not expired")
		return
	}

	// Update media status to pending
	m.Status = media.StatusPending
	// Reset expiration to give it a new 24h window once downloaded (the executor will do this later, but we can set it now or let executor handle it. For now, just change status)
	if err := h.mediaRepo.Update(ctx, m); err != nil {
		h.logger.Error("failed to update media status for recovery", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to update media")
		return
	}

	now := time.Now().UTC()

	// Create a new job
	jobID := ulid.Make().String()
	job := &jobs.Job{
		ID:        jobID,
		MediaID:   &m.ID,
		Type:      jobs.TypeDownload,
		Status:    jobs.StatusQueued,
		Progress:  0.0,
		CreatedAt: now,
		SourceURL: m.SourceURL,
	}

	if err := h.jobsRepo.Create(ctx, job); err != nil {
		h.logger.Error("failed to create recovery job", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to create recovery job")
		return
	}

	if err := h.pool.Enqueue(context.Background(), job); err != nil {
		h.logger.Error("failed to enqueue recovery job", "error", err)
		writeError(w, http.StatusInternalServerError, "queue_error", "Failed to enqueue recovery job")
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}
