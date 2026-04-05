---
name: autopilot
description: ina 파이프라인 오케스트레이터 — think, plan, build, review, commit 자동 실행
argument-hint: [task description]
---

# Autopilot

아이디어에서 커밋까지 전체 파이프라인을 자동 실행한다. 각 스킬을 순차 호출하고, 상태를 `.state/pipeline.json`에 기록하여 크래시 복구를 보장한다.

## 언제 사용

- 아이디어부터 커밋까지 전자동으로 진행하고 싶을 때
- `ina launch --task "..."` 로 무인 실행할 때

## 사용하지 말 것

- 특정 단계만 실행 → 해당 스킬 직접 호출 (`/ina:think`, `/ina:build` 등)
- 이미 TASKS.md가 잘 정의되어 있고 바로 구현 → `/ina:build`

## 인자

- 자연어 태스크 설명
- 스펙이 이미 있으면: 스펙 파일 경로

## 파이프라인 상태

`.state/pipeline.json`에 현재 진행 상태를 기록한다:

```json
{
  "stage": "execute",
  "skill": "ina:build",
  "task": "인증 시스템 추가",
  "spec_path": ".omc/specs/think-auth.md",
  "plan_path": ".claude/plans/auth.md",
  "tasks_total": 5,
  "tasks_done": 2,
  "review_loops": 0,
  "review_loop_max": 3,
  "started_at": "2026-04-05T10:00:00Z",
  "updated_at": "2026-04-05T10:30:00Z"
}
```

**매 Stage 전환 시** 이 파일을 업데이트한다. 크래시 후 재시작 시 이 파일을 읽어서 해당 Stage부터 재개한다.

## ina 연동

- 각 Stage 진입: `ina_report_progress(in_progress="autopilot: {stage}", context="pipeline.json 경로")`
- 막히면: `ina_mark_blocked(reason="autopilot: {stage}에서 {reason}")`
- 데몬이 MCP 호출 실패를 반환하면 무시하고 계속 진행 (데몬 다운 시에도 파이프라인 중단 안 함)

## 전체 흐름

```
autopilot 시작
│
├─ .state/pipeline.json 존재? → 재개 모드 (해당 stage부터)
└─ 없으면 → 새 파이프라인 생성
│
├─ Stage 1: THINK ──────────────────────────────────┐
│   /ina:think 호출                                 │
│   검증: .omc/specs/{type}-{slug}.md 존재            │
│         + Goal, Constraints, Criteria 섹션 비어있지 않음│
│   pipeline.json → stage="plan"                     │
│                                                    │
├─ Stage 2: PLAN ───────────────────────────────────┤
│   /ina:plan 호출 (스펙 파일 경로 전달)              │
│   검증: .claude/plans/{slug}.md 존재                │
│         + TASKS.md에 - [ ] 항목 최소 1개             │
│   pipeline.json → stage="execute"                  │
│                                                    │
├─ Stage 3: EXECUTE ────────────────────────────────┤
│   /ina:build 호출                                 │
│   검증: TASKS.md 전체 - [x]                         │
│         + 프로젝트 테스트 통과                        │
│   pipeline.json → stage="review"                   │
│                                                    │
├─ Stage 4: REVIEW ─────────────────────── loop ◄──┤
│   /ina:review 호출                                │
│   결과:                                            │
│   - CLEAN → stage="commit"                         │
│   - MECHANICAL FIX → review가 직접 수정 후 재검증    │
│   - CODE CHANGE REQUIRED:                          │
│     review_loops++                                 │
│     review_loops ≤ 3 → stage="execute" (루프백)     │
│     review_loops > 3 → ina_mark_blocked + 중단    │
│   이슈 목록: .state/review-issues.md에 기록          │
│                                                    │
├─ Stage 5: COMMIT ─────────────────────────────────┤
│   업데이트가 필요한 문서 수정                         │
│   커밋 (사용자 허락 후)                              │
│   pipeline.json 삭제                               │
└────────────────────────────────────────────────────┘
```

## Stage 상세

### >>> Stage 1: THINK

> `ina_report_progress(in_progress="autopilot: think")`

