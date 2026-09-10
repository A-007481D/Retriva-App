<h1 align="center">
  <br>
  ▼ Retriva
  <br>
</h1>

<p align="center">
  <strong>Personal universal media downloader.</strong><br>
  Self-hostable. One container. Works on every device with a browser.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#development">Development</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#license">License</a>
</p>

---

## What is Retriva?

Retriva is a self-hosted media downloader that runs as a single Docker container. Open it from any device — phone, tablet, PC — paste a URL, and download the media.

```
Paste URL → Resolve → Preview → Download to device
                   ↓
             Server keeps a temporary copy for 7 days
                   ↓
             Download again if you need it
```

**Supported sources (V1):**
- Direct media URLs (`.mp4`, `.jpg`, `.webm`, `.m3u8`, etc.)
- More sources added via the adapter system

---

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/A-007481D/retriva.git
cd retriva

# 2. Configure
cp .env.example .env
# Edit .env if needed (defaults work fine for local use)

# 3. Run
docker compose up -d

# 4. Open
open http://localhost:8080
```

That's it. No database setup. No Redis. No external dependencies.

---

## Configuration

All configuration is via environment variables. Defaults work out of the box.

| Variable | Default | Description |
|---|---|---|
| `RETRIVA_PORT` | `8080` | HTTP port |
| `RETRIVA_DATA_DIR` | `/data` | Data directory (media, DB) |
| `RETRIVA_DATABASE` | `/data/retriva.db` | SQLite database path |
| `RETRIVA_RETENTION_DAYS` | `7` | Days to keep server-side copies |
| `RETRIVA_MAX_FILE_SIZE_BYTES` | `2GB` | Max download size |
| `RETRIVA_REQUEST_TIMEOUT` | `30s` | HTTP request timeout |
| `RETRIVA_DOWNLOAD_TIMEOUT` | `30m` | Download timeout |
| `RETRIVA_MAX_REDIRECTS` | `5` | Max HTTP redirects |
| `RETRIVA_WORKERS` | `2` | Concurrent download workers |
| `RETRIVA_AUTH_TOKEN` | _(empty)_ | Bearer token (empty = no auth) |
| `RETRIVA_LOG_LEVEL` | `info` | Log level: debug/info/warn/error |

### Remote access

Retriva only listens on HTTP. To access it from outside your local network:

- **Tailscale** (recommended) — install on your server and phone, access via `http://100.x.x.x:8080`
- **Cloudflare Tunnel** — `cloudflared tunnel` for a public HTTPS URL
- **Reverse proxy** — put Nginx or Caddy in front for TLS

---

## Development

### Prerequisites

- Go 1.23+
- Node.js 22+
- Docker (optional, for container testing)

### Run locally

```bash
# Terminal 1 — backend
cd server
go run ./cmd/retriva

# Terminal 2 — frontend (proxies /api to :8080)
cd web
npm install
npm run dev
```

Open `http://localhost:5173`.

### Run tests

```bash
# Backend
cd server && go test -race ./...

# Frontend
cd web && npm test
```

### Build Docker image

```bash
docker build -t retriva:local .
docker run -p 8080:8080 -v retriva-data:/data retriva:local
```

---

## Architecture

```
┌──────────────────────┐
│     Browser/PWA      │
│  React + TypeScript  │
└──────────┬───────────┘
           │ HTTP
           ▼
┌──────────────────────┐
│   Retriva Server     │
│                      │
│ Go · REST API        │
│ Job Manager          │
│ Resolver Engine      │
│ Download Engine      │
│ Vault                │
└────────┬──────┬──────┘
         │      │
    ┌────┘      └────┐
    ▼               ▼
┌────────┐    ┌──────────┐
│ SQLite │    │Filesystem│
│        │    │          │
│metadata│    │  media/  │
│  jobs  │    │ thumbs/  │
│history │    │   tmp/   │
└────────┘    └──────────┘
```

### Scaling path

| Stage | Stack |
|---|---|
| Personal (V1) | 1 container, SQLite, filesystem |
| Small server | Same, expose via Tailscale/tunnel |
| Public service | PostgreSQL + S3 + Redis (swap via interfaces) |

---

## License

MIT © 2026 [A-007481D](https://github.com/A-007481D)
