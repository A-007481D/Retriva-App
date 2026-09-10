package jobs

import (
	"context"
	"log/slog"
	"sync"
)

// Executor defines the logic to process a job.
type Executor interface {
	Execute(ctx context.Context, job *Job) error
}

// WorkerPool manages a pool of workers processing jobs in the background.
type WorkerPool struct {
	concurrency int
	queue       chan *Job
	executor    Executor
	logger      *slog.Logger
	wg          sync.WaitGroup
	quit        chan struct{}
}

// NewWorkerPool creates a new pool.
func NewWorkerPool(concurrency int, queueSize int, executor Executor, logger *slog.Logger) *WorkerPool {
	if concurrency <= 0 {
		concurrency = 1
	}
	if queueSize <= 0 {
		queueSize = 100
	}
	return &WorkerPool{
		concurrency: concurrency,
		queue:       make(chan *Job, queueSize),
		executor:    executor,
		logger:      logger,
		quit:        make(chan struct{}),
	}
}

// Start spawns the worker goroutines.
func (p *WorkerPool) Start() {
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// Stop gracefully shuts down the pool, waiting for running jobs to finish.
func (p *WorkerPool) Stop() {
	close(p.quit)
	// We do not close p.queue immediately because there might be concurrent Enqueue calls
	// that would panic. However, since the HTTP server usually shuts down first, 
	// no new Enqueues should happen. 
	// For simplicity, we just wait for workers to drain the queue or finish current tasks.
	// Actually, closing the queue is safe if we guarantee Stop() is called after all producers stop.
	close(p.queue)
	p.wg.Wait()
}

// Enqueue pushes a job to the worker pool.
func (p *WorkerPool) Enqueue(ctx context.Context, job *Job) error {
	select {
	case <-p.quit:
		return context.Canceled
	case <-ctx.Done():
		return ctx.Err()
	case p.queue <- job:
		return nil
	}
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()
	// Process jobs until the queue is closed and empty
	for job := range p.queue {
		// Use a background context so the job can finish even if the original request context was canceled
		p.logger.Info("job started", slog.String("job_id", job.ID))
		err := p.executor.Execute(context.Background(), job)
		if err != nil {
			p.logger.Error("job failed", slog.String("job_id", job.ID), slog.Any("error", err))
		} else {
			p.logger.Info("job completed successfully", slog.String("job_id", job.ID))
		}
	}
}
