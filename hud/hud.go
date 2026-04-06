package hud

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jinto/ina/config"
)

// StatuslineStdin is the JSON Claude Code pipes to statusline commands.
type StatuslineStdin struct {
	CWD            string         `json:"cwd"`
	TranscriptPath string         `json:"transcript_path"`
	Model          *Model         `json:"model"`
	ContextWindow  *ContextWindow `json:"context_window"`
	Cost           *Cost          `json:"cost"`
	RateLimits     *RateLimits    `json:"rate_limits"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type ContextWindow struct {
	Size           int     `json:"context_window_size"`
	UsedPercentage float64 `json:"used_percentage"`
}

type Cost struct {
	TotalCostUSD float64 `json:"total_cost_usd"`
}

type RateLimits struct {
	FiveHour *RateLimit `json:"five_hour"`
	SevenDay *RateLimit `json:"seven_day"`
}

type RateLimit struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
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
	cyan   = "\033[36m"
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

	cfg, _ := config.Load()
	compact := cfg.HUD.IsCompact()

	pct := clamp(int(math.Round(stdin.ContextWindow.UsedPercentage)), 0, 100)
	sev := classify(pct)

	if compact {
		fmt.Fprintln(w, renderCompact(&stdin, pct, sev))
	} else {
		fmt.Fprintln(w, renderFull(&stdin, pct, sev))
	}

	writeContextPct(pct)
	return nil
}

// renderCompact: minimal single line
// ▓▓░░░░6% 5h:23%
func renderCompact(stdin *StatuslineStdin, pct int, sev severity) string {
	c := sev.color()
	bar := renderBar(pct, 6, c)

	parts := []string{
		bar + " " + c + fmt.Sprintf("%d%%", pct) + reset,
	}

	if stdin.RateLimits != nil {
		if rl := stdin.RateLimits.FiveHour; rl != nil {
			p := int(math.Round(rl.UsedPercentage))
			parts = append(parts, rateLimitCompact("5h", p))
		}
	}
	if pct >= ThresholdCompress {
		icon := "!"
		if pct >= 90 {
			icon = "!!"
		}
		parts = append(parts, sev.color()+bold+icon+reset)
	}

	return strings.Join(parts, " ")
}

// renderFull: single line
// infinite-agent │ ████░░░░ 9%
func renderFull(stdin *StatuslineStdin, pct int, sev severity) string {
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

	return strings.Join(parts, sep)
}

func renderBar(pct, width int, color string) string {
	filled := int(math.Round(float64(pct) / 100.0 * float64(width)))
	empty := width - filled
	return color + strings.Repeat("█", filled) + dim + strings.Repeat("░", empty) + reset
}

func shortModel(stdin *StatuslineStdin) string {
	if stdin.Model == nil || stdin.Model.DisplayName == "" {
		return "ina"
	}
	name := stdin.Model.DisplayName
	if i := strings.IndexByte(name, ' '); i > 0 {
		return name[:i]
	}
	return name
}

func rateLimitCompact(label string, pct int) string {
	c := rateLimitColor(pct)
	return fmt.Sprintf("%s%s:%d%%%s", c, label, pct, reset)
}

func rateLimitFull(label string, rl *RateLimit) string {
	pct := int(math.Round(rl.UsedPercentage))
	c := rateLimitColor(pct)
	bar := renderBar(pct, 4, c)
	s := dim + label + reset + " " + bar + " " + c + fmt.Sprintf("%d%%", pct) + reset
	if rl.ResetsAt > 0 {
		s += " " + dim + "(" + formatResetTime(rl.ResetsAt) + ")" + reset
	}
	return s
}

func rateLimitColor(pct int) string {
	if pct >= 80 {
		return red
	}
	if pct >= 50 {
		return yellow
	}
	return green
}

func formatResetTime(unix int64) string {
	d := time.Until(time.Unix(unix, 0))
	if d <= 0 {
		return "now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("resets in %dh %dm", h, m)
	}
	return fmt.Sprintf("resets in %dm", m)
}

func renderWarning(pct int) string {
	if pct < ThresholdCompress {
		return ""
	}
	icon := "!"
	c := yellow
	if pct >= 90 {
		icon = "!!"
		c = red
	}
	return fmt.Sprintf("%s%s[%s] ctx %d%% >= %d%% — run /compact%s",
		c, bold, icon, pct, ThresholdCompress, reset)
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
