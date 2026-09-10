// Package resolver provides interfaces and a registry for extracting media from URLs.
package resolver

import (
	"context"
)

// MediaInfo represents the resolved metadata and the final download URL.
type MediaInfo struct {
	URL       string // The final, SSRF-validated URL to download from
	MIMEType  string
	Filename  string
	SizeBytes int64 // -1 if unknown
}

// Resolver defines the interface for URL handlers.
type Resolver interface {
	// CanHandle returns true if this resolver knows how to process the given URL.
	// This is typically a fast regex or string match check without network I/O.
	CanHandle(rawURL string) bool

	// Resolve extracts the actual media info from the given URL.
	// It is responsible for making HTTP requests (if needed) to fetch metadata,
	// and MUST ensure the final MediaInfo.URL passes SSRF validation.
	Resolve(ctx context.Context, rawURL string) (*MediaInfo, error)
}
