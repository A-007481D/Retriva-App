package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/A-007481D/retriva/server/internal/jobs"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/oklog/ulid/v2"
)

type createJobRequest struct {
	URL string `json:"url"`
}

func (h *Handler) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "missing_url", "URL is required")
		return
	}

	ctx := r.Context()
	now := time.Now().UTC()

	// 1. Create Media record (pending)
	mediaID := ulid.Make().String()
	m := &media.Item{
		ID:         mediaID,
		OwnerID:    "local", // multi-user support could come later
		SourceURL:  req.URL,
		SourceType: media.SourceGeneric, // the resolver might update this later if we add specific resolvers
		Filename:   "resolving...",
		MIMEType:   "application/octet-stream",
		Status:     media.StatusPending,
		CreatedAt:  now,
		ExpiresAt:  now.Add(24 * time.Hour),
	}

	if err := h.mediaRepo.Create(ctx, m); err != nil {
		h.logger.Error("failed to create media", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to create media record")
		return
	}

	// 2. Create Job record
	jobID := ulid.Make().String()
	job := &jobs.Job{
		ID:        jobID,
		MediaID:   &mediaID,
		Type:      jobs.TypeDownload,
		Status:    jobs.StatusQueued,
		Progress:  0.0,
		CreatedAt: now,
		SourceURL: req.URL, // API response only
	}

	if err := h.jobsRepo.Create(ctx, job); err != nil {
		h.logger.Error("failed to create job", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to create job record")
		return
	}

	// 3. Enqueue Job
	// Use background context so enqueueing isn't cancelled if client disconnects immediately
	if err := h.pool.Enqueue(context.Background(), job); err != nil {
		h.logger.Error("failed to enqueue job", "error", err)
		writeError(w, http.StatusInternalServerError, "queue_error", "Failed to enqueue job")
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (h *Handler) handleListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pending, err := h.jobsRepo.ListPending(ctx)
	if err != nil {
		h.logger.Error("failed to list jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to list pending jobs")
		return
	}

	if pending == nil {
		pending = make([]*jobs.Job, 0)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jobs": pending,
	})
}