1. `/ina:think` 스킬 호출 (태스크 설명 전달)
2. 완료 후 산출물 검증:
   - `.omc/specs/think-{slug}.md` 존재
   - Goal, Constraints, Acceptance Criteria 섹션이 비어있지 않음
3. 검증 실패 시: 사용자에게 스펙 보완 요청
4. `pipeline.json` 업데이트: `stage="plan"`, `spec_path` 기록

**스킵 조건:**
- 스펙 파일이 이미 인자로 전달된 경우 → Stage 2로 직행
- 사용자가 명확한 요구사항을 직접 제공한 경우 → Stage 2로 직행

### >>> Stage 2: PLAN

> `ina_report_progress(in_progress="autopilot: plan", completed="think")`

1. `/ina:plan` 스킬 호출 (스펙 파일 경로 전달)
2. 완료 후 산출물 검증:
   - `.claude/plans/{slug}.md` 존재
   - `TASKS.md`에 `- [ ]` 항목이 최소 1개
3. `pipeline.json` 업데이트: `stage="execute"`, `plan_path` 기록, `tasks_total` 카운트

### >>> Stage 3: EXECUTE

> `ina_report_progress(in_progress="autopilot: execute ({tasks_done}/{tasks_total})", completed="think, plan")`

1. `/ina:build` 스킬 호출
2. build가 TASKS.md의 태스크를 순차/병렬로 처리
3. 완료 후 검증:
   - `TASKS.md` 전체 항목이 `- [x]`
   - 프로젝트 테스트 통과 (CLAUDE.md의 테스트 명령 또는 기본값)
4. `pipeline.json` 업데이트: `stage="review"`, `tasks_done` 갱신

**루프백 재진입 시:** `.state/review-issues.md`를 읽어서 해당 이슈만 수정

### >>> Stage 4: REVIEW

> `ina_report_progress(in_progress="autopilot: review (loop {review_loops}/3)", completed="think, plan, execute")`

1. `/ina:review` 스킬 호출
2. 결과 처리:
   - **CLEAN**: `pipeline.json` → `stage="commit"`
   - **MECHANICAL FIX**: review가 직접 수정 + 재검증 후 CLEAN이면 commit으로
   - **CODE CHANGE REQUIRED**:
     - 이슈를 `.state/review-issues.md`에 기록
     - `review_loops++`
     - `review_loops ≤ 3` → `stage="execute"` (build가 이슈 파일 참고하여 수정)
     - `review_loops > 3` → `ina_mark_blocked(reason="review 3회 루프백 초과")` + 사용자 보고

### >>> Stage 5: COMMIT

> `ina_report_progress(in_progress="autopilot: commit", completed="think, plan, execute, review")`

1. 문서 업데이트 확인 — 다음 파일을 읽고 변경사항과 일치하는지 확인:
   - `CLAUDE.md` — 새 명령/스킬/규칙 반영 필요 여부
   - `README.md` / `README.ko.md` — 사용법, 설치 방법 변경 여부
   - `TASKS.md` — 완료 체크박스 `[x]` 처리
   - 기타 `docs/` 문서 — 변경된 API/구조와 불일치 여부
   - 불일치 발견 시 수정
2. **커밋 전 반드시 사용자 허락**
3. 커밋 완료 후 `.state/pipeline.json` 삭제
4. `ina_report_progress(in_progress="완료", completed="전체 파이프라인")`

## 크래시 복구

데몬이 에이전트를 재시작하면:

1. `.state/pipeline.json` 존재 확인
2. 존재하면 현재 `stage` 읽기
3. 해당 stage의 스킬부터 재개
4. 이전 stage의 산출물은 이미 파일로 존재하므로 재활용

## 오동작 방지

- 각 Stage 진입 시 `>>> Autopilot Stage: {name}` 출력
- Stage 전환 시 반드시 산출물 검증 (파일 존재 + 내용 유효성)
- review 루프백 3회 초과 시 즉시 중단 + 사용자 보고
- 커밋 전 반드시 사용자 허락
- 데몬 MCP 호출 실패 시 무시하고 계속 진행

## 입출력

- **입력**: 자연어 태스크 설명 또는 스펙 파일 경로
- **출력**: 커밋된 코드 + 완료 처리된 TASKS.md
