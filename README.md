# ina — Infinite Angel

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[한국어](README.ko.md)

**Daemon-supervised coding angels that never stop.**

_Launch an agent. Go to sleep. Wake up to working code._

[Quick Start](#quick-start) • [Skills](#skills) • [Pipeline](#pipeline) • [Acknowledgements](#acknowledgements)

---

## Quick Start

```bash
# 1. Install (binaries + auto-configuration)
curl -sSL https://raw.githubusercontent.com/jinto/infinite-angel/main/install.sh | sh
source ~/.zshrc  # or open a new terminal

# 2. Register daemon (auto-start on login)
ina install

# 3. Install skills (in Claude Code)
/plugin marketplace add https://github.com/jinto/infinite-angel
/plugin install ina

# 4. Example
/ina:autopilot Implement a login feature.

# Uninstall (removes daemon, hooks, HUD, MCP server)
ina uninstall          # keep config (~/.ina)
ina uninstall --purge  # remove everything
```

---

## What makes ina different

Other Claude Code plugins are **prompt libraries**. ina is **infrastructure**.

| | Prompt libraries | ina |
|---|---|---|
| Agent crashes | Gone forever | **Daemon restarts + resumes** |
| Context resets | Start over | **Pipeline state preserved** |
| Long-running tasks | Hope for the best | **Monitored + alerts** |
| Multiple agents | Manual | **Registry + coordination** |

---

## Skills

```
  EXPLORE       THINK         PLAN          BUILD (impl→review→commit)       SHIP
 ┌────────┐   ┌──────┐     ┌──────┐     ┌──────────────────────────┐     ┌──────┐
 │ Build  │   │ Idea │     │ Plan │     │ Code → 3-lane Review     │     │  PR  │
 │ it?    │──▶│ Spec │ ──▶ │Tasks │ ──▶ │ → Fix-first → Commit     │ ──▶ │Merge │
 └────────┘   └──────┘     └──────┘     └──────────────────────────┘     └──────┘
 GO/NO-GO/
  PIVOT
```

| Skill | Description |
|-------|-------------|
| `explore` | "Should I build this?" — inline market research + GO / NO-GO / PIVOT verdict |
| `autopilot` | Full pipeline: think → plan → build |
| `think` | Idea → spec (technical / business / improve) |
| `rethink` | Codebase health check — full scan + codex review + improvement plan |
| `plan` | Consensus planning + TDD task breakdown |
| `build` | Implement → review → commit in one shot (3-lane review built-in) |
| `review` | Standalone 3-lane review (adversarial + security + simplify) |
| `research` | Multi-angle decomposition + parallel search |
| `design` | UI implementation + visual verification |
| `test` | Test runner + failure analysis + fix |
| `ship` | Create PR with auto-generated summary |
| `guard` | Safety guardrails for unattended execution |

Don't know which skill to use? Just describe what you want — ina auto-selects the right skill via the built-in reference guide.

---

## Usage Scenarios

### "Should I even build this?"

```
/ina:explore I'm thinking of a SaaS tool for async code review
```

Detects Startup or Builder mode → questions with inline WebSearch → competitive landscape → premise challenge → GO / NO-GO / PIVOT verdict saved to `.ina/specs/`.

### "I have a vague idea"

```
/ina:think I want to add user authentication
```

Socratic interview → multi-perspective validation (Architect/Critic/CEO) → spec document saved to `.ina/specs/`.

### "Is this code any good?"

```
/ina:rethink
```

Full codebase scan + parallel codex review → "If I rebuilt this from scratch, what would I do differently?" → improvement plan saved to `.ina/specs/`. No code modified.

### "I have a spec, need a plan"

```
/ina:plan .ina/specs/20260405-1000-think-auth.md
```

Consensus planning (Planner → Architect → Critic) → TDD task breakdown → TASKS.md.

### "I have tasks, just build it"

```
/ina:build
```

Implement → 3-lane review → commit, all in one shot. Auto-delegates: direct for 1 task, subagent for 2-3, team for 4+.

### "Do everything from scratch"

```
/ina:autopilot Add user authentication with OAuth2
```

Full pipeline: think → plan → build (with review + commit). Crash-recoverable via `.state/pipeline.json`.

### "Review before commit"

```
/ina:review
```

Parallel 3-lane review (adversarial + security + simplify) + fix-first auto-correction.

### "Run tests and fix failures"

```
/ina:test
```

Root cause analysis + fix + re-run (max 5 cycles).

### "Create a PR"

```
/ina:ship
```

Auto-generates summary from git log/diff + runs tests before PR creation.

---

## Pipeline

```
explore ──GO──▶ think → plan → build → review → commit
   │                              ↑         │
PIVOT/NO-GO                       └─────────┘ (loopback max 3)
   │
   ▼
 stop or
re-explore
```

**Crash recovery** via `.state/pipeline.json` — the daemon restarts the agent and resumes from the recorded stage.

**Multi-perspective validation** — think and plan invoke Architect, Critic, and CEO subagents in parallel to validate specs and plans before execution.

**Execution delegation** — build auto-selects direct execution, subagent parallelism, or team coordination based on task count and independence.

**Plan sizing** — plans target 3-7 tasks. Larger scopes are split into multiple plans tracked as sections in TASKS.md. Build processes only the active section.

---

## Generated Files

```
.ina/specs/                                ← specs, verdicts, analysis
├── {YYYYMMDD-HHMM}-explore-{slug}.md
├── {YYYYMMDD-HHMM}-think-{slug}.md
└── {YYYYMMDD-HHMM}-rethink-{slug}.md

.claude/plans/{slug}.md                    ← execution plans
TASKS.md                                   ← task checklist (sectioned by plan)

docs/{YYYYMMDD-HHMM}-research-{slug}.md    ← research documents

.state/pipeline.json                       ← autopilot crash recovery
```

---

## Architecture

```
┌─────────────────────────────────────────────┐
│  Skill Layer (SKILL.md)                     │
│  In-session orchestration via               │
│  Claude Code native tools                   │
│                                             │
│  explore / autopilot / think / rethink /    │
│  plan / build / review / research /         │
│  design / test / ship / guard               │
├─────────────────────────────────────────────┤
│  Daemon Layer (Go binary)                   │
│  Out-of-session process supervision         │
│  Crash recovery, restart, Discord alerts    │
│                                             │
│  ina daemon / watcher / hooks / MCP         │
└─────────────────────────────────────────────┘
```

---

## CLI Reference

| Command | Description |
|---------|-------------|
| `ina setup` | Install Claude Code hooks and MCP server |
| `ina install` | Register daemon as macOS launch agent (auto-start on login) |
| `ina uninstall` | Remove daemon, hooks, HUD, and MCP server |
| `ina upgrade` | Upgrade ina to the latest version |
| `ina status` | Show all tracked agents |
| `ina launch <path> <task>` | Launch a new agent on a project |
| `ina restart <name\|id>` | Restart a dead or stalled agent |
| `ina stop <name\|id>` | Stop a specific agent |
| `ina attach <name>` | Attach to an agent's live log output |
| `ina log [agent-name]` | View agent or daemon logs |
| `ina hud on\|off` | Toggle HUD statusline in Claude Code |
| `ina daemon` | Start the watchdog daemon (usually managed by launchd) |

---

## Build & Test

```bash
go build -o ina .
go build -o ina-mcp ./mcp/
go test ./... -count=1 -race

# E2E skill routing tests (costs API credits)
INFA_E2E=1 go test ./test/ -run TestSkillRouting -v

# LLM-Judge eval — skill quality verification (costs API credits)
INFA_EVAL=1 go test ./test/ -run TestSkillEval -v -timeout 600s
```

---

## Acknowledgements

Built with insights from:

- [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) — Multi-agent orchestration
- [superpowers](https://github.com/obra/superpowers) — Process discipline + TDD enforcement
- [gstack](https://github.com/garrytan/gstack) — Solo builder software factory
- [agent-skills](https://github.com/addyosmani/agent-skills) — Google SWE culture encoding

---

## License

[MIT](LICENSE)
