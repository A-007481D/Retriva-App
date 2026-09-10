package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/A-007481D/retriva/server/internal/downloader"
	"github.com/A-007481D/retriva/server/internal/history"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/oklog/ulid/v2"
)

// DefaultExecutor coordinates the actual download process.
type DefaultExecutor struct {
	resolver    *resolver.Registry
	downloader  downloader.Downloader
	jobsRepo    Repository
	mediaRepo   media.Repository
	historyRepo history.Repository
	logger      *slog.Logger
}

// NewExecutor creates a new job executor.
func NewExecutor(
	res *resolver.Registry,
	dl downloader.Downloader,
	jr Repository,
	mr media.Repository,
	hr history.Repository,
	logger *slog.Logger,
) *DefaultExecutor {
	return &DefaultExecutor{
		resolver:    res,
		downloader:  dl,
		jobsRepo:    jr,
		mediaRepo:   mr,
		historyRepo: hr,
		logger:      logger,
	}
}

// Execute performs the resolution and download for a job.
func (e *DefaultExecutor) Execute(ctx context.Context, job *Job) error {
	now := time.Now().UTC()
	job.StartedAt = &now

	failJob := func(err error) error {
		errMsg := err.Error()
		job.Status = StatusFailed
		job.Error = &errMsg
		finished := time.Now().UTC()
		job.FinishedAt = &finished
		if updateErr := e.jobsRepo.Update(ctx, job); updateErr != nil {
			e.logger.Error("failed to mark job as failed", slog.String("job_id", job.ID), slog.Any("error", updateErr))
		}
		
		// On job failure, we leave the media in 'pending' status.
		return err
	}

	if job.MediaID == nil {
		return failJob(fmt.Errorf("job has no media_id"))
	}

	m, err := e.mediaRepo.Get(ctx, *job.MediaID)
	if err != nil {
		return failJob(fmt.Errorf("get media failed: %w", err))
	}

	// 1. Mark resolving
	job.Status = StatusResolving
	if err := e.jobsRepo.Update(ctx, job); err != nil {
		return failJob(fmt.Errorf("update job status to resolving: %w", err))
	}

	// 2. Resolve URL
	info, err := e.resolver.Resolve(ctx, m.SourceURL)
	if err != nil {
		return failJob(fmt.Errorf("resolve failed: %w", err))
	}
	
	// Update media with early info (e.g. filename) if we want, but we can wait until download finishes.

	// 3. Mark downloading
	job.Status = StatusDownloading
	if err := e.jobsRepo.Update(ctx, job); err != nil {
		return failJob(fmt.Errorf("update job status to downloading: %w", err))
	}

	// 4. Download
	res, err := e.downloader.Download(ctx, info, func(written, total int64) {
		var pct float64
		if total > 0 {
			pct = float64(written) / float64(total) * 100.0
			if pct > 100 {
				pct = 100
			}
		}
		
		job.Progress = pct
		// Update DB with progress. In a real system, you'd throttle this. 
		// The downloader already throttles the progress callback to 500ms.
		e.jobsRepo.Update(ctx, job)
	})
	if err != nil {
		return failJob(fmt.Errorf("download failed: %w", err))
	}

	// 5. Update Media
	m.StorageKey = res.StorageKey
	m.MIMEType = info.MIMEType
	m.Filename = info.Filename
	m.SizeBytes = &res.SizeBytes
	m.MediaHash = res.MediaHash
	m.Status = media.StatusActive
	if err := e.mediaRepo.Update(ctx, m); err != nil {
		return failJob(fmt.Errorf("update media failed: %w", err))
	}

	// 6. Create History Entry
	historyEntry := &history.Record{
		ID:        ulid.Make().String(),
		MediaID:   m.ID,
		OwnerID:   m.OwnerID,
		CreatedAt: time.Now().UTC(),
	}
	if err := e.historyRepo.Create(ctx, historyEntry); err != nil {
		e.logger.Error("failed to create history entry", slog.String("job_id", job.ID), slog.Any("error", err))
		// We don't fail the job if history fails
	}

	// 7. Mark Job Completed
	job.Status = StatusCompleted
	job.Progress = 100.0
	finished := time.Now().UTC()
	job.FinishedAt = &finished
	
	if err := e.jobsRepo.Update(ctx, job); err != nil {
		e.logger.Error("failed to mark job completed", slog.String("job_id", job.ID), slog.Any("error", err))
	}

	return nil
}
