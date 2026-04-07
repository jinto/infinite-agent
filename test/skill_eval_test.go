package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Tier 3: 유료, 수동 — LLM-Judge Eval
// 스킬 프롬프트의 품질을 자동 검증 (detection + quality + regression)
//
// 실행: INFA_EVAL=1 go test ./test/ -run TestSkillEval -v -timeout 600s
// 특정 스킬만: EVAL_SKILLS=review INFA_EVAL=1 go test ./test/ -run TestSkillEval -v

// --- Types ---

type rubricItem struct {
	Type        string `json:"type"`        // "keyword" or "judge"
	Check       string `json:"check"`       // keyword to search (keyword type only)
	Description string `json:"description"` // what this rubric checks
}

type evalScenario struct {
	Name       string       `json:"name"`
	Skill      string       `json:"skill"`
	Fixture    string       `json:"fixture"`
	Prompt     string       `json:"prompt"`
	TimeoutSec int          `json:"timeout_sec"`
	MinScore   float64      `json:"min_score"`
	Rubric     []rubricItem `json:"rubric"`
}

type judgeResponse struct {
	Score     int    `json:"score"`
	Reasoning string `json:"reasoning"`
}

type evalResult struct {
	Name      string    `json:"name"`
	Skill     string    `json:"skill"`
	Score     float64   `json:"score"`
	MinScore  float64   `json:"min_score"`
	Pass      bool      `json:"pass"`
	Details   []string  `json:"details"`
	Duration  string    `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}

type evalRun struct {
	Timestamp time.Time    `json:"timestamp"`
	Results   []evalResult `json:"results"`
}

// --- Scenario Loading ---

func loadEvalScenarios(t *testing.T) []evalScenario {
	t.Helper()
	data, err := os.ReadFile("eval_scenarios.json")
	if err != nil {
		t.Fatalf("cannot read eval_scenarios.json: %v", err)
	}
	var scenarios []evalScenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		t.Fatalf("parse eval_scenarios.json: %v", err)
	}
	return scenarios
}

func filterScenarios(scenarios []evalScenario, skills string) []evalScenario {
	if skills == "" || skills == "all" {
		return scenarios
	}
	allowed := make(map[string]bool)
	for _, s := range strings.Split(skills, ",") {
		allowed[strings.TrimSpace(s)] = true
	}
	var filtered []evalScenario
	for _, sc := range scenarios {
		if allowed[sc.Skill] {
			filtered = append(filtered, sc)
		}
	}
	return filtered
}

// --- Keyword Scoring ---

func scoreKeyword(output string, item rubricItem) (float64, string) {
	if strings.Contains(strings.ToLower(output), strings.ToLower(item.Check)) {
		return 5.0, fmt.Sprintf("KEYWORD PASS: '%s' found", item.Check)
	}
	return 0.0, fmt.Sprintf("KEYWORD FAIL: '%s' not found", item.Check)
}

// --- Judge Scoring ---

func buildJudgePrompt(skill, fixtureContent, skillOutput, rubricDesc string) string {
	return fmt.Sprintf(`당신은 AI 코딩 에이전트의 출력 품질을 채점하는 심사관입니다.

## 채점 대상
%s 스킬이 다음 입력에 대해 생성한 출력을 평가합니다.

## 채점 항목
%s

## Scoring Guide
1: 전혀 충족하지 못함
2: 언급은 있으나 부정확하거나 불완전
3: 부분적으로 충족 (핵심은 있으나 세부 부족)
4: 대부분 충족 (사소한 누락만 있음)
5: 완벽히 충족

## 입력
%s

## 스킬 출력
%s

## 응답 포맷 (JSON만, 다른 텍스트 없이)
{"score": N, "reasoning": "한 줄 근거"}`, skill, rubricDesc, fixtureContent, skillOutput)
}

func callJudge(t *testing.T, prompt string) (int, string, error) {
	t.Helper()

	cmd := exec.Command("claude", "-p", "--model", "haiku", "--output-format", "text", prompt)
	cmd.Dir = ".."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return 0, "", fmt.Errorf("start claude judge: %w", err)
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return 0, "", fmt.Errorf("claude judge error: %w (stderr: %s)", err, stderr.String())
		}
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return 0, "", fmt.Errorf("claude judge timed out")
	}

	return parseJudgeResponse(stdout.String())
}

func parseJudgeResponse(raw string) (int, string, error) {
	raw = strings.TrimSpace(raw)

	// Try to extract JSON from response (may have markdown wrapping)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return 0, "", fmt.Errorf("no JSON found in judge response: %s", truncate(raw, 200))
	}

	jsonStr := raw[start : end+1]
	var resp judgeResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return 0, "", fmt.Errorf("parse judge JSON: %w (raw: %s)", err, truncate(jsonStr, 200))
	}

	if resp.Score < 1 || resp.Score > 5 {
		return 0, "", fmt.Errorf("judge score out of range: %d", resp.Score)
	}

	return resp.Score, resp.Reasoning, nil
}

func scoreJudge(t *testing.T, skill, fixtureContent, skillOutput string, item rubricItem) (float64, string) {
	t.Helper()

	prompt := buildJudgePrompt(skill, fixtureContent, skillOutput, item.Description)

	// Try up to 2 times (1 retry on parse failure)
	for attempt := 0; attempt < 2; attempt++ {
		score, reasoning, err := callJudge(t, prompt)
		if err != nil {
			if attempt == 0 {
				t.Logf("judge attempt 1 failed, retrying: %v", err)
				continue
			}
			t.Logf("judge failed after retry: %v", err)
			return 0.0, fmt.Sprintf("JUDGE ERROR: %v", err)
		}
		return float64(score), fmt.Sprintf("JUDGE %d/5: %s (%s)", score, item.Description, reasoning)
	}

	return 0.0, "JUDGE ERROR: unreachable"
}

// --- Skill Execution ---

func runSkillEval(t *testing.T, sc evalScenario) string {
	t.Helper()

	fixtureContent, err := os.ReadFile(filepath.Join("..", sc.Fixture))
	if err != nil {
		t.Fatalf("read fixture %s: %v", sc.Fixture, err)
	}

	prompt := fmt.Sprintf(`%s

다음 파일의 내용:
---
%s
---

파일 경로: %s`, sc.Prompt, string(fixtureContent), sc.Fixture)

	timeout := time.Duration(sc.TimeoutSec) * time.Second
	cmd := exec.Command("claude", "-p", "--output-format", "text", prompt)
	cmd.Dir = ".."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start claude: %v", err)
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("stderr: %s", stderr.String())
			t.Fatalf("claude exited with error: %v", err)
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		t.Fatalf("claude timed out after %v", timeout)
	}

	return stdout.String()
}

// --- Result Persistence ---

func saveEvalResults(t *testing.T, run evalRun) {
	t.Helper()

	dir := filepath.Join("..", ".state", "eval")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("cannot create eval dir: %v", err)
		return
	}

	filename := fmt.Sprintf("%s.json", run.Timestamp.Format("2006-01-02T15-04-05"))
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Logf("marshal eval results: %v", err)
		return
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Logf("write eval results: %v", err)
		return
	}

	// Also write as "latest.json" for easy comparison
	latestPath := filepath.Join(dir, "latest.json")
	os.WriteFile(latestPath, data, 0o644)

	t.Logf("eval results saved to %s", path)
}

func loadPreviousResults(t *testing.T) *evalRun {
	t.Helper()

	path := filepath.Join("..", ".state", "eval", "latest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // no previous results (bootstrap)
	}

	var run evalRun
	if err := json.Unmarshal(data, &run); err != nil {
		t.Logf("parse previous results: %v", err)
		return nil
	}
	return &run
}

func checkRegression(t *testing.T, current evalResult, previous *evalRun) {
	t.Helper()
	if previous == nil {
		t.Logf("no previous results — bootstrap mode, skipping regression check")
		return
	}

	for _, prev := range previous.Results {
		if prev.Name == current.Name {
			delta := current.Score - prev.Score
			if delta < -1.0 {
				t.Errorf("REGRESSION: %s score dropped %.1f → %.1f (delta: %.1f, tolerance: ±1.0)",
					current.Name, prev.Score, current.Score, delta)
			} else if delta < 0 {
				t.Logf("minor score decrease: %s %.1f → %.1f (within ±1.0 tolerance)",
					current.Name, prev.Score, current.Score)
			}
			return
		}
	}
	t.Logf("new scenario %s — no previous baseline", current.Name)
}

// --- Main Test ---

func TestSkillEval(t *testing.T) {
	if os.Getenv("INFA_EVAL") == "" {
		t.Skip("LLM-Judge eval disabled. Set INFA_EVAL=1 to run (costs API credits).")
	}

	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found in PATH")
	}

	scenarios := loadEvalScenarios(t)
	scenarios = filterScenarios(scenarios, os.Getenv("EVAL_SKILLS"))

	if len(scenarios) == 0 {
		t.Skip("no scenarios match EVAL_SKILLS filter")
	}

	previous := loadPreviousResults(t)

	run := evalRun{
		Timestamp: time.Now(),
	}

	t.Logf("running %d eval scenarios", len(scenarios))

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			start := time.Now()

			// Read fixture for judge context
			fixtureContent, err := os.ReadFile(filepath.Join("..", sc.Fixture))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			// Execute skill
			t.Logf("executing skill: %s (fixture: %s)", sc.Skill, sc.Fixture)
			output := runSkillEval(t, sc)
			t.Logf("skill output: %s", truncate(output, 500))

			// Score rubric items
			var totalScore float64
			var details []string
			for _, item := range sc.Rubric {
				var score float64
				var detail string
				switch item.Type {
				case "keyword":
					score, detail = scoreKeyword(output, item)
				case "judge":
					score, detail = scoreJudge(t, sc.Skill, string(fixtureContent), output, item)
				default:
					t.Errorf("unknown rubric type: %s", item.Type)
					continue
				}
				totalScore += score
				details = append(details, detail)
				t.Logf("  %s", detail)
			}

			avgScore := totalScore / float64(len(sc.Rubric))
			pass := avgScore >= sc.MinScore

			result := evalResult{
				Name:      sc.Name,
				Skill:     sc.Skill,
				Score:     avgScore,
				MinScore:  sc.MinScore,
				Pass:      pass,
				Details:   details,
				Duration:  time.Since(start).Round(time.Millisecond).String(),
				Timestamp: time.Now(),
			}

			run.Results = append(run.Results, result)

			// Check regression
			checkRegression(t, result, previous)

			if !pass {
				t.Errorf("FAIL: %s — score %.1f < min %.1f", sc.Name, avgScore, sc.MinScore)
			} else {
				t.Logf("PASS: %s — score %.1f (min %.1f)", sc.Name, avgScore, sc.MinScore)
			}
		})
	}

	// Save results
	saveEvalResults(t, run)

	// Print summary table
	t.Log("\n=== Eval Summary ===")
	t.Logf("%-30s %-8s %-8s %-8s %s", "Scenario", "Score", "Min", "Result", "Duration")
	t.Logf("%-30s %-8s %-8s %-8s %s", "--------", "-----", "---", "------", "--------")
	for _, r := range run.Results {
		status := "PASS"
		if !r.Pass {
			status = "FAIL"
		}
		t.Logf("%-30s %-8.1f %-8.1f %-8s %s", r.Name, r.Score, r.MinScore, status, r.Duration)
	}
}

// --- Helpers ---

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// --- Unit Tests for Internal Logic ---

func TestParseJudgeResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		score   int
		wantErr bool
	}{
		{"valid json", `{"score": 4, "reasoning": "good"}`, 4, false},
		{"wrapped in markdown", "```json\n{\"score\": 3, \"reasoning\": \"ok\"}\n```", 3, false},
		{"with extra text", "Here is my evaluation:\n{\"score\": 5, \"reasoning\": \"perfect\"}", 5, false},
		{"no json", "This is just text", 0, true},
		{"invalid score", `{"score": 6, "reasoning": "too high"}`, 0, true},
		{"zero score", `{"score": 0, "reasoning": "invalid"}`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _, err := parseJudgeResponse(tt.input)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && score != tt.score {
				t.Errorf("score = %d, want %d", score, tt.score)
			}
		})
	}
}

