# ina — Infinite Agent

Coding agents that never stop. 에이전트 감시, 재시작, 오케스트레이션 데몬 + 10개 스킬 시스템.

## Skills

- `/ina:autopilot` — 파이프라인 오케스트레이터 (think → plan → build → review → commit)
- `/ina:think` — 소크라틱 인터뷰로 기술 아이디어 → 스펙
- `/ina:plan` — 합의 기반 플래닝 + TDD 태스크 분해
- `/ina:build` — TASKS.md 태스크 구현 (직접/subagent/team 자동 선택)
- `/ina:review` — 멀티모델 코드 리뷰 + fix-first 자동 수정
- `/ina:research` — 다각도 분해 + 병렬 검색
- `/ina:design` — 프레임워크 감지 + 구현 + 시각적 검증
- `/ina:test` — 독립 테스트 실행 + 실패 분석 + root cause 수정
- `/ina:ship` — PR 생성 (변경 요약 + 테스트 확인)
- `/ina:guard` — 무인 실행 안전장치 (위험 명령 차단, blast radius)

### Pipeline

```
autopilot: think → plan → build → review → commit
                                    ↑         │
                                    └─────────┘ (루프백 최대 3회)
```

상태 파일: `.state/pipeline.json` — 크래시 복구용

## Skill I/O Convention

| From | Output Path | To | Input |
|------|------------|-----|-------|
| think | `.omc/specs/think-{slug}.md` | plan | 스펙 파일 경로 |
| plan | `.claude/plans/{slug}.md` + `TASKS.md` | build | TASKS.md 체크박스 |
| build | 구현된 코드 + `TASKS.md [x]` | review | `git diff` |
| review | CLEAN / CODE CHANGE REQUIRED | autopilot | 루프백 판단 |
| research | `.omc/research/{slug}.md` | think/plan | 참고 자료 |
| design | 구현된 디자인 코드 | review | `git diff` |

## HUD (Statusline)

Claude Code 하단에 프로젝트명과 context 사용률 표시.

- `ina hud on` — HUD 활성화
- `ina hud off` — HUD 비활성화
- `ina hud compact` — 1줄 미니멀 모드
- `ina hud full` — 1줄 풀 모드 (기본, `│` 구분자)
- context 80% 이상 시 `/compact` 안내 표시

## MCP Tools

- `ina_report_progress` — 진행 상황 보고
- `ina_mark_blocked` — 차단 상태 보고
- `ina_check_agents` — 에이전트 상태 조회

## Build & Test

```bash
go build -o ina .
go build -o ina-mcp ./mcp/
go test ./... -count=1 -race
```

## Conventions

- 커밋 전 반드시 사용자 허락
- Python: `uv run` 사용
- 한국어 출력 우선
- co-author 적지 않음
