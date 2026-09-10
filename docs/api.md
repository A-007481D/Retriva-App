# API Reference

Base URL: `http://localhost:8080/api/v1`

All responses are JSON. Errors follow `{"error": "message", "code": "ERROR_CODE"}`.

## Health

```
GET /health   → {"status":"ok","version":"0.1.0"}
GET /ready    → {"status":"ready"}  (503 if DB/storage unavailable)
```

## Jobs

### Create job
```
POST /api/v1/jobs
Body: {"url": "https://example.com/video.mp4"}

Response 202:
{
  "id": "01J...",
  "status": "queued",
  "sourceUrl": "https://...",
  "createdAt": "2026-09-10T..."
}
```

### Get job status
```
GET /api/v1/jobs/:id

Response 200:
{
  "id": "01J...",
  "status": "downloading",  // queued|resolving|downloading|completed|failed|cancelled
  "progress": 72.5,
  "mediaId": "01J...",      // set when resolved
  "error": null,
  "createdAt": "...",
  "startedAt": "...",
  "finishedAt": null
}
```

### Cancel job
```
DELETE /api/v1/jobs/:id → 204
```

## Media

### Get media metadata
```
GET /api/v1/media/:id → MediaItem
```

### Download media file
```
GET /api/v1/media/:id/file
Response: file stream (Content-Disposition: attachment)
Supports: Range requests
```

### Delete media
```
DELETE /api/v1/media/:id → 204
```

## History

```
GET /api/v1/history?limit=20&offset=0

Response 200:
{
  "items": [...],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

## Vault

```
GET /api/v1/vault?status=active&limit=20&offset=0

POST /api/v1/vault/:id/recover
Response 202: { "jobId": "01J..." }
```

## Authentication

If `RETRIVA_AUTH_TOKEN` is set, all `/api/v1/*` routes require:
```
Authorization: Bearer <token>
```

Returns `401` with `WWW-Authenticate: Bearer` if missing/invalid.
