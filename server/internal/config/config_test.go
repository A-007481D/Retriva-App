package config

import (
	"os"
	"testing"
	"time"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"2GB", 2 * (1 << 30), false},
		{"500MB", 500 * (1 << 20), false},
		{"1KB", 1024, false},
		{"100B", 100, false},
		{"2147483648", 2147483648, false},
		{"0", 0, false},
		{"bad", 0, true},
		{"-1GB", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		got, err := parseBytes(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseBytes(%q): expected error, got %d", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseBytes(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseBytes(%q): got %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear all RETRIVA_ env vars to test defaults
	for _, key := range []string{
		"RETRIVA_PORT", "RETRIVA_DATA_DIR", "RETRIVA_DATABASE",
		"RETRIVA_RETENTION_DAYS", "RETRIVA_MAX_FILE_SIZE_BYTES",
		"RETRIVA_REQUEST_TIMEOUT", "RETRIVA_DOWNLOAD_TIMEOUT",
		"RETRIVA_MAX_REDIRECTS", "RETRIVA_WORKERS",
		"RETRIVA_AUTH_TOKEN", "RETRIVA_LOG_LEVEL",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with defaults failed: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port: got %d, want 8080", cfg.Port)
	}
	if cfg.Workers != 2 {
		t.Errorf("Workers: got %d, want 2", cfg.Workers)
	}
	if cfg.RetentionDays != 7 {
		t.Errorf("RetentionDays: got %d, want 7", cfg.RetentionDays)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout: got %v, want 30s", cfg.RequestTimeout)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want \"info\"", cfg.LogLevel)
	}
}

func TestLoad_Validation(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		envVal  string
		wantErr bool
	}{
		{"invalid port", "RETRIVA_PORT", "99999", true},
		{"zero workers", "RETRIVA_WORKERS", "0", true},
		{"bad log level", "RETRIVA_LOG_LEVEL", "verbose", true},
		{"bad timeout", "RETRIVA_REQUEST_TIMEOUT", "notaduration", true},
		{"valid debug level", "RETRIVA_LOG_LEVEL", "debug", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envVal)
			t.Cleanup(func() { os.Unsetenv(tt.envKey) })

			_, err := Load()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %s=%s, got nil", tt.envKey, tt.envVal)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %s=%s: %v", tt.envKey, tt.envVal, err)
			}
		})
	}
}
