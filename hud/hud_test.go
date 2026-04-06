package hud

import (
	"bytes"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	input := `{"cwd":"/home/user/project","context_window":{"context_window_size":200000,"used_percentage":42}}`
	var buf bytes.Buffer
	if err := Render(strings.NewReader(input), &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"project", "42%", "│"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
}

func TestRenderCompressWarning(t *testing.T) {
	input := `{"context_window":{"context_window_size":200000,"used_percentage":85}}`
	var buf bytes.Buffer
	if err := Render(strings.NewReader(input), &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "/compact") {
		t.Errorf("expected /compact at 85%%: %q", buf.String())
	}
}

func TestRenderEmptyStdin(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(strings.NewReader(""), &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[ina]") {
		t.Errorf("expected fallback, got %q", buf.String())
	}
}

func TestRenderNoContextWindow(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(strings.NewReader(`{"cwd":"/tmp"}`), &buf); err != nil {
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
		if got := classify(tt.pct); got != tt.want {
			t.Errorf("classify(%d) = %d, want %d", tt.pct, got, tt.want)
		}
	}
}

func TestRenderBar(t *testing.T) {
	bar := renderBar(50, 8, green)
	if !strings.Contains(bar, "████") {
		t.Errorf("expected 4 filled blocks at 50%% of 8: %q", bar)
	}
}
