package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/A-007481D/retriva/server/internal/jobs"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/oklog/ulid/v2"
)

type createJobRequest struct {
	URL   string   `json:"url"`
	Items []string `json:"items,omitempty"`
	Item  string   `json:"item,omitempty"`
}

func (h *Handler) createAndEnqueueJob(ctx context.Context, sourceURL string, now time.Time, expiresAt time.Time) (*jobs.Job, error) {
	mediaID := ulid.Make().String()
	m := &media.Item{
		ID:         mediaID,
		OwnerID:    "local", // multi-user support could come later
		SourceURL:  sourceURL,
		SourceType: media.SourceGeneric, // the resolver might update this later if we add specific resolvers
		Filename:   "resolving...",
		MIMEType:   "application/octet-stream",
		Status:     media.StatusPending,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
	}

	if err := h.mediaRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create media: %w", err)
	}

	jobID := ulid.Make().String()
	job := &jobs.Job{
		ID:        jobID,
		MediaID:   &mediaID,
		Type:      jobs.TypeDownload,
		Status:    jobs.StatusQueued,
		Progress:  0.0,
		CreatedAt: now,
		SourceURL: sourceURL, // API response only
	}

	if err := h.jobsRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	// Use background context so enqueueing isn't cancelled if client disconnects immediately
	if err := h.pool.Enqueue(context.Background(), job); err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	return job, nil
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

	if req.Item != "" && len(req.Items) == 0 {
		req.Items = []string{req.Item}
	}

	ctx := r.Context()
	now := time.Now().UTC()

	var expiresAt time.Time
	if h.retentionDays > 0 {
		expiresAt = now.Add(time.Duration(h.retentionDays) * 24 * time.Hour)
	} else {
		// 0 means retention never expires
		expiresAt = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	if len(req.Items) > 1 {
		var createdJobs []*jobs.Job
		for _, item := range req.Items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			itemURL := req.URL
			sep := "?"
			if strings.Contains(itemURL, "?") {
				sep = "&"
			}
			itemURL = fmt.Sprintf("%s%sitem=%s", itemURL, sep, item)

			job, err := h.createAndEnqueueJob(ctx, itemURL, now, expiresAt)
			if err != nil {
				h.logger.Error("failed to create/enqueue job", "error", err, "item", item)
				continue
			}
			createdJobs = append(createdJobs, job)
		}

		if len(createdJobs) == 0 {
			writeError(w, http.StatusInternalServerError, "job_creation_failed", "Failed to create jobs")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{
			"jobs": createdJobs,
		})
		return
	}

	// If no specific items were provided, check if the URL is an inspectable collection
	// (such as an Instagram Highlight reel or multi-item playlist) and auto-expand into jobs for all items.
	if len(req.Items) == 0 && !strings.Contains(req.URL, "item=") && !strings.Contains(req.URL, "index=") {
		if h.resolver != nil {
			if info, err := h.resolver.Inspect(ctx, req.URL); err == nil && len(info.Items) > 1 {
				var createdJobs []*jobs.Job
				for _, item := range info.Items {
					itemId := strings.TrimSpace(item.ID)
					if itemId == "" {
						itemId = strconv.Itoa(item.Index)
					}
					itemURL := req.URL
					sep := "?"
					if strings.Contains(itemURL, "?") {
						sep = "&"
					}
					itemURL = fmt.Sprintf("%s%sitem=%s", itemURL, sep, itemId)

					job, err := h.createAndEnqueueJob(ctx, itemURL, now, expiresAt)
					if err != nil {
						h.logger.Error("failed to create/enqueue job", "error", err, "item", itemId)
						continue
					}
					createdJobs = append(createdJobs, job)
				}
				if len(createdJobs) > 0 {
					writeJSON(w, http.StatusAccepted, map[string]any{
						"jobs": createdJobs,
					})
					return
				}
			}
		}
	}

	singleURL := req.URL
	if len(req.Items) == 1 {
		item := strings.TrimSpace(req.Items[0])
		if !strings.Contains(singleURL, "item=") && !strings.Contains(singleURL, "story_id=") {
			sep := "?"
			if strings.Contains(singleURL, "?") {
				sep = "&"
			}
			singleURL = fmt.Sprintf("%s%sitem=%s", singleURL, sep, item)
		}
	}

	job, err := h.createAndEnqueueJob(ctx, singleURL, now, expiresAt)
	if err != nil {
		h.logger.Error("failed to create job", "error", err)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to create job")
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

func (h *Handler) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Job ID is required")
		return
	}

	job, err := h.jobsRepo.Get(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get job", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to get job")
		return
	}

	if job == nil {
		writeError(w, http.StatusNotFound, "not_found", "Job not found")
		return
	}

	writeJSON(w, http.StatusOK, job)
}
