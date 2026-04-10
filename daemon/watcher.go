package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jinto/ina/agent"
	"github.com/jinto/ina/state"
)

func (d *Daemon) watchAgent(a *agent.Agent) {
	defer d.wg.Done()

	interval := d.cfg.Daemon.CheckIntervalDuration()
	threshold := d.cfg.Daemon.IdleThresholdDuration()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.checkAgent(a, threshold)

			if a.GetState() == agent.StateDead {
				d.handleDeath(a)
				return
			}
		}
	}
}

func (d *Daemon) checkAgent(a *agent.Agent, threshold time.Duration) {
	pid := a.PID()

	if !agent.IsAlive(pid) {
		a.SetState(agent.StateDead)
		d.logger.Printf("agent %s (pid=%d) died", a.Name, pid)
		return
	}

	progress, err := state.Read(a.CWD)
	if err == nil && progress.Blocked {
		if a.GetState() != agent.StateBlocked {
			a.SetState(agent.StateBlocked)
			d.notifier.AgentBlocked(a.Snapshot())
			d.logger.Printf("agent %s is blocked", a.Name)
		}
		return
	}

	lastActive := agent.LatestActivity(a.CWD)
	if !lastActive.IsZero() {
		a.SetLastActive(lastActive)
	}

	if time.Since(a.LastActive()) > threshold {
		if a.GetState() != agent.StateStalled {
			a.SetState(agent.StateStalled)
			d.notifier.AgentStalled(a.Snapshot())
			d.logger.Printf("agent %s stalled (no activity for %v)", a.Name, threshold)
		}
	} else {
		a.SetState(agent.StateRunning)
	}
}

// maxContextRestarts is the maximum consecutive context restarts without
// stage advancement before the daemon gives up.
const maxContextRestarts = 2

func (d *Daemon) handleDeath(a *agent.Agent) {
	// Wait for cmd.Wait() goroutine to set the exit code.
	// Without this, we might read exitCode=0 and miss a context restart request.
	if !a.WaitForExit(5 * time.Second) {
		d.logger.Printf("agent %s: timed out waiting for exit code, proceeding with exitCode=%d", a.Name, a.ExitCode())
	}

	snap := a.Snapshot()

	// Check if this was a context restart request (exit code 42).
	if a.ExitCode() == agent.ExitCodeContextRestart {
		d.handleContextRestart(a)
		return
	}

	d.notifier.AgentDied(snap)

	// Normal death — reset context restart counter since this wasn't a context exit.
	a.ResetContextRestarts()

	if !d.cfg.Daemon.AutoRestart {
		d.logger.Printf("auto-restart disabled, agent %s stays dead", a.Name)
		d.abandonAgent(a)
		return
	}

	if snap.RestartCount >= d.cfg.Daemon.MaxRestarts {
		d.notifier.Send(fmt.Sprintf("Agent **%s** exceeded max restarts (%d). Manual intervention needed.", a.Name, d.cfg.Daemon.MaxRestarts))
		d.logger.Printf("agent %s exceeded max restarts", a.Name)
		d.abandonAgent(a)
		return
	}

	d.logger.Printf("auto-restarting agent %s (attempt %d/%d)", a.Name, snap.RestartCount+1, d.cfg.Daemon.MaxRestarts)

	if err := d.restartAgent(a, false); err != nil {
		d.logger.Printf("restart failed: %v", err)
		d.notifier.Send(fmt.Sprintf("Failed to restart agent **%s**: %v", a.Name, err))
		d.abandonAgent(a)
		return
	}

	d.wg.Add(1)
	go d.watchAgent(a)
}

