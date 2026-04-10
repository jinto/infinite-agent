package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindClaude Kind = "claude"
	KindCodex  Kind = "codex"
)

func ValidKind(k Kind) bool {
	return k == KindClaude || k == KindCodex
}

type State string

const (
	StateRunning State = "running"
	StateStalled State = "stalled"
	StateDead    State = "dead"
	StateBlocked State = "blocked"
)

// ExitCodeContextRestart is the exit code an agent uses to request
// a fresh session due to context pressure. The daemon treats this
// differently from a crash: it starts a new session with a focused
// resume prompt instead of using --continue.
const ExitCodeContextRestart = 42

// Mutable fields are protected by mu; access through getter/setter methods.
type Agent struct {
	mu sync.RWMutex `json:"-"`

	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     Kind   `json:"kind"`
	CWD      string `json:"cwd"`
	Worktree string `json:"worktree,omitempty"`
	TaskDesc string `json:"task_desc"`

	pid                 int           `json:"-"`
	state               State         `json:"-"`
	startedAt           time.Time     `json:"-"`
	lastActive          time.Time     `json:"-"`
	restartCount        int           `json:"-"`
	contextRestartCount int           `json:"-"` // consecutive context restarts without stage advancement
	exitCode            int           `json:"-"` // exit code from last process death
	lastStage           string        `json:"-"` // pipeline stage at last context restart (circuit breaker)
	lastActiveWindow    time.Duration `json:"-"` // minimum interval between lastActive writes
	waitDone            chan struct{} `json:"-"` // closed when cmd.Wait() completes and exitCode is set
}

// Snapshot is a read-only copy safe for serialization and cross-goroutine use.
type Snapshot struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Kind                Kind      `json:"kind"`
	PID                 int       `json:"pid"`
	CWD                 string    `json:"cwd"`
	Worktree            string    `json:"worktree,omitempty"`
	State               State     `json:"state"`
	TaskDesc            string    `json:"task_desc"`
	StartedAt           time.Time `json:"started_at"`
	LastActive          time.Time `json:"last_active"`
	RestartCount        int       `json:"restart_count"`
	ContextRestartCount int       `json:"context_restart_count,omitempty"`
	ExitCode            int       `json:"exit_code,omitempty"`
}

// DefaultActiveWindow is the minimum interval between lastActive writes.
// High-frequency hooks (post-tool-use) are debounced to this cadence.
const DefaultActiveWindow = 1 * time.Second

func New(name string, kind Kind, cwd, taskDesc string) *Agent {
	now := time.Now()
	return &Agent{
		ID:               uuid.New().String(),
		Name:             name,
		Kind:             kind,
		CWD:              cwd,
		TaskDesc:         taskDesc,
		state:            StateRunning,
		startedAt:        now,
		lastActive:       now,
		lastActiveWindow: DefaultActiveWindow,
	}
}

func (a *Agent) PID() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.pid
}

func (a *Agent) GetState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

func (a *Agent) LastActive() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastActive
}

func (a *Agent) RestartCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.restartCount
}

func (a *Agent) SetPID(pid int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pid = pid
}

func (a *Agent) SetState(s State) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = s
}

func (a *Agent) SetLastActive(t time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastActive = t
}

// TouchLastActive updates lastActive only if at least lastActiveWindow has
// elapsed since the previous update. This avoids lock contention from
// high-frequency hook events (e.g. post-tool-use) that fire many times per
// second while the watcher only checks every 30s.
// Returns true if the update was applied.
func (a *Agent) TouchLastActive(now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastActiveWindow > 0 && now.Sub(a.lastActive) < a.lastActiveWindow {
		return false
	}
	a.lastActive = now
	return true
}

func (a *Agent) ExitCode() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.exitCode
}

func (a *Agent) SetExitCode(code int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.exitCode = code
}

func (a *Agent) ContextRestartCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.contextRestartCount
}

// IncrContextRestarts increments the context restart counter and checks
// the circuit breaker. If the pipeline stage hasn't advanced since the
// last context restart, it may indicate the task inherently exceeds one
// session's context. Returns the new count and whether the stage advanced.
func (a *Agent) IncrContextRestarts(currentStage string) (count int, stageAdvanced bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	stageAdvanced = a.lastStage != "" && a.lastStage != currentStage
	if stageAdvanced {
		a.contextRestartCount = 0
	}
	a.contextRestartCount++
	a.lastStage = currentStage
	return a.contextRestartCount, stageAdvanced
}

