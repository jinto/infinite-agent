package hud

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// StatuslineStdin is the JSON Claude Code pipes to statusline commands.
type StatuslineStdin struct {
	CWD            string         `json:"cwd"`
	TranscriptPath string         `json:"transcript_path"`
	Model          *Model         `json:"model"`
	ContextWindow  *ContextWindow `json:"context_window"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type ContextWindow struct {
	Size           int     `json:"context_window_size"`
	UsedPercentage float64 `json:"used_percentage"`
}

// Thresholds for context severity levels.
const (
	ThresholdWarning  = 70
	ThresholdCompress = 80
	ThresholdCritical = 85
)

// ANSI color codes.
const (
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	white  = "\033[37m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	reset  = "\033[0m"
)

// Unicode box-drawing separator.
const sep = dim + " │ " + reset

type severity int

const (
	sevNormal severity = iota
	sevWarning
	sevCompress
	sevCritical
)

func classify(pct int) severity {
	switch {
	case pct >= ThresholdCritical:
		return sevCritical
	case pct >= ThresholdCompress:
		return sevCompress
	case pct >= ThresholdWarning:
		return sevWarning
	default:
		return sevNormal
	}
}

func (s severity) color() string {
	switch s {
	case sevCritical:
		return red
	case sevCompress, sevWarning:
		return yellow
	default:
		return green
	}
}

// DisabledFile is the flag file that disables the HUD.
var DisabledFile = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ina", "hud_disabled")
}()

// IsDisabled checks whether the HUD is turned off.
func IsDisabled() bool {
	_, err := os.Stat(DisabledFile)
	return err == nil
}

// Render reads Claude Code's statusline stdin and writes formatted output.
// Output: project │ ████░░░░ 9%
func Render(r io.Reader, w io.Writer) error {
	if IsDisabled() {
		return nil
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		fmt.Fprintln(w, "[ina] no stdin")
		return nil
	}

	var stdin StatuslineStdin
	if err := json.Unmarshal(data, &stdin); err != nil {
		fmt.Fprintln(w, "[ina] bad stdin")
		return nil
	}

	if stdin.ContextWindow == nil {
		fmt.Fprintln(w, "[ina]")
		return nil
	}

	pct := clamp(int(math.Round(stdin.ContextWindow.UsedPercentage)), 0, 100)
	sev := classify(pct)

	c := sev.color()
	ctxBar := renderBar(pct, 8, c)
	ctxLabel := ctxBar + " " + c + fmt.Sprintf("%d%%", pct) + reset
	if pct >= ThresholdCompress {
		ctxLabel += " " + c + bold + "/compact" + reset
	}

	var parts []string
	if stdin.CWD != "" {
		parts = append(parts, white+filepath.Base(stdin.CWD)+reset)
	}
	parts = append(parts, ctxLabel)

	fmt.Fprintln(w, strings.Join(parts, sep))

	writeContextPct(pct)
	return nil
}

func renderBar(pct, width int, color string) string {
	filled := int(math.Round(float64(pct) / 100.0 * float64(width)))
	empty := width - filled
	return color + strings.Repeat("█", filled) + dim + strings.Repeat("░", empty) + reset
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ContextPctFile is where the last known context percentage is stored.
var ContextPctFile = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ina", "ctx_pct")
}()

func writeContextPct(pct int) {
	_ = os.WriteFile(ContextPctFile, []byte(fmt.Sprintf("%d", pct)), 0600)
}

// RenderFromStdin is a convenience for CLI use.
func RenderFromStdin() error {
	return Render(os.Stdin, os.Stdout)
}
