package api

import (
	"errors"
	"io"
	"net/http"

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
