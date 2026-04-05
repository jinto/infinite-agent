# ina — Infinite Agent

Daemon-supervised coding agents that never stop.

Go daemon (process watching, crash recovery, Discord alerts) + 10 Claude Code plugin skills with autopilot pipeline orchestration.

## Skills

| Skill | Description |
|-------|-------------|
| `autopilot` | Full pipeline: think → plan → build → review → commit |
| `think` | Idea → spec (technical / business / improve) |
| `plan` | Consensus planning + TDD task breakdown |
| `build` | Execute tasks (direct / subagent / team) |
| `review` | Code review + fix-first auto-correction |
| `research` | Facet decomposition + parallel search |
| `design` | UI implementation + visual verification |
| `test` | Test runner + failure analysis + fix |
| `ship` | Create PR with auto-generated summary |
| `guard` | Safety guardrails for unattended execution |

## Pipeline

```
autopilot: think → plan → build → review → commit
                                    ↑         │
                                    └─────────┘ (loopback, max 3)
```

Crash recovery via `.state/pipeline.json` — the daemon restarts the agent and resumes from the recorded stage.

## Build

```bash
go build -o ina .
go build -o ina-mcp ./mcp/
go test ./... -count=1 -race
```

## Acknowledgements

Built with insights from:

- [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) — Multi-agent orchestration (ralph, autopilot, team, ralplan)
- [superpowers](https://github.com/obra/superpowers) — Process discipline, Iron Laws, TDD enforcement
- [gstack](https://github.com/garrytan/gstack) — Solo builder software factory, browser daemon, design system
- [agent-skills](https://github.com/addyosmani/agent-skills) — Google SWE culture encoding, anti-rationalization, scope discipline

## License

MIT
