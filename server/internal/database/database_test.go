package database

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	db, err := Open(dbPath, logger)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestMigrations_AppliedOnce(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Open twice — migrations should only run once.
	db1, err := Open(dbPath, logger)
	if err != nil {
		t.Fatalf("first Open() error: %v", err)
	}
	db1.Close()

	db2, err := Open(dbPath, logger)
	if err != nil {
		t.Fatalf("second Open() error: %v", err)
	}
	defer db2.Close()

	// Verify tables exist.
	tables := []string{"media", "jobs", "history", "schema_migrations"}
	for _, table := range tables {
		var name string
		err := db2.QueryRowContext(context.Background(),
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after migrations: %v", table, err)
		}
	}
}

func TestMigrations_VersionTracked(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	db, err := Open(dbPath, logger)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM schema_migrations",
	).Scan(&count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one migration to be recorded")
	}
}
