-- Migration: 0001_initial
-- Created: 2026-09-10
-- Description: Initial schema — media, jobs, history tables with indices.

CREATE TABLE IF NOT EXISTS media (
    id           TEXT PRIMARY KEY,
    owner_id     TEXT NOT NULL DEFAULT 'local',
    source_url   TEXT NOT NULL,
    source_type  TEXT NOT NULL,
    filename     TEXT NOT NULL,
    mime_type    TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes   INTEGER,
    duration_s   INTEGER,
    width        INTEGER,
    height       INTEGER,
    storage_key  TEXT,
    thumb_key    TEXT,
    status       TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'active', 'expired', 'deleted')),
    media_hash   TEXT,
    created_at   TEXT NOT NULL,
    expires_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
    id           TEXT PRIMARY KEY,
    media_id     TEXT REFERENCES media(id) ON DELETE SET NULL,
    type         TEXT NOT NULL DEFAULT 'download',
    status       TEXT NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued', 'resolving', 'downloading', 'completed', 'failed', 'cancelled')),
    progress     REAL NOT NULL DEFAULT 0
                    CHECK (progress >= 0 AND progress <= 100),
    error        TEXT,
    created_at   TEXT NOT NULL,
    started_at   TEXT,
    finished_at  TEXT
);

CREATE TABLE IF NOT EXISTS history (
    id           TEXT PRIMARY KEY,
    media_id     TEXT NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    owner_id     TEXT NOT NULL DEFAULT 'local',
    created_at   TEXT NOT NULL
);

-- Indices for common query patterns
CREATE INDEX IF NOT EXISTS idx_media_status     ON media(status);
CREATE INDEX IF NOT EXISTS idx_media_expires_at ON media(expires_at);
CREATE INDEX IF NOT EXISTS idx_media_owner      ON media(owner_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status      ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_media_id    ON jobs(media_id);
CREATE INDEX IF NOT EXISTS idx_history_owner    ON history(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_media    ON history(media_id);