func (d *Daemon) handleContextRestart(a *agent.Agent) {
	stage := readPipelineStage(a.CWD)
	if stage == "" {
		d.logger.Printf("agent %s: no pipeline.json found, circuit breaker will count without stage tracking", a.Name)
	}
	count, stageAdvanced := a.IncrContextRestarts(stage)

	d.logger.Printf("agent %s requested context restart (stage=%s, count=%d, advanced=%v)",
		a.Name, stage, count, stageAdvanced)

	// Circuit breaker: if stuck at the same stage, abort.
	if count > maxContextRestarts && !stageAdvanced {
		msg := fmt.Sprintf("Agent **%s** requested %d context restarts at stage `%s` without advancing. Aborting.",
			a.Name, count, stage)
		d.notifier.Send(msg)
		d.logger.Printf("circuit breaker: %s", msg)
		d.abandonAgent(a)
		return
	}

	d.notifier.Send(fmt.Sprintf("Agent **%s** context restart → fresh session (stage=%s, attempt=%d)",
		a.Name, stage, count))

	if err := d.restartContextAgent(a); err != nil {
		d.logger.Printf("context restart failed: %v", err)
		d.notifier.Send(fmt.Sprintf("Failed to context-restart agent **%s**: %v", a.Name, err))
		d.abandonAgent(a)
		return
	}

	d.wg.Add(1)
	go d.watchAgent(a)
}

// readPipelineStage reads the current stage from .state/pipeline.json.
// Returns empty string if the file doesn't exist or can't be parsed.
func readPipelineStage(cwd string) string {
	data, err := os.ReadFile(filepath.Join(cwd, ".state", "pipeline.json"))
	if err != nil {
		return ""
	}
	var p struct {
		Stage    string `json:"stage"`
		SubPhase string `json:"sub_phase"`
	}
	if json.Unmarshal(data, &p) != nil {
		return ""
	}
	if p.SubPhase != "" {
		return p.Stage + ":" + p.SubPhase
	}
	return p.Stage
}

// abandonAgent cleans up and removes an agent that won't be restarted.
func (d *Daemon) abandonAgent(a *agent.Agent) {
	d.cleanupWorktree(a)
	d.removeAgent(a.ID)
}

// restartContextAgent restarts the agent for a context restart.
// Unlike restartAgent(fresh=true), it does NOT call state.Init(),
// preserving the checkpoint files (progress.md, pipeline.json).
func (d *Daemon) restartContextAgent(a *agent.Agent) error {
	oldPID := a.PID()
	if agent.IsAlive(oldPID) {
		syscall.Kill(oldPID, syscall.SIGTERM)
		time.Sleep(2 * time.Second)
		if agent.IsAlive(oldPID) {
			syscall.Kill(oldPID, syscall.SIGKILL)
		}
	}

	// Do NOT call state.Init() — preserve the checkpoint.
	// Do NOT call IncrRestarts() — context restarts don't count against MaxRestarts.

	pid, err := d.launchProcess(a, true)
	if err != nil {
		return fmt.Errorf("launch: %w", err)
	}

	a.SetPID(pid)
	a.SetState(agent.StateRunning)
	a.SetLastActive(time.Now())

	snap := a.Snapshot()
	d.notifier.AgentRestarted(snap)
	d.logger.Printf("agent %s context-restarted (pid=%d, ctx_restarts=%d)", a.Name, snap.PID, snap.ContextRestartCount)

	return nil
}

func (d *Daemon) cleanupWorktree(a *agent.Agent) {
	if a.Worktree == "" {
		return
	}
	removeWorktree(filepath.Dir(filepath.Dir(a.Worktree)), a.Worktree)
	d.logger.Printf("cleaned up worktree %s", a.Worktree)
}

func (d *Daemon) restartAgent(a *agent.Agent, fresh bool) error {
	oldPID := a.PID()
	if agent.IsAlive(oldPID) {
		syscall.Kill(oldPID, syscall.SIGTERM)
		time.Sleep(2 * time.Second)
		if agent.IsAlive(oldPID) {
			syscall.Kill(oldPID, syscall.SIGKILL)
		}
	}

	a.IncrRestarts()

	if fresh {
		if err := state.Init(a.CWD, a.TaskDesc, string(a.Kind)); err != nil {
			d.logger.Printf("warning: reset state file: %v", err)
		}
	}

	pid, err := d.launchProcess(a, fresh)
	if err != nil {
		return fmt.Errorf("launch: %w", err)
	}

	a.SetPID(pid)
	a.SetState(agent.StateRunning)
	a.SetLastActive(time.Now())

	snap := a.Snapshot()
	d.notifier.AgentRestarted(snap)
	d.logger.Printf("agent %s restarted (pid=%d, attempt=%d)", a.Name, snap.PID, snap.RestartCount)

	return nil
}
