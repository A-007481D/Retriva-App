package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/A-007481D/retriva/server/internal/storage"
)

func (h *Handler) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Media ID is required")
		return
	}

	ctx := r.Context()
	m, err := h.mediaRepo.Get(ctx, id)
	if err != nil {
		if err.Error() == "not found" { // Should ideally be a specific error type from repository
			writeError(w, http.StatusNotFound, "not_found", "Media not found")
			return
		}
		h.logger.Error("failed to get media", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "db_error", "Failed to retrieve media")
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) handleGetMediaFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Media ID is required")
		return
	}

	ctx := r.Context()
	m, err := h.mediaRepo.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Media not found")
		return
	}

	if m.StorageKey == "" {
		writeError(w, http.StatusNotFound, "not_ready", "Media file is not ready yet")
		return
	}

	reader, err := h.storage.Get(ctx, m.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Media file has been deleted or expired")
			return
		}
		h.logger.Error("failed to open media file", "error", err, "key", m.StorageKey)
		writeError(w, http.StatusInternalServerError, "storage_error", "Failed to open media file")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", m.MIMEType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+m.Filename+`"`)
	if m.SizeBytes != nil {
		// Can't use strconv if we use net/http serveContent, but we stream manually here
		// Actually for Range requests we'd use http.ServeContent, but our storage returns io.ReadCloser
		// We can just stream it down. Range requests would be nice in the future.
	}

	io.Copy(w, reader)
}

func (h *Handler) deleteMediaItem(ctx context.Context, id string) error {
	m, err := h.mediaRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	// Delete file from disk storage
	if m.StorageKey != "" {
		if err := h.storage.Delete(ctx, m.StorageKey); err != nil {
			h.logger.Warn("failed to delete media file from storage", "id", id, "key", m.StorageKey, "error", err)
		}
	}
	if m.ThumbKey != "" {
		if err := h.storage.Delete(ctx, m.ThumbKey); err != nil {
			h.logger.Warn("failed to delete thumb file from storage", "id", id, "key", m.ThumbKey, "error", err)
		}
	}

	// Remove from database
	if err := h.mediaRepo.HardDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete media from database: %w", err)
	}

	return nil
}

func (h *Handler) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Media ID is required")
		return
	}

	ctx := r.Context()
	err := h.deleteMediaItem(ctx, id)
	if err != nil {
		if err.Error() == "media not found" {
			writeError(w, http.StatusNotFound, "not_found", "Media not found")
			return
		}
		h.logger.Error("failed to delete media", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "delete_error", "Failed to delete media")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"id":      id,
	})
}

type batchDeleteMediaRequest struct {
	IDs []string `json:"ids"`
}

func (h *Handler) handleBatchDeleteMedia(w http.ResponseWriter, r *http.Request) {
	var req batchDeleteMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON request body")
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "No IDs provided")
		return
	}

	ctx := r.Context()
	deleted := make([]string, 0, len(req.IDs))
	failed := make([]string, 0)

	for _, id := range req.IDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := h.deleteMediaItem(ctx, id); err != nil {
			h.logger.Warn("failed to delete media in batch", "id", id, "error", err)
			failed = append(failed, id)
		} else {
			deleted = append(deleted, id)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"deleted": deleted,
		"failed":  failed,
		"total":   len(deleted),
	})
}
