# syntax=docker/dockerfile:1
# =============================================================================
# Stage 1: Build web frontend
# =============================================================================
FROM node:22-alpine AS web-builder
WORKDIR /app/client
COPY client/package*.json ./
RUN npm ci --silent
COPY client/ ./
RUN npm run build

# =============================================================================
# Stage 2: Build Go server (embeds the frontend)
# =============================================================================
FROM golang:1.23-alpine AS server-builder
RUN apk add --no-cache git
WORKDIR /app/server
COPY server/go.* ./
RUN go mod download
COPY server/ ./
# Embed the built frontend into the binary via go:embed
COPY --from=web-builder /app/client/dist ./static
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /retriva \
    ./cmd/retriva

# =============================================================================
# Stage 3: Minimal runtime image
# =============================================================================
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata python3 ffmpeg curl && \
    curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp && \
    chmod a+rx /usr/local/bin/yt-dlp && \
    addgroup -g 1001 -S retriva && \
    adduser -u 1001 -S -G retriva retriva

COPY --from=server-builder /retriva /usr/local/bin/retriva

USER retriva
VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["retriva"]
