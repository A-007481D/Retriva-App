package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLiteRepository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, job *Job) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO jobs (id, media_id, type, status, progress, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		job.ID, job.MediaID, job.Type, job.Status, job.Progress,
		job.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("jobs.Create: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (*Job, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, media_id, type, status, progress, error, created_at, started_at, finished_at
		FROM jobs WHERE id = ?`, id)

	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("jobs.Get: %w", err)
	}
	return job, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, job *Job) error {
	var startedAt, finishedAt *string
	if job.StartedAt != nil {
		s := job.StartedAt.UTC().Format(time.RFC3339)
		startedAt = &s
	}
	if job.FinishedAt != nil {
		s := job.FinishedAt.UTC().Format(time.RFC3339)
		finishedAt = &s
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs SET
			media_id = ?, status = ?, progress = ?, error = ?,
			started_at = ?, finished_at = ?
		WHERE id = ?`,
		job.MediaID, job.Status, job.Progress, job.Error,
		startedAt, finishedAt,
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("jobs.Update: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListPending(ctx context.Context) ([]*Job, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, media_id, type, status, progress, error, created_at, started_at, finished_at
		FROM jobs WHERE status IN ('queued', 'resolving', 'downloading')
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("jobs.ListPending: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJobRow(rows)
		if err != nil {
			return nil, fmt.Errorf("jobs.ListPending scan: %w", err)
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *SQLiteRepository) Cancel(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE jobs SET status = ?, finished_at = ?
		WHERE id = ? AND status NOT IN ('completed', 'failed', 'cancelled')`,
		StatusCancelled, now, id,
	)
	if err != nil {
		return fmt.Errorf("jobs.Cancel: %w", err)
	}
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(s rowScanner) (*Job, error) {
	return scanJobInternal(s)
}

func scanJobRow(rows *sql.Rows) (*Job, error) {
	return scanJobInternal(rows)
}

func scanJobInternal(s rowScanner) (*Job, error) {
	var job Job
	var mediaID sql.NullString
	var errStr sql.NullString
	var createdAt string
	var startedAt, finishedAt sql.NullString

	err := s.Scan(
		&job.ID, &mediaID, &job.Type, &job.Status, &job.Progress, &errStr,
		&createdAt, &startedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}

	if mediaID.Valid {
		job.MediaID = &mediaID.String
	}
	if errStr.Valid {
		job.Error = &errStr.String
	}
	job.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if finishedAt.Valid {
		t, _ := time.Parse(time.RFC3339, finishedAt.String)
		job.FinishedAt = &t
	}

	return &job, nil
}
