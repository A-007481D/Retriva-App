package security

import (
	"path/filepath"
	"strings"
	"unicode"
)

// SanitizeFilename removes path separators, null bytes, and non-printable characters.
// It also enforces a maximum length.
func SanitizeFilename(name string) string {
	// Remove null bytes
	name = strings.ReplaceAll(name, "\x00", "")

	// Remove path separators (both slash and backslash for safety)
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	// Keep only printable characters
	var sb strings.Builder
	for _, r := range name {
		if unicode.IsPrint(r) {
			sb.WriteRune(r)
		}
	}
	name = sb.String()

	name = strings.TrimSpace(name)
	
	// Basic fallback
	if name == "" || name == "." || name == ".." {
		return "downloaded_file"
	}

	// Enforce max length of 200 characters to leave room for paths in the storage system
	if len(name) > 200 {
		ext := filepath.Ext(name)
		// Try to keep the extension if it's reasonable
		if len(ext) > 10 {
			ext = ""
		}
		base := name[:200-len(ext)]
		name = base + ext
	}

	return name
}
