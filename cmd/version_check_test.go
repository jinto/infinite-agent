package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMaxVersion(t *testing.T) {
	tests := []struct {
		name    string
		dirs    []string
		want    string
		wantOK  bool
	}{
		{
			name:   "single version",
			dirs:   []string{"1.4.0"},
			want:   "1.4.0",
			wantOK: true,
		},
		{
			name:   "multiple versions picks highest",
			dirs:   []string{"1.0.0", "1.3.0", "1.1.0", "1.4.0"},
			want:   "1.4.0",
			wantOK: true,
		},
		{
			name:   "major version difference",
			dirs:   []string{"1.9.9", "2.0.0", "1.10.0"},
			want:   "2.0.0",
			wantOK: true,
		},
		{
			name:   "empty list",
			dirs:   []string{},
			want:   "",
			wantOK: false,
		},
		{
			name:   "non-semver entries ignored",
			dirs:   []string{"latest", "1.2.0", "nope"},
			want:   "1.2.0",
			wantOK: true,
		},
		{
			name:   "all non-semver",
			dirs:   []string{"latest", "nope"},
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, ok := maxVersion(tt.dirs)
			if ok != tt.wantOK {
				t.Errorf("maxVersion() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("maxVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckPluginVersion(t *testing.T) {
	// Helper: create fake cache dir with version subdirectories.
	makeCache := func(t *testing.T, versions ...string) string {
		t.Helper()
		dir := t.TempDir()
		for _, v := range versions {
			os.MkdirAll(filepath.Join(dir, v), 0755)
		}
		return dir
	}

	t.Run("binary outdated — prints warning", func(t *testing.T) {
		cache := makeCache(t, "1.0.0", "1.3.0", "1.4.0")
		var buf bytes.Buffer
		checkPluginVersionIn(cache, "v1.2.0", &buf)
		want := "ina 1.4.0 available (binary: 1.2.0). Offer to run `ina upgrade` — if the user agrees, execute it via Bash.\n"
		if buf.String() != want {
			t.Errorf("got %q, want %q", buf.String(), want)
		}
	})

	t.Run("binary up to date — no output", func(t *testing.T) {
		cache := makeCache(t, "1.0.0", "1.4.0")
		var buf bytes.Buffer
		checkPluginVersionIn(cache, "v1.4.0", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})

	t.Run("cache dir missing — silent pass", func(t *testing.T) {
		var buf bytes.Buffer
		checkPluginVersionIn("/nonexistent/path", "v1.2.0", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})

	t.Run("version parse failure — silent pass", func(t *testing.T) {
		cache := makeCache(t, "not-a-version")
		var buf bytes.Buffer
		checkPluginVersionIn(cache, "v1.2.0", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})

	t.Run("binary newer than cache — no output", func(t *testing.T) {
		cache := makeCache(t, "1.0.0", "1.2.0")
		var buf bytes.Buffer
		checkPluginVersionIn(cache, "v2.0.0", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})

	t.Run("empty cache dir — silent pass", func(t *testing.T) {
		cache := t.TempDir() // empty
		var buf bytes.Buffer
		checkPluginVersionIn(cache, "v1.2.0", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output, got %q", buf.String())
		}
	})
}
