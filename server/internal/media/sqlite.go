package media

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SQLiteRepository implements Repository using a *sql.DB backed by SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLiteRepository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, item *Item) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO media (
			id, owner_id, source_url, source_type, filename, mime_type,
			size_bytes, duration_s, width, height, storage_key, thumb_key,
			status, media_hash, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.OwnerID, item.SourceURL, item.SourceType, item.Filename, item.MIMEType,
		item.SizeBytes, item.DurationS, item.Width, item.Height, item.StorageKey, item.ThumbKey,
		item.Status, item.MediaHash,
		item.CreatedAt.UTC().Format(time.RFC3339),
		item.ExpiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("media.Create: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (*Item, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, owner_id, source_url, source_type, filename, mime_type,
		       size_bytes, duration_s, width, height, storage_key, thumb_key,
		       status, media_hash, created_at, expires_at
		FROM media WHERE id = ?`, id)

	item, err := scanItem(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("media.Get: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) List(ctx context.Context, f ListFilter) ([]*Item, error) {
	query, args := buildListQuery(f, false)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("media.List: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		item, err := scanItemRow(rows)
		if err != nil {
			return nil, fmt.Errorf("media.List scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) Count(ctx context.Context, f ListFilter) (int, error) {
	query, args := buildListQuery(f, true)
	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("media.Count: %w", err)
	}
	return count, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, item *Item) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE media SET
			filename = ?, mime_type = ?, size_bytes = ?, duration_s = ?,
			width = ?, height = ?, storage_key = ?, thumb_key = ?,
			status = ?, media_hash = ?, expires_at = ?
		WHERE id = ?`,
		item.Filename, item.MIMEType, item.SizeBytes, item.DurationS,
		item.Width, item.Height, item.StorageKey, item.ThumbKey,
		item.Status, item.MediaHash, item.ExpiresAt.UTC().Format(time.RFC3339),
		item.ID,
	)
	if err != nil {
		return fmt.Errorf("media.Update: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE media SET status = ? WHERE id = ?", StatusDeleted, id)
	if err != nil {
		return fmt.Errorf("media.Delete: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListExpired(ctx context.Context) ([]*Item, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, owner_id, source_url, source_type, filename, mime_type,
		       size_bytes, duration_s, width, height, storage_key, thumb_key,
		       status, media_hash, created_at, expires_at
		FROM media
		WHERE expires_at < ? AND status = ?`, now, StatusActive)
	if err != nil {
		return nil, fmt.Errorf("media.ListExpired: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		item, err := scanItemRow(rows)
		if err != nil {
			return nil, fmt.Errorf("media.ListExpired scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildListQuery(f ListFilter, count bool) (string, []any) {
	var sb strings.Builder
	var args []any

	if count {
		sb.WriteString("SELECT COUNT(*) FROM media WHERE 1=1")
	} else {
		sb.WriteString(`SELECT id, owner_id, source_url, source_type, filename, mime_type,
		       size_bytes, duration_s, width, height, storage_key, thumb_key,
		       status, media_hash, created_at, expires_at
		FROM media WHERE 1=1`)
	}

	if f.OwnerID != "" {
		sb.WriteString(" AND owner_id = ?")
		args = append(args, f.OwnerID)
	}

	if len(f.Status) > 0 {
		placeholders := strings.Repeat("?,", len(f.Status))
		placeholders = placeholders[:len(placeholders)-1]
		sb.WriteString(fmt.Sprintf(" AND status IN (%s)", placeholders))
		for _, s := range f.Status {
			args = append(args, s)
		}
	}

	if !count {
		sb.WriteString(" ORDER BY created_at DESC")
		if f.Limit > 0 {
			sb.WriteString(" LIMIT ?")
			args = append(args, f.Limit)
		}
		if f.Offset > 0 {
			sb.WriteString(" OFFSET ?")
			args = append(args, f.Offset)
		}
	}

	return sb.String(), args
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(s scanner) (*Item, error) {
	return scanItemInternal(s)
}

func scanItemRow(rows *sql.Rows) (*Item, error) {
	return scanItemInternal(rows)
}

func scanItemInternal(s scanner) (*Item, error) {
	var item Item
	var createdAt, expiresAt string
	var sizeBytes sql.NullInt64
	var durationS, width, height sql.NullInt64
	var storageKey, thumbKey, mediaHash sql.NullString

	err := s.Scan(
		&item.ID, &item.OwnerID, &item.SourceURL, &item.SourceType,
		&item.Filename, &item.MIMEType,
		&sizeBytes, &durationS, &width, &height,
		&storageKey, &thumbKey,
		&item.Status, &mediaHash,
		&createdAt, &expiresAt,
	)
	if err != nil {
		return nil, err
	}

	if sizeBytes.Valid {
		v := sizeBytes.Int64
		item.SizeBytes = &v
	}
	if durationS.Valid {
		v := int(durationS.Int64)
		item.DurationS = &v
	}
	if width.Valid {
		v := int(width.Int64)
		item.Width = &v
	}
	if height.Valid {
		v := int(height.Int64)
		item.Height = &v
	}
	if storageKey.Valid {
		item.StorageKey = storageKey.String
	}
	if thumbKey.Valid {
		item.ThumbKey = thumbKey.String
	}
	if mediaHash.Valid {
		item.MediaHash = mediaHash.String
	}

	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	return &item, nil
}
