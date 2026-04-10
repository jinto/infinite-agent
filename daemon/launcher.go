package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jinto/ina/agent"
	"github.com/jinto/ina/config"
)

func (d *Daemon) launchProcess(a *agent.Agent, fresh bool) (int, error) {
	isRestart := !fresh && a.RestartCount() > 0
	isContextRestart := fresh && a.ContextRestartCount() > 0

	var prompt string
	switch {
	case isContextRestart:
		prompt = buildContextResumePrompt(a)
	case isRestart:
		// handled below via --continue
	default:
		prompt = buildInitialPrompt(a.TaskDesc)
	}

	logDir := filepath.Join(config.DataDir(), "logs", a.Name)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return 0, fmt.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", time.Now().Format("20060102-150405")))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return 0, fmt.Errorf("create log: %w", err)
	}

	var cmd *exec.Cmd

	binary := string(a.Kind)
	if _, err := exec.LookPath(binary); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("%s binary not found in PATH: %w", binary, err)
	}

	switch a.Kind {
	case agent.KindClaude:
		baseFlags := []string{
			"--print",
			"--output-format", "stream-json",
			"--dangerously-skip-permissions",
		}
		if isRestart && !isContextRestart {
			args := append([]string{"--continue"}, baseFlags...)
			args = append(args, buildResumePrompt(a))
			cmd = exec.Command(binary, args...)
		} else {
			args := append(baseFlags, "-p", prompt)
			cmd = exec.Command(binary, args...)
		}
	case agent.KindCodex:
		cmd = exec.Command(binary,
			"exec",
			"--json",
			prompt,
		)
	default:
		logFile.Close()
		return 0, fmt.Errorf("unknown agent kind: %s", a.Kind)
	}

	cmd.Dir = a.CWD
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	a.InitWaitDone()

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("start process: %w", err)
	}

	go func() {
		cmd.Wait()
		if cmd.ProcessState != nil {
			a.SetExitCode(cmd.ProcessState.ExitCode())
		}
		a.SignalWaitDone()
		logFile.Close()
	}()

	d.logger.Printf("agent %s log: %s", a.Name, logPath)

	return cmd.Process.Pid, nil
}

func buildInitialPrompt(task string) string {
	return fmt.Sprintf(`%s

IMPORTANT: Maintain a progress file at .state/progress.md with this structure:
- YAML frontmatter with: task, agent, session_id, updated_at, status, blocked, restart_count
- Markdown sections: ## Completed, ## In Progress, ## Remaining, ## Context for Restart
- Update the file after each significant step
- If you are blocked and need human input, set blocked: true in the frontmatter
- The "Context for Restart" section should always contain enough info for another agent to continue your work
`, task)
}

func buildResumePrompt(a *agent.Agent) string {
	pipelinePath := filepath.Join(a.CWD, ".state", "pipeline.json")
	if _, err := os.Stat(pipelinePath); err == nil {
		return fmt.Sprintf(`Resume the autopilot pipeline. Read .state/pipeline.json for the current stage and context.
Also check .state/progress.md for additional context.
Continue from the recorded stage — do not restart the pipeline from the beginning.`)
	}
	return "Continue from where you left off. Check .state/progress.md for context."
}

// buildContextResumePrompt creates a focused prompt for context restart sessions.
// Unlike buildResumePrompt (used with --continue), this is used for fresh sessions
// where the previous conversation is NOT loaded. The prompt must be self-contained.
func buildContextResumePrompt(a *agent.Agent) string {
	return fmt.Sprintf(`You are resuming work after a context restart. The previous session ran out of context window space and exited cleanly after writing a checkpoint.

IMPORTANT: This is a FRESH session. You do NOT have the previous conversation. All context is in the state files below.

## What to do

1. Read .state/pipeline.json — it tells you the current stage and sub-phase
2. Read .state/progress.md — the "Context for Restart" section has everything the previous session recorded:
   - What files were changed
   - What tests passed
   - What work remains
   - Key decisions made
3. Read TASKS.md — check which tasks are done ([x]) and which remain ([ ])
4. Resume from the recorded stage. Do NOT redo completed work.

## Original task

%s

## Rules

- Maintain .state/progress.md as you work (update "Context for Restart" before each phase boundary)
- If you approach context limits again, write checkpoint and exit(42) to request another context restart
- Report progress via ina_report_progress MCP tool
`, a.TaskDesc)
}