func (a *Agent) ResetContextRestarts() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.contextRestartCount = 0
	a.lastStage = ""
}

// InitWaitDone creates the channel that signals cmd.Wait() completion.
// Must be called before launching the process.
func (a *Agent) InitWaitDone() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.waitDone = make(chan struct{})
}

// SignalWaitDone closes the waitDone channel, unblocking WaitForExit.
func (a *Agent) SignalWaitDone() {
	a.mu.RLock()
	ch := a.waitDone
	a.mu.RUnlock()
	if ch != nil {
		close(ch)
	}
}

// WaitForExit blocks until cmd.Wait() has set the exit code, or until
// the timeout elapses. Returns true if the signal was received.
func (a *Agent) WaitForExit(timeout time.Duration) bool {
	a.mu.RLock()
	ch := a.waitDone
	a.mu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (a *Agent) IncrRestarts() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.restartCount++
	return a.restartCount
}

// Snapshot returns a read-only copy of the agent's current state.
func (a *Agent) Snapshot() Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return Snapshot{
		ID:                  a.ID,
		Name:                a.Name,
		Kind:                a.Kind,
		PID:                 a.pid,
		CWD:                 a.CWD,
		Worktree:            a.Worktree,
		State:               a.state,
		TaskDesc:            a.TaskDesc,
		StartedAt:           a.startedAt,
		LastActive:          a.lastActive,
		RestartCount:        a.restartCount,
		ContextRestartCount: a.contextRestartCount,
		ExitCode:            a.exitCode,
	}
}

type Registry struct {
	mu     sync.RWMutex
	agents map[string]*Agent
	byCWD  map[string]*Agent // secondary index: CWD -> Agent
}

func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*Agent),
		byCWD:  make(map[string]*Agent),
	}
}

func (r *Registry) Add(a *Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[a.ID] = a
	if a.CWD != "" {
		r.byCWD[a.CWD] = a
	}
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.agents[id]; ok {
		delete(r.byCWD, a.CWD)
	}
	delete(r.agents, id)
}

// FindByCWD returns the agent registered for the given working directory, or nil.
func (r *Registry) FindByCWD(cwd string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byCWD[cwd]
}

func (r *Registry) NameExists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.agents {
		if a.Name == name {
			return true
		}
	}
	return false
}

// FindByNameOrPrefix resolves by exact name first, then by ID prefix.
func (r *Registry) FindByNameOrPrefix(target string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Exact name match takes priority
	for _, a := range r.agents {
		if a.Name == target {
			return a
		}
	}
	for _, a := range r.agents {
		if strings.HasPrefix(a.ID, target) {
			return a
		}
	}
	return nil
}

func (r *Registry) All() []Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Snapshot, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, a.Snapshot())
	}
	return out
}

// Agents returns the live agent pointers (for daemon internal use only).
func (r *Registry) Agents() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Agent, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, a)
	}
	return out
}

// SaveToFile persists the registry to disk as JSON.
func (r *Registry) SaveToFile(path string) error {
	snapshots := r.All()
	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	f, err := os.CreateTemp(filepath.Dir(path), ".registry-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write registry: %w", err)
	}
	f.Close()

	return os.Rename(tmpPath, path)
}

// LoadFromFile restores agents from a JSON file on disk.
func (r *Registry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range snapshots {
		a := &Agent{
			ID:               s.ID,
			Name:             s.Name,
			Kind:             s.Kind,
			CWD:              s.CWD,
			Worktree:         s.Worktree,
			TaskDesc:         s.TaskDesc,
			pid:              s.PID,
			state:            s.State,
			startedAt:        s.StartedAt,
			lastActive:       s.LastActive,
			restartCount:     s.RestartCount,
			lastActiveWindow: DefaultActiveWindow,
		}
		r.agents[a.ID] = a
		if a.CWD != "" {
			r.byCWD[a.CWD] = a
		}
	}
	return nil
}
