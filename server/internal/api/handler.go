// Package api implements the HTTP handler and all REST endpoints for Retriva.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/A-007481D/retriva/server/internal/database"
	"github.com/A-007481D/retriva/server/internal/history"
	"github.com/A-007481D/retriva/server/internal/jobs"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/security"
	"github.com/A-007481D/retriva/server/internal/storage"
	"github.com/oklog/ulid/v2"
)

// Version is set at build time via -ldflags "-X main.version=...".
var Version = "dev"

type contextKey string

const requestIDKey contextKey = "requestID"

// Handler is the root HTTP handler. It owns the ServeMux and middleware chain.
type Handler struct {
	mux           *http.ServeMux
	logger        *slog.Logger
	db            *database.DB
	storage       storage.Storage
	jobsRepo      jobs.Repository
	mediaRepo     media.Repository
	historyRepo   history.Repository
	resolver      *resolver.Registry
	pool          *jobs.WorkerPool
	authToken     string
	retentionDays int
}

// New creates a fully configured Handler with all routes registered.
func New(
	logger *slog.Logger,
	db *database.DB,
	store storage.Storage,
	jobsRepo jobs.Repository,
	mediaRepo media.Repository,
	historyRepo history.Repository,
	resRegistry *resolver.Registry,
	pool *jobs.WorkerPool,
	authToken string,
	retentionDays int,
) *Handler {
	h := &Handler{
		mux:           http.NewServeMux(),
		logger:        logger,
		db:            db,
		storage:       store,
		jobsRepo:      jobsRepo,
		mediaRepo:     mediaRepo,
		historyRepo:   historyRepo,
		resolver:      resRegistry,
		pool:          pool,
		authToken:     authToken,
		retentionDays: retentionDays,
	}
	h.registerRoutes()
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	chain := h.recovery(h.requestID(h.logging(h.cors(h.mux))))
	chain.ServeHTTP(w, r)
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("GET /health", h.handleHealth)
	h.mux.HandleFunc("GET /ready", h.handleReady)

	// API v1 (Auth Protected)
	h.mux.Handle("POST /api/v1/jobs", h.requireAuth(http.HandlerFunc(h.handleCreateJob)))
	h.mux.Handle("GET /api/v1/jobs", h.requireAuth(http.HandlerFunc(h.handleListJobs)))
	h.mux.Handle("GET /api/v1/jobs/{id}", h.requireAuth(http.HandlerFunc(h.handleGetJob)))
	h.mux.Handle("POST /api/v1/inspect", h.requireAuth(http.HandlerFunc(h.handleInspect)))
	h.mux.Handle("GET /api/v1/inspect", h.requireAuth(http.HandlerFunc(h.handleInspect)))
	h.mux.Handle("GET /api/v1/history", h.requireAuth(http.HandlerFunc(h.handleListHistory)))
	h.mux.Handle("GET /api/v1/vault", h.requireAuth(http.HandlerFunc(h.handleListVault)))
	h.mux.Handle("POST /api/v1/vault/{id}/recover", h.requireAuth(http.HandlerFunc(h.handleRecoverMedia)))
	h.mux.Handle("GET /api/v1/media/{id}", h.requireAuth(http.HandlerFunc(h.handleGetMedia)))
	h.mux.Handle("DELETE /api/v1/media/{id}", h.requireAuth(http.HandlerFunc(h.handleDeleteMedia)))
	h.mux.Handle("POST /api/v1/media/batch-delete", h.requireAuth(http.HandlerFunc(h.handleBatchDeleteMedia)))
	
	// API v1 (Public - ULID acts as capability token)
	h.mux.HandleFunc("GET /api/v1/media/{id}/file", h.handleGetMediaFile)
	h.mux.HandleFunc("GET /api/v1/proxy", h.handleImageProxy)

	// Serve static frontend files (SPA fallback to index.html)
	candidates := []string{
		os.Getenv("RETRIVA_STATIC_DIR"),
		"client/dist",
		"../client/dist",
		"../../client/dist",
		"/app/client/dist",
	}
	distDir := ""
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			distDir = c
			break
		}
	}

	if distDir != "" {
		fs := http.FileServer(http.Dir(distDir))
		h.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If it's an API route that fell through, 404
			if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
				http.NotFound(w, r)
				return
			}

			// Try to serve the requested file if it exists
			filePath := filepath.Join(distDir, filepath.Clean(r.URL.Path))
			if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
				fs.ServeHTTP(w, r)
				return
			}

			// Otherwise, fall back to index.html
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
		}))
	}
}

// ─── Middleware ──────────────────────────────────────────────────────────────

func (h *Handler) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ulid.Make().String()
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		reqID, _ := r.Context().Value(requestIDKey).(string)
		h.logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.status),
			slog.Duration("latency", time.Since(start)),
			slog.String("remote", r.RemoteAddr),
			slog.String("request_id", reqID),
		)
	})
}

func (h *Handler) recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Error("panic recovered",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				writeError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.authToken == "" {
			// No auth configured, allow all
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Missing Authorization header")
			return
		}

		// Check "Bearer <token>"
		const prefix = "Bearer "
		if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid Authorization header format")
			return
		}

		token := authHeader[len(prefix):]
		if token != h.authToken {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) handleInspect(w http.ResponseWriter, r *http.Request) {
	urlStr := ""
	if r.Method == http.MethodGet {
		urlStr = r.URL.Query().Get("url")
	} else {
		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			urlStr = req.URL
		}
	}

	if urlStr == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "URL is required")
		return
	}

	if h.resolver == nil {
		writeError(w, http.StatusInternalServerError, "not_available", "Resolver is not configured")
		return
	}

	info, err := h.resolver.Inspect(r.Context(), urlStr)
	if err != nil {
		h.logger.Error("failed to inspect URL", "url", urlStr, "error", err)
		writeError(w, http.StatusBadRequest, "inspect_failed", fmt.Sprintf("Failed to inspect URL: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// ─── Response helpers ────────────────────────────────────────────────────────

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: message, Code: code})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (rw *responseWriter) WriteHeader(status int) {
	if !rw.written {
		rw.status = status
		rw.written = true
		rw.ResponseWriter.WriteHeader(status)
	}
}

func (h *Handler) handleImageProxy(w http.ResponseWriter, r *http.Request) {
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	parsed, err := security.ValidateURL(r.Context(), rawURL)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.instagram.com/")

	client := security.SafeHTTPClient(15*time.Second, 5)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to fetch image", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("upstream returned %d", resp.StatusCode), resp.StatusCode)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	}

	_, _ = io.Copy(w, resp.Body)
}
