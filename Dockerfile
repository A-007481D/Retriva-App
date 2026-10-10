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
# Stage 2: Build Go server
# =============================================================================
FROM golang:alpine AS server-builder
RUN apk add --no-cache git
WORKDIR /app/server
COPY server/go.* ./
RUN go mod download
COPY server/ ./
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /retriva \
    ./cmd/retriva

# =============================================================================
# Stage 3: Minimal runtime image
# =============================================================================
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1001 -S retriva && \
    adduser -u 1001 -S -G retriva retriva && \
    mkdir -p /data && \
    chown -R retriva:retriva /data

WORKDIR /app
COPY --from=server-builder /retriva /usr/local/bin/retriva
COPY --from=web-builder /app/client/dist /app/client/dist

USER retriva
VOLUME ["/data"]
EXPOSE 8080

ENV RETRIVA_PORT=8080 \
    RETRIVA_DATA_DIR=/data \
    RETRIVA_DATABASE=/data/retriva.db \
    RETRIVA_STATIC_DIR=/app/client/dist

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:${RETRIVA_PORT:-8080}/health || exit 1

ENTRYPOINT ["retriva"]
