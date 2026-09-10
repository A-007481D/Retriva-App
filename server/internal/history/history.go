// Package history tracks which media items a user has downloaded.
package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Record represents a single history entry.
type Record struct {
	ID        string    `json:"id"`
	MediaID   string    `json:"mediaId"`
	OwnerID   string    `json:"ownerId"`
	CreatedAt time.Time `json:"createdAt"`
}

// ListFilter for history queries.
type ListFilter struct {
	OwnerID string
	Limit   int
	Offset  int
}

// Repository is the data access interface for download history.
type Repository interface {
	Create(ctx context.Context, record *Record) error
	List(ctx context.Context, filter ListFilter) ([]*Record, error)
	Count(ctx context.Context, ownerID string) (int, error)
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLiteRepository for history.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, rec *Record) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO history (id, media_id, owner_id, created_at) VALUES (?, ?, ?, ?)",
		rec.ID, rec.MediaID, rec.OwnerID, rec.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("history.Create: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) List(ctx context.Context, f ListFilter) ([]*Record, error) {
	query := `SELECT id, media_id, owner_id, created_at FROM history WHERE owner_id = ?
	          ORDER BY created_at DESC`
	args := []any{f.OwnerID}

	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("history.List: %w", err)
	}
	defer rows.Close()

	var records []*Record
	for rows.Next() {
		var rec Record
		var createdAt string
		if err := rows.Scan(&rec.ID, &rec.MediaID, &rec.OwnerID, &createdAt); err != nil {
			return nil, fmt.Errorf("history.List scan: %w", err)
		}
		rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		records = append(records, &rec)
	}
	return records, rows.Err()
}

func (r *SQLiteRepository) Count(ctx context.Context, ownerID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM history WHERE owner_id = ?", ownerID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("history.Count: %w", err)
	}
	return count, nil
}
