package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Learning represents a single insight recorded during agent work.
type Learning struct {
	Timestamp  string   `json:"ts"`
	Type       string   `json:"type"`                 // pattern, pitfall, preference, architecture
	Key        string   `json:"key"`                  // short kebab-case identifier
	Insight    string   `json:"insight"`              // one-sentence description
	Confidence int      `json:"confidence"`           // 1-10
	Source     string   `json:"source,omitempty"`      // which skill discovered this
	Files      []string `json:"files,omitempty"`
}

// Event represents a structured record of skill execution.
type Event struct {
	Timestamp string          `json:"ts"`
	Skill     string          `json:"skill"`              // review, build, test, ship
	Status    string          `json:"status"`             // clean, issues_found, pass, fail
	Summary   string          `json:"summary,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
	Commit    string          `json:"commit,omitempty"`
}

// ProjectDir returns ~/.ina/projects/{slug}/ for the current git repository.
func ProjectDir() (string, error) {
	slug := repoSlug()
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".ina", "projects", slug)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func repoSlug() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Base(cwd)
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

func currentCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func appendJSONL(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// SaveLearning appends a learning entry with auto-generated timestamp.
func SaveLearning(dir string, l Learning) error {
	l.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if l.Confidence == 0 {
		l.Confidence = 7
	}
	return appendJSONL(filepath.Join(dir, "learnings.jsonl"), l)
}

// SearchLearnings reads learnings, deduplicates by key (latest wins),
// filters by query/type, returns up to limit results (most recent first).
func SearchLearnings(dir, query, typeFilter string, limit int) ([]Learning, error) {
	path := filepath.Join(dir, "learnings.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]Learning)
	var order []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var l Learning
		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			continue
		}
		if _, exists := seen[l.Key]; !exists {
			order = append(order, l.Key)
		}
		seen[l.Key] = l
	}

	query = strings.ToLower(query)
	var results []Learning
	for i := len(order) - 1; i >= 0; i-- {
		l := seen[order[i]]
		if typeFilter != "" && l.Type != typeFilter {
			continue
		}
		if query != "" {
			combined := strings.ToLower(l.Key + " " + l.Insight + " " + l.Type)
			if !strings.Contains(combined, query) {
				continue
			}
		}
		results = append(results, l)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, nil
}

// LogEvent appends an event entry with auto-generated timestamp and commit.
func LogEvent(dir string, e Event) error {
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if e.Commit == "" {
		e.Commit = currentCommit()
	}
	return appendJSONL(filepath.Join(dir, "events.jsonl"), e)
}

// RecentEvents returns the latest events (most recent first).
func RecentEvents(dir string, limit int) ([]Event, error) {
	path := filepath.Join(dir, "events.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		all = append(all, e)
	}

	if limit <= 0 || limit > len(all) {
		limit = len(all)
	}
	results := make([]Event, limit)
	for i := 0; i < limit; i++ {
		results[i] = all[len(all)-1-i]
	}
	return results, nil
}
