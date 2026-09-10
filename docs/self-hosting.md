# Self-Hosting Guide

## Requirements

- Docker + Docker Compose (any modern version)
- 512 MB RAM minimum
- Disk space for your media (default retention: 7 days)

## Quick Start

```bash
git clone https://github.com/A-007481D/retriva.git
cd retriva
cp .env.example .env
docker compose up -d
```

Open `http://localhost:8080`.

## Accessing from your phone

### Same network (WiFi)

Find your server's local IP and open `http://192.168.x.x:8080` in your phone's browser.

### From anywhere (recommended: Tailscale)

1. Install Tailscale on your server and phone
2. Access via `http://100.x.x.x:8080`
3. Set `RETRIVA_AUTH_TOKEN` in `.env` for security

### Public URL (Cloudflare Tunnel)

```bash
# Install cloudflared, then:
cloudflared tunnel --url http://localhost:8080
```

## Updates

```bash
docker compose pull
docker compose up -d
```

## Data location

All data is stored in the `retriva-data` Docker volume:
- `retriva.db` — SQLite database (metadata, history, jobs)
- `media/` — Downloaded media files
- `thumbnails/` — Thumbnails
- `tmp/` — In-progress downloads

## Backup

```bash
docker compose stop
docker run --rm -v retriva-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/retriva-backup-$(date +%Y%m%d).tar.gz /data
docker compose start
```
