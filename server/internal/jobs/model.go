// Package jobs defines the Job model and its repository interface.
package jobs

import (
	"context"
	"time"
)

// Job type.
const TypeDownload = "download"

// Job status values — ordered to reflect lifecycle progression.
const (
	StatusQueued      = "queued"
	StatusResolving   = "resolving"
	StatusDownloading = "downloading"
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusCancelled   = "cancelled"
)

// Job represents a single download operation tracked through its lifecycle.
type Job struct {
	ID         string     `json:"id"`
	MediaID    *string    `json:"mediaId,omitempty"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	Progress   float64    `json:"progress"`
	Error      *string    `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	// SourceURL is stored client-side only; not persisted to DB.
	SourceURL string `json:"sourceUrl,omitempty"`
}

// IsTerminal reports whether the job has reached a final state.
func (j *Job) IsTerminal() bool {
	switch j.Status {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	}
	return false
}

// Repository is the data access interface for jobs.
type Repository interface {
	Create(ctx context.Context, job *Job) error
	Get(ctx context.Context, id string) (*Job, error)
	Update(ctx context.Context, job *Job) error
	ListPending(ctx context.Context) ([]*Job, error)
	Cancel(ctx context.Context, id string) error
}
