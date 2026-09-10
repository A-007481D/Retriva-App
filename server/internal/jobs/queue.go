package jobs

import "context"

// Queue represents a mechanism for enqueueing jobs to be processed.
type Queue interface {
	Enqueue(ctx context.Context, job *Job) error
}
