package jobs

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type mockExecutor struct {
	called int32
	wait   time.Duration
}

func (m *mockExecutor) Execute(ctx context.Context, job *Job) error {
	atomic.AddInt32(&m.called, 1)
	if m.wait > 0 {
		select {
		case <-time.After(m.wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func TestWorkerPool(t *testing.T) {
	exec := &mockExecutor{}
	pool := NewWorkerPool(2, 10, exec, slog.Default())
	pool.Start()

	for i := 0; i < 5; i++ {
		_ = pool.Enqueue(context.Background(), &Job{ID: "test"})
	}

	// Wait for workers to pick up
	time.Sleep(50 * time.Millisecond)

	pool.Stop()

	if atomic.LoadInt32(&exec.called) != 5 {
		t.Errorf("expected 5 calls, got %d", exec.called)
	}
}

func TestWorkerPool_EnqueueCancel(t *testing.T) {
	exec := &mockExecutor{wait: 500 * time.Millisecond}
	pool := NewWorkerPool(1, 1, exec, slog.Default())
	pool.Start()

	// Fill the single worker
	_ = pool.Enqueue(context.Background(), &Job{ID: "test1"})
	// Fill the queue (size 1)
	_ = pool.Enqueue(context.Background(), &Job{ID: "test2"})

	// Now enqueue should block. We'll use a context with timeout to cancel it.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := pool.Enqueue(ctx, &Job{ID: "test3"})
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}

	pool.Stop()
}
