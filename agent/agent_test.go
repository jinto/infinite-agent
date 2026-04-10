package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewAgent(t *testing.T) {
	a := New("test", KindClaude, "/tmp/test", "do something")

	if a.ID == "" {
		t.Error("ID is empty")
	}
	if a.Name != "test" {
		t.Errorf("Name = %q, want %q", a.Name, "test")
	}
	if a.Kind != KindClaude {
		t.Errorf("Kind = %q, want %q", a.Kind, KindClaude)
	}
	if a.GetState() != StateRunning {
		t.Errorf("State = %q, want %q", a.GetState(), StateRunning)
	}
}

func TestAgentSnapshot(t *testing.T) {
	a := New("snap", KindCodex, "/tmp/snap", "task")
	a.SetPID(1234)
	a.SetState(StateStalled)

	snap := a.Snapshot()

	if snap.PID != 1234 {
		t.Errorf("PID = %d, want 1234", snap.PID)
	}
	if snap.State != StateStalled {
		t.Errorf("State = %q, want %q", snap.State, StateStalled)
	}
	if snap.Name != "snap" {
		t.Errorf("Name = %q, want %q", snap.Name, "snap")
	}
}

func TestIncrRestarts(t *testing.T) {
	a := New("r", KindClaude, "/tmp", "task")

	n := a.IncrRestarts()
	if n != 1 {
		t.Errorf("IncrRestarts = %d, want 1", n)
	}
	n = a.IncrRestarts()
	if n != 2 {
		t.Errorf("IncrRestarts = %d, want 2", n)
	}
	if a.RestartCount() != 2 {
		t.Errorf("RestartCount = %d, want 2", a.RestartCount())
	}
}

func TestRegistryAddRemove(t *testing.T) {
	r := NewRegistry()
	a := New("reg", KindClaude, "/tmp", "task")

	r.Add(a)
	if len(r.All()) != 1 {
		t.Fatalf("All() len = %d, want 1", len(r.All()))
	}

	r.Remove(a.ID)
	if len(r.All()) != 0 {
		t.Fatalf("All() len = %d, want 0", len(r.All()))
	}
}

func TestRegistryFindByNameOrPrefix(t *testing.T) {
	r := NewRegistry()
	a := New("myagent", KindClaude, "/tmp", "task")
	r.Add(a)

	// Find by name
	found := r.FindByNameOrPrefix("myagent")
	if found == nil || found.ID != a.ID {
		t.Error("FindByNameOrPrefix by name failed")
	}

	// Find by ID prefix
	found = r.FindByNameOrPrefix(a.ID[:8])
	if found == nil || found.ID != a.ID {
		t.Error("FindByNameOrPrefix by ID prefix failed")
	}

	// Not found
	found = r.FindByNameOrPrefix("nonexistent")
	if found != nil {
		t.Error("FindByNameOrPrefix should return nil for unknown target")
	}
}

func TestRegistryNamePriority(t *testing.T) {
	r := NewRegistry()
	a1 := New("dead", KindClaude, "/tmp/a", "task1")
	a2 := New("other", KindClaude, "/tmp/b", "task2")
	r.Add(a1)
	r.Add(a2)

	// "dead" should match by name, not by ID prefix
	found := r.FindByNameOrPrefix("dead")
	if found == nil || found.Name != "dead" {
		t.Error("exact name match should take priority over ID prefix")
	}
}

func TestRegistrySaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r := NewRegistry()
	a := New("persist", KindClaude, "/tmp/persist", "task")
	a.SetPID(9999)
	a.SetState(StateRunning)
	r.Add(a)

	if err := r.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	data, _ := os.ReadFile(path)
	var snapshots []Snapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		t.Fatalf("Unmarshal saved file: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("saved %d agents, want 1", len(snapshots))
	}
	if snapshots[0].PID != 9999 {
		t.Errorf("saved PID = %d, want 9999", snapshots[0].PID)
	}

	r2 := NewRegistry()
	if err := r2.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	loaded := r2.All()
	if len(loaded) != 1 {
		t.Fatalf("loaded %d agents, want 1", len(loaded))
	}
	if loaded[0].Name != "persist" {
		t.Errorf("loaded Name = %q, want %q", loaded[0].Name, "persist")
	}
}

func TestValidKind(t *testing.T) {
	if !ValidKind(KindClaude) {
		t.Error("KindClaude should be valid")
	}
	if !ValidKind(KindCodex) {
		t.Error("KindCodex should be valid")
	}
	if ValidKind("invalid") {
		t.Error("'invalid' should not be valid")
	}
}

func TestExitCode(t *testing.T) {
	a := New("exit", KindClaude, "/tmp", "task")

	if a.ExitCode() != 0 {
		t.Errorf("ExitCode = %d, want 0", a.ExitCode())
	}

	a.SetExitCode(ExitCodeContextRestart)
	if a.ExitCode() != 42 {
		t.Errorf("ExitCode = %d, want 42", a.ExitCode())
	}

	snap := a.Snapshot()
	if snap.ExitCode != 42 {
		t.Errorf("Snapshot.ExitCode = %d, want 42", snap.ExitCode)
	}
}

func TestContextRestartCircuitBreaker(t *testing.T) {
	a := New("ctx", KindClaude, "/tmp", "task")

	// First context restart at stage "build:review"
	count, advanced := a.IncrContextRestarts("build:review")
	if count != 1 || advanced != false {
		t.Errorf("first: count=%d advanced=%v, want 1/false", count, advanced)
	}

	// Second at same stage — no advancement
	count, advanced = a.IncrContextRestarts("build:review")
	if count != 2 || advanced != false {
		t.Errorf("second: count=%d advanced=%v, want 2/false", count, advanced)
	}

	// Third at same stage — circuit breaker should trigger (count > maxContextRestarts)
	count, advanced = a.IncrContextRestarts("build:review")
	if count != 3 || advanced != false {
		t.Errorf("third: count=%d advanced=%v, want 3/false", count, advanced)
	}

	// Now advance to a new stage — counter resets
	count, advanced = a.IncrContextRestarts("build:commit")
	if count != 1 || advanced != true {
		t.Errorf("advanced: count=%d advanced=%v, want 1/true", count, advanced)
	}
}

func TestResetContextRestarts(t *testing.T) {
	a := New("reset", KindClaude, "/tmp", "task")
	a.IncrContextRestarts("build")
	a.IncrContextRestarts("build")

	if a.ContextRestartCount() != 2 {
		t.Errorf("ContextRestartCount = %d, want 2", a.ContextRestartCount())
	}

	a.ResetContextRestarts()
	if a.ContextRestartCount() != 0 {
		t.Errorf("after reset: ContextRestartCount = %d, want 0", a.ContextRestartCount())
	}
}

func TestConcurrentAccess(t *testing.T) {
	a := New("concurrent", KindClaude, "/tmp", "task")
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			a.SetState(StateRunning)
			a.SetPID(i)
			a.IncrRestarts()
			a.SetExitCode(i)
			a.IncrContextRestarts("stage")
		}()
		go func() {
			defer wg.Done()
			_ = a.Snapshot()
			_ = a.GetState()
			_ = a.PID()
			_ = a.ExitCode()
			_ = a.ContextRestartCount()
		}()
	}
	wg.Wait()
}
