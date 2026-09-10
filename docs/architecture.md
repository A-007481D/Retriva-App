# Architecture

## Overview

```
┌──────────────────────┐
│     Browser/PWA      │
│  React + TypeScript  │
└──────────┬───────────┘
           │ HTTP REST
           ▼
┌──────────────────────┐
│   Retriva Server     │
│                      │
│ Go · net/http        │
│ Config               │
│ API Handler          │
│ Job Manager          │
│ Resolver Engine      │
│ Download Engine      │
│ Vault Service        │
│ Security Layer       │
└────────┬──────┬──────┘
         │      │
    ┌────┘      └────┐
    ▼               ▼
┌────────┐    ┌──────────┐
│ SQLite │    │Filesystem│
│metadata│    │  /data/  │
└────────┘    └──────────┘
```

## Key Interfaces

These interfaces decouple business logic from infrastructure. Swapping SQLite → PostgreSQL or Filesystem → S3 requires only a new implementation of the respective interface.

### Storage

```go
type Storage interface {
    Put(ctx context.Context, key string, r io.Reader, size int64) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

### MediaRepository

```go
type Repository interface {
    Create(ctx context.Context, item *Item) error
    Get(ctx context.Context, id string) (*Item, error)
    List(ctx context.Context, filter ListFilter) ([]*Item, error)
    Update(ctx context.Context, item *Item) error
    Delete(ctx context.Context, id string) error
    ListExpired(ctx context.Context) ([]*Item, error)
}
```

### Resolver

```go
type Resolver interface {
    CanHandle(rawURL string) bool
    Resolve(ctx context.Context, rawURL string) (*MediaInfo, error)
}
```

### Downloader

```go
type Downloader interface {
    Download(ctx context.Context, info *resolver.MediaInfo, progressFn ProgressFn) (*DownloadResult, error)
}
```

## Scaling Path

| Stage | Database | Storage | Jobs |
|---|---|---|---|
| V1 Personal | SQLite | Filesystem | Go worker pool |
| V2 Small server | SQLite or PostgreSQL | Filesystem or S3 | Go worker pool |
| V3 Public | PostgreSQL | S3/R2 | Redis queue |
