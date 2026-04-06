package hud

import (
	"bytes"
	"strings"
	"testing"
)

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
	if !strings.Contains(out, "67%") {
		t.Errorf("expected 67%% in output, got %q", out)
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

func TestRenderFull(t *testing.T) {
	stdin := &StatuslineStdin{
		CWD:           "/home/user/project",
		Model:         &Model{DisplayName: "Opus 4.6 (1M context)"},
		ContextWindow: &ContextWindow{Size: 200000, UsedPercentage: 42},
		Cost:          &Cost{TotalCostUSD: 2.07},
		RateLimits: &RateLimits{
			FiveHour: &RateLimit{UsedPercentage: 23.4},
		},
	}
	out := renderFull(stdin, 42, sevNormal)

	for _, want := range []string{"project", "42%", "│"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderFull missing %q in %q", want, out)
		}
	}
}

func TestRenderFullCompressWarning(t *testing.T) {
	stdin := &StatuslineStdin{
		ContextWindow: &ContextWindow{Size: 200000, UsedPercentage: 85},
	}
	out := renderFull(stdin, 85, sevCritical)
	if !strings.Contains(out, "/compact") {
		t.Errorf("expected /compact at 85%%: %q", out)
	}
}

func TestRenderCompact(t *testing.T) {
	stdin := &StatuslineStdin{
		Model:         &Model{DisplayName: "Opus"},
		ContextWindow: &ContextWindow{Size: 200000, UsedPercentage: 42},
		Cost:          &Cost{TotalCostUSD: 0.12},
		RateLimits: &RateLimits{
			FiveHour: &RateLimit{UsedPercentage: 23.4},
		},
	}
	out := renderCompact(stdin, 42, sevNormal)

	for _, want := range []string{"42%", "5h:23%"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderCompact missing %q in %q", want, out)
		}
	}
	// compact should NOT have │ separators
	if strings.Contains(out, "│") {
		t.Errorf("compact should not use │ separator: %q", out)
	}
}

func TestRenderCompactWarningIcon(t *testing.T) {
	stdin := &StatuslineStdin{
		ContextWindow: &ContextWindow{Size: 200000, UsedPercentage: 92},
	}
	out := renderCompact(stdin, 92, sevCritical)
	if !strings.Contains(out, "!!") {
		t.Errorf("expected !! icon at 92%%: %q", out)
	}
}

func TestRateLimitColor(t *testing.T) {
	if c := rateLimitColor(20); c != green {
		t.Errorf("expected green at 20%%, got %q", c)
	}
	if c := rateLimitColor(55); c != yellow {
		t.Errorf("expected yellow at 55%%, got %q", c)
	}
	if c := rateLimitColor(85); c != red {
		t.Errorf("expected red at 85%%, got %q", c)
	}
}

func TestShortModel(t *testing.T) {
	tests := []struct {
		display string
		want    string
	}{
		{"Opus 4.6 (1M context)", "Opus"},
		{"Sonnet", "Sonnet"},
		{"Haiku 4.5", "Haiku"},
		{"", "ina"},
	}
	for _, tt := range tests {
		var m *Model
		if tt.display != "" {
			m = &Model{DisplayName: tt.display}
		}
		got := shortModel(&StatuslineStdin{Model: m})
		if got != tt.want {
			t.Errorf("shortModel(%q) = %q, want %q", tt.display, got, tt.want)
		}
	}
}

func TestFormatResetTime(t *testing.T) {
	// "now" for past timestamps
	if got := formatResetTime(0); got != "now" {
		t.Errorf("expected 'now', got %q", got)
	}
}

func TestRenderBar(t *testing.T) {
	bar := renderBar(50, 8, green)
	if !strings.Contains(bar, "████") {
		t.Errorf("expected 4 filled blocks at 50%% of 8: %q", bar)
	}
}
