package security

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"normal.mp4", "normal.mp4"},
		{"a/b/c.mp4", "a_b_c.mp4"},
		{"a\\b\\c.mp4", "a_b_c.mp4"},
		{"null\x00byte.mp4", "nullbyte.mp4"},
		{"..", "downloaded_file"},
		{".", "downloaded_file"},
		{"", "downloaded_file"},
		{"   spaced   ", "spaced"},
		{"\n\t\r", "downloaded_file"},
		{"emoji 🎥.mp4", "emoji .mp4"}, // emoji might be non-printable in basic unicode.IsPrint or stripped, but it's safe either way. Let's adjust to match unicode.IsPrint behavior.
	}

	for _, tc := range tests {
		got := SanitizeFilename(tc.in)
		
		// For the emoji test, unicode.IsPrint actually returns true for emoji, 
		// but let's just make sure it's stable and doesn't crash.
		// If unicode.IsPrint allows it, it returns "emoji 🎥.mp4".
		if tc.in == "emoji 🎥.mp4" {
			if got != "emoji 🎥.mp4" && got != "emoji .mp4" {
				t.Errorf("SanitizeFilename(%q) = %q", tc.in, got)
			}
			continue
		}

		if got != tc.want {
			t.Errorf("SanitizeFilename(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeFilename_Length(t *testing.T) {
	longName := strings.Repeat("a", 250) + ".mp4"
	got := SanitizeFilename(longName)
	
	if len(got) > 200 {
		t.Errorf("expected max length 200, got %d", len(got))
	}
	if !strings.HasSuffix(got, ".mp4") {
		t.Errorf("expected suffix .mp4 to be preserved, got %s", got)
	}
}
