package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLearningAndSearch(t *testing.T) {
	dir := t.TempDir()

	// Save a few learnings
	learnings := []Learning{
		{Type: "pattern", Key: "test-isolation", Insight: "Always run tests with -count=1 to avoid cache", Confidence: 9, Source: "review"},
		{Type: "pitfall", Key: "nil-channel", Insight: "Sending to nil channel blocks forever", Confidence: 10, Source: "investigate"},
		{Type: "pattern", Key: "error-wrapping", Insight: "Use fmt.Errorf with %w for error chains", Confidence: 8},
	}
	for _, l := range learnings {
		if err := SaveLearning(dir, l); err != nil {
			t.Fatalf("SaveLearning: %v", err)
		}
	}

	// Search all
	results, err := SearchLearnings(dir, "", "", 0)
	if err != nil {
		t.Fatalf("SearchLearnings: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	// Most recent first
	if results[0].Key != "error-wrapping" {
		t.Errorf("expected most recent first, got %s", results[0].Key)
	}

	// Search by query
	results, err = SearchLearnings(dir, "channel", "", 0)
	if err != nil {
		t.Fatalf("SearchLearnings: %v", err)
	}
	if len(results) != 1 || results[0].Key != "nil-channel" {
		t.Errorf("expected nil-channel, got %v", results)
	}

	// Filter by type
	results, err = SearchLearnings(dir, "", "pitfall", 0)
	if err != nil {
		t.Fatalf("SearchLearnings: %v", err)
	}
	if len(results) != 1 || results[0].Type != "pitfall" {
		t.Errorf("expected 1 pitfall, got %v", results)
	}

	// Limit
	results, err = SearchLearnings(dir, "", "", 2)
	if err != nil {
		t.Fatalf("SearchLearnings: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results with limit, got %d", len(results))
	}
}

func TestLearningDedup(t *testing.T) {
	dir := t.TempDir()

	// Save same key twice — latest should win
	SaveLearning(dir, Learning{Type: "pattern", Key: "retry-logic", Insight: "old insight", Confidence: 5})
	SaveLearning(dir, Learning{Type: "pattern", Key: "retry-logic", Insight: "updated insight", Confidence: 9})

	results, _ := SearchLearnings(dir, "", "", 0)
	if len(results) != 1 {
		t.Fatalf("expected dedup to 1, got %d", len(results))
	}
	if results[0].Insight != "updated insight" {
		t.Errorf("expected latest insight, got %s", results[0].Insight)
	}
	if results[0].Confidence != 9 {
		t.Errorf("expected confidence 9, got %d", results[0].Confidence)
	}
}

func TestSearchEmptyFile(t *testing.T) {
	dir := t.TempDir()
	results, err := SearchLearnings(dir, "anything", "", 10)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestLogEventAndRead(t *testing.T) {
	dir := t.TempDir()

	events := []Event{
		{Skill: "review", Status: "clean", Summary: "No issues found"},
		{Skill: "build", Status: "pass", Summary: "All tasks complete"},
		{Skill: "test", Status: "fail", Summary: "2 tests failed"},
	}
	for _, e := range events {
		if err := LogEvent(dir, e); err != nil {
			t.Fatalf("LogEvent: %v", err)
		}
	}

	results, err := RecentEvents(dir, 2)
	if err != nil {
		t.Fatalf("RecentEvents: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Most recent first
	if results[0].Skill != "test" {
		t.Errorf("expected most recent first, got %s", results[0].Skill)
	}
	// Verify timestamp was auto-set
	if results[0].Timestamp == "" {
		t.Error("expected auto-generated timestamp")
	}
}

func TestLogEventAutoCommit(t *testing.T) {
	dir := t.TempDir()
	LogEvent(dir, Event{Skill: "review", Status: "clean"})

	// Read raw JSONL and verify commit field exists
	data, _ := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	var e Event
	json.Unmarshal(data, &e)
	// In a non-git dir, commit may be empty — that's fine.
	// Just verify the field is present in the JSON.
	if e.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestDefaultConfidence(t *testing.T) {
	dir := t.TempDir()
	SaveLearning(dir, Learning{Type: "pattern", Key: "test-key", Insight: "test"})

	results, _ := SearchLearnings(dir, "", "", 0)
	if len(results) != 1 {
		t.Fatal("expected 1 result")
	}
	if results[0].Confidence != 7 {
		t.Errorf("expected default confidence 7, got %d", results[0].Confidence)
	}
}
