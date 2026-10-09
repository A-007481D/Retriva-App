// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
// All values are loaded from environment variables with sensible defaults.
type Config struct {
	// Server
	Port int

	// Storage paths
	DataDir  string
	Database string

	// Retention (0 = never expire)
	RetentionDays int

	// Limits
	MaxFileSizeBytes int64
	RequestTimeout   time.Duration
	DownloadTimeout  time.Duration
	MaxRedirects     int

	// Workers
	Workers int

	// Auth
	AuthToken string

	// Logging
	LogLevel string

	// Extractors
	YtDlpPath    string
	YtDlpCookies string
}

// loadDotEnv attempts to read key-values from .env or ../.env if not already set.
func loadDotEnv() {
	for _, envFile := range []string{".env", "../.env"} {
		data, err := os.ReadFile(envFile)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
					val = val[1 : len(val)-1]
				}
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
		break
	}
}

// Load reads all configuration from environment variables, applies defaults,
// and validates the result. Returns an error listing all problems found.
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{}
	var errs []string

	cfg.Port = envInt("RETRIVA_PORT", 8080)
	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, "RETRIVA_PORT must be between 1 and 65535")
	}

	cfg.DataDir = envStr("RETRIVA_DATA_DIR", "/data")
	cfg.Database = envStr("RETRIVA_DATABASE", "/data/retriva.db")

	// RetentionDays = 0 means media will never expire.
	cfg.RetentionDays = envInt("RETRIVA_RETENTION_DAYS", 0)
	if cfg.RetentionDays < 0 {
		errs = append(errs, "RETRIVA_RETENTION_DAYS must be >= 0 (0 = never expire)")
	}

	var err error
	cfg.MaxFileSizeBytes, err = parseBytes(envStr("RETRIVA_MAX_FILE_SIZE_BYTES", "2GB"))
	if err != nil {
		errs = append(errs, fmt.Sprintf("RETRIVA_MAX_FILE_SIZE_BYTES: %v", err))
	}

	cfg.RequestTimeout, err = time.ParseDuration(envStr("RETRIVA_REQUEST_TIMEOUT", "30s"))
	if err != nil {
		errs = append(errs, fmt.Sprintf("RETRIVA_REQUEST_TIMEOUT: %v", err))
	}

	cfg.DownloadTimeout, err = time.ParseDuration(envStr("RETRIVA_DOWNLOAD_TIMEOUT", "30m"))
	if err != nil {
		errs = append(errs, fmt.Sprintf("RETRIVA_DOWNLOAD_TIMEOUT: %v", err))
	}

	cfg.MaxRedirects = envInt("RETRIVA_MAX_REDIRECTS", 5)
	if cfg.MaxRedirects < 0 || cfg.MaxRedirects > 20 {
		errs = append(errs, "RETRIVA_MAX_REDIRECTS must be between 0 and 20")
	}

	cfg.Workers = envInt("RETRIVA_WORKERS", 2)
	if cfg.Workers < 1 || cfg.Workers > 32 {
		errs = append(errs, "RETRIVA_WORKERS must be between 1 and 32")
	}

	cfg.AuthToken = envStr("RETRIVA_AUTH_TOKEN", "")

	cfg.YtDlpPath = envStr("RETRIVA_YTDLP_PATH", "yt-dlp")

	defaultCookies := ""
	for _, candidate := range []string{"data/cookies.txt", "../data/cookies.txt", "cookies.txt", "/data/cookies.txt"} {
		if _, err := os.Stat(candidate); err == nil {
			if abs, err := filepath.Abs(candidate); err == nil {
				defaultCookies = abs
			} else {
				defaultCookies = candidate
			}
			break
		}
	}
	cfg.YtDlpCookies = envStr("RETRIVA_YTDLP_COOKIES", defaultCookies)

	cfg.LogLevel = strings.ToLower(envStr("RETRIVA_LOG_LEVEL", "info"))
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, "RETRIVA_LOG_LEVEL must be one of: debug, info, warn, error")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return cfg, nil
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// parseBytes parses a human-readable byte size string.
// Supports suffixes: GB, MB, KB, B, or raw integer (bytes).
// Examples: "2GB", "500MB", "1024", "2147483648".
func parseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(upper, sf.suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(sf.suffix)])
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil || n < 0 {
				return 0, fmt.Errorf("invalid size %q", s)
			}
			return n * sf.mult, nil
		}
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return n, nil
}