func TestScoreKeyword(t *testing.T) {
	tests := []struct {
		name   string
		output string
		check  string
		score  float64
	}{
		{"found exact", "Found SQL injection vulnerability", "SQL injection", 5.0},
		{"found case insensitive", "found sql INJECTION", "SQL injection", 5.0},
		{"not found", "Everything looks fine", "SQL injection", 0.0},
		{"empty output", "", "SQL injection", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := rubricItem{Type: "keyword", Check: tt.check}
			score, _ := scoreKeyword(tt.output, item)
			if score != tt.score {
				t.Errorf("score = %.1f, want %.1f", score, tt.score)
			}
		})
	}
}

func TestFilterScenarios(t *testing.T) {
	scenarios := []evalScenario{
		{Name: "a", Skill: "review"},
		{Name: "b", Skill: "plan"},
		{Name: "c", Skill: "review"},
		{Name: "d", Skill: "build"},
	}

	tests := []struct {
		name   string
		skills string
		want   int
	}{
		{"empty = all", "", 4},
		{"all = all", "all", 4},
		{"review only", "review", 2},
		{"review,plan", "review,plan", 3},
		{"nonexistent", "ship", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterScenarios(scenarios, tt.skills)
			if len(got) != tt.want {
				t.Errorf("got %d scenarios, want %d", len(got), tt.want)
			}
		})
	}
}

func TestCheckRegression(t *testing.T) {
	// This test verifies the regression logic without calling t.Errorf directly,
	// since we want to test the detection behavior.
	current := evalResult{Name: "test_scenario", Score: 3.5}

	// No previous = bootstrap, no error
	checkRegression(t, current, nil)

	// Within tolerance
	prev := &evalRun{Results: []evalResult{{Name: "test_scenario", Score: 4.0}}}
	checkRegression(t, current, prev)

	// New scenario not in previous
	newScenario := evalResult{Name: "new_scenario", Score: 3.0}
	checkRegression(t, newScenario, prev)
}
