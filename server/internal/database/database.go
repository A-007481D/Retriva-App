// Package database provides SQLite database access for Retriva.
// It opens the database with appropriate pragmas and runs embedded migrations.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps sql.DB with logging and migration support.
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// Open opens the SQLite database at path, configures it, and runs any pending migrations.
// It returns a ready-to-use DB or an error.
func Open(path string, logger *slog.Logger) (*DB, error) {
	// DSN with pragmas for performance and correctness.
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-8000",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite is not safe for concurrent writes; serialize them.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	d := &DB{DB: db, logger: logger}

	if err := d.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return d, nil
}

// Ping checks the database connection. Used by the /ready endpoint.
func (d *DB) Ping(ctx context.Context) error {
	return d.PingContext(ctx)
}

// migrate runs all pending SQL migrations in lexicographic order.
// Migration state is tracked in the schema_migrations table.
func (d *DB) migrate(ctx context.Context) error {
	// Ensure the migrations table exists.
	_, err := d.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Read all migration files.
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := strings.TrimSuffix(entry.Name(), ".sql")

		// Check if already applied.
		var count int
		err := d.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}

		// Read and execute the migration.
		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if _, err := d.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("exec migration %s: %w", version, err)
		}

		// Record it as applied.
		if _, err := d.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
			version, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}

		d.logger.Info("migration applied", slog.String("version", version))
	}

	return nil
}
