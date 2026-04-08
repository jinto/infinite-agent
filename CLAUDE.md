# ina — Infinite Angel

Coding angels that never stop. 에이전트 감시, 재시작, 오케스트레이션 데몬 + 12개 스킬 시스템.

## Skills

- `/ina:autopilot` — 파이프라인 오케스트레이터 (think → plan → build)
- `/ina:explore` — "만들까 말까?" 탐색 — 시장조사 인라인 + GO/NO-GO/PIVOT 판정
- `/ina:think` — 소크라틱 인터뷰로 기술 아이디어 → 스펙
- `/ina:rethink` — 코드베이스 종합검진 + "처음부터 다시 만든다면?" + 수정 계획
- `/ina:plan` — 합의 기반 플래닝 + TDD 태스크 분해
- `/ina:build` — 구현 → 리뷰 → 커밋 한방 실행기 (3-lane 리뷰 내장)
- `/ina:review` — 병렬 3-lane 코드 리뷰 (단독 실행용)
- `/ina:research` — 다각도 분해 + 병렬 검색
- `/ina:design` — 프레임워크 감지 + 구현 + 시각적 검증
- `/ina:test` — 독립 테스트 실행 + 실패 분석 + root cause 수정
- `/ina:ship` — PR 생성 (변경 요약 + 테스트 확인)
- `/ina:guard` — 무인 실행 안전장치 (위험 명령 차단, blast radius, 리뷰 게이트)

### Pipeline

```
autopilot: think → plan → build
                           │
                           ├─ Phase 1: 구현 (RED → GREEN per task)
                           ├─ Phase 2: 리뷰 (3-lane + fix-first, 루프백 최대 3회)
                           └─ Phase 3: 커밋
```

상태 파일: `.state/pipeline.json` — 크래시 복구용

## Skill I/O Convention

| From | Output Path | To | Input |
|------|------------|-----|-------|
| explore | `.ina/specs/{YYYYMMDD-HHMM}-explore-{slug}.md` | think (GO 시) | 판정 문서 |
| think | `.ina/specs/{YYYYMMDD-HHMM}-think-{slug}.md` | plan | 스펙 파일 경로 |
| rethink | `.ina/specs/{YYYYMMDD-HHMM}-rethink-{slug}.md` | plan | 수정 계획 |
| plan | `.claude/plans/{slug}.md` + `TASKS.md` | build | TASKS.md 체크박스 |
| build | 커밋된 코드 (구현→리뷰→커밋 내장) | autopilot | 완료 신호 |
| review | CLEAN / CODE CHANGE REQUIRED | build | 단독 실행 시 |
| research | `docs/{YYYYMMDD-HHMM}-research-{slug}.md` | think/plan | 참고 자료 |
| design | 구현된 디자인 코드 | build | Phase 2에서 리뷰 |

## HUD (Statusline)

Claude Code 하단에 프로젝트명과 context 사용률 1줄 표시: `project │ ████░░░░ 9%`

- `ina hud on` — HUD 활성화
- `ina hud off` — HUD 비활성화
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

# Tier 2: E2E 스킬 라우팅 (유료)
INFA_E2E=1 go test ./test/ -run TestSkillRouting -v

# Tier 3: LLM-Judge Eval (유료, pre-push hook으로도 실행)
INFA_EVAL=1 go test ./test/ -run TestSkillEval -v -timeout 600s
```

## Versioning

**git tag와 `.claude-plugin/plugin.json`의 version은 반드시 동일해야 한다.**

- 버전을 올릴 때 두 곳을 **항상 함께** 수정할 것
- 사용자가 태그 생성/푸시를 요청하면, `.claude-plugin/plugin.json` version도 함께 변경할지 반드시 확인할 것
- pre-push hook이 불일치 시 푸시를 차단하지만, 커밋 단계에서부터 맞춰야 함
- 태그 형식: `v1.4.0`, plugin.json 형식: `"1.4.0"` (v 접두사 없음)

## Conventions

- 커밋 전 반드시 사용자 허락
- Python: `uv run` 사용
- 한국어 출력 우선
- co-author 적지 않음
