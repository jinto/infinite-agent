package hud

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderContextBar(t *testing.T) {
	tests := []struct {
		name    string
		pct     int
		wantSev severity
		wantHas string
	}{
		{"low", 30, sevNormal, "30%"},
		{"warning", 72, sevWarning, "72%"},
		{"compress", 82, sevCompress, "COMPRESS?"},
		{"critical", 90, sevCritical, "CRITICAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderContextBar(tt.pct, tt.wantSev, 10)
			if !strings.Contains(got, tt.wantHas) {
				t.Errorf("renderContextBar(%d) = %q, want containing %q", tt.pct, got, tt.wantHas)
			}
		})
	}
}

func TestRenderWarning(t *testing.T) {
	if w := renderWarning(50); w != "" {
		t.Errorf("expected no warning at 50%%, got %q", w)
	}
	if w := renderWarning(80); !strings.Contains(w, "/compact") {
		t.Errorf("expected /compact in warning at 80%%, got %q", w)
	}
	if w := renderWarning(92); !strings.Contains(w, "!!") {
		t.Errorf("expected !! in critical warning at 92%%, got %q", w)
	}
}

func TestRenderFromJSON(t *testing.T) {
	input := `{"context_window":{"context_window_size":1000000,"used_percentage":67.3}}`
	var buf bytes.Buffer
	err := Render(strings.NewReader(input), &buf)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ctx:") {
		t.Errorf("expected ctx: in output, got %q", out)
	}
	if !strings.Contains(out, "67%") {
		t.Errorf("expected 67%% in output, got %q", out)
	}
}

func TestRenderCriticalWithWarning(t *testing.T) {
	input := `{"context_window":{"context_window_size":1000000,"used_percentage":91.5}}`
	var buf bytes.Buffer
	err := Render(strings.NewReader(input), &buf)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "CRITICAL") {
		t.Errorf("expected CRITICAL at 92%%, got %q", out)
	}
	if !strings.Contains(out, "/compact") {
		t.Errorf("expected /compact warning at 92%%, got %q", out)
	}
}

func TestRenderEmptyStdin(t *testing.T) {
	var buf bytes.Buffer
	err := Render(strings.NewReader(""), &buf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[ina]") {
		t.Errorf("expected fallback, got %q", buf.String())
	}
}

func TestRenderNoContextWindow(t *testing.T) {
	var buf bytes.Buffer
	err := Render(strings.NewReader(`{"cwd":"/tmp"}`), &buf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[ina]") {
		t.Errorf("expected fallback, got %q", buf.String())
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		pct  int
		want severity
	}{
		{0, sevNormal},
		{69, sevNormal},
		{70, sevWarning},
		{79, sevWarning},
		{80, sevCompress},
		{84, sevCompress},
		{85, sevCritical},
		{100, sevCritical},
	}
	for _, tt := range tests {
		got := classify(tt.pct)
		if got != tt.want {
			t.Errorf("classify(%d) = %d, want %d", tt.pct, got, tt.want)
		}
	}
}
