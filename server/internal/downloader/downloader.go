package downloader

import (
	"context"

	"github.com/A-007481D/retriva/server/internal/resolver"
)

// ProgressFn is called periodically during download to report progress.
// totalBytes may be -1 if the size is unknown.
type ProgressFn func(bytesWritten int64, totalBytes int64)

// DownloadResult contains the metadata of the downloaded file.
type DownloadResult struct {
	StorageKey string
	MediaHash  string // SHA-256 hex
	SizeBytes  int64
}

// Downloader handles the process of fetching media and saving it to storage.
type Downloader interface {
	Download(ctx context.Context, info *resolver.MediaInfo, progress ProgressFn) (*DownloadResult, error)
}
