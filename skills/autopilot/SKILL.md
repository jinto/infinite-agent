---
name: autopilot
description: 파이프라인 오케스트레이터 — think → plan → build (리뷰+커밋 포함)
argument-hint: [task description]
---

# Autopilot

아이디어에서 커밋까지 전체 파이프라인을 자동 실행한다. 각 스킬을 순차 호출하고, 상태를 `.state/pipeline.json`에 기록하여 크래시 복구를 보장한다.

## 언제 사용

- 아이디어부터 커밋까지 전자동으로 진행하고 싶을 때
- `ina launch --task "..."` 로 무인 실행할 때

## 사용하지 말 것

- plan 이후 실행만 남았을 때 → `/ina:build` (리뷰+커밋 포함)
- 특정 단계만 실행 → 해당 스킬 직접 호출 (`/ina:think`, `/ina:plan` 등)

## 인자

- 자연어 태스크 설명
- 스펙이 이미 있으면: 스펙 파일 경로

## 파이프라인 상태

`.state/pipeline.json`에 현재 진행 상태를 기록한다:

```json
{
  "stage": "build",
  "skill": "ina:build",
  "task": "인증 시스템 추가",
  "spec_path": ".ina/specs/20260405-1000-think-auth.md",
  "plan_path": ".claude/plans/auth.md",
  "started_at": "2026-04-05T10:00:00Z",
  "updated_at": "2026-04-05T10:30:00Z"
}
```

**매 Stage 전환 시** 이 파일을 업데이트한다. 크래시 후 재시작 시 이 파일을 읽어서 해당 Stage부터 재개한다.

## ina 연동

- 각 Stage 진입: `ina_report_progress(in_progress="autopilot: {stage}")`
- 막히면: `ina_mark_blocked(reason="autopilot: {stage}에서 {reason}")`
- 데몬 MCP 호출 실패 시 무시하고 계속 진행

## 전체 흐름

```
autopilot 시작
│
├─ .state/pipeline.json 존재? → 재개 모드 (해당 stage부터)
└─ 없으면 → 새 파이프라인 생성
│
├─ Stage 1: THINK ──────────────────────────────────┐
│   /ina:think 호출                                 │
│   검증: .ina/specs/{type}-{slug}.md 존재            │
│   pipeline.json → stage="plan"                     │
│                                                    │
├─ Stage 2: PLAN ───────────────────────────────────┤
│   /ina:plan 호출 (스펙 파일 경로 전달)              │
│   검증: .claude/plans/{slug}.md + TASKS.md 존재     │
│   pipeline.json → stage="build"                    │
│                                                    │
├─ Stage 3: BUILD ──────────────────────────────────┤
│   /ina:build 호출                                 │
│   build가 내부적으로 구현 → 리뷰 → 커밋까지 처리     │
│   pipeline.json 삭제                               │
└────────────────────────────────────────────────────┘
```

## Stage 상세

### >>> Stage 1: THINK

> `ina_report_progress(in_progress="autopilot: think")`

1. `/ina:think` 스킬 호출 (태스크 설명 전달)
2. 완료 후 산출물 검증:
   - `.ina/specs/{YYYYMMDD-HHMM}-think-{slug}.md` 존재
   - Goal, Constraints, Acceptance Criteria 섹션이 비어있지 않음
3. 검증 실패 시: 사용자에게 스펙 보완 요청
4. **Stage 경계 체크포인트:**
   - `pipeline.json` 업데이트: `stage="plan"`, `spec_path` 기록
   - `.state/progress.md`의 `## Context for Restart`에 스펙 파일 경로 기록
   - context가 높으면 `exit(42)` — daemon이 plan stage부터 fresh session으로 재개

**스킵 조건:**
- 스펙 파일이 이미 인자로 전달된 경우 → Stage 2로 직행
- 사용자가 명확한 요구사항을 직접 제공한 경우 → Stage 2로 직행

### >>> Stage 2: PLAN

> `ina_report_progress(in_progress="autopilot: plan", completed="think")`

1. `/ina:plan` 스킬 호출 (스펙 파일 경로 전달)
   - plan은 pipeline.json을 감지하여 Phase 2(다관점 검증)를 자동 스킵 — think에서 검증 완료
2. 완료 후 산출물 검증:
   - `.claude/plans/{slug}.md` 존재
   - `TASKS.md`에 `- [ ]` 항목이 최소 1개
3. **Stage 경계 체크포인트:**
   - `pipeline.json` 업데이트: `stage="build"`, `plan_path` 기록
   - `.state/progress.md`의 `## Context for Restart`에 플랜 파일 경로 + 태스크 수 기록
   - context가 높으면 `exit(42)` — daemon이 build stage부터 fresh session으로 재개

### >>> Stage 3: BUILD

> `ina_report_progress(in_progress="autopilot: build", completed="think, plan")`

1. `/ina:build` 스킬 호출
2. build가 내부적으로 3 Phase를 실행:
   - Phase 1: 구현 (태스크 순차/병렬 처리)
   - Phase 2: 리뷰 (티어별 1-3 lane + fix-first + 루프백)
   - Phase 3: 커밋 (문서 확인 + 사용자 허락)
3. build 내부에도 Phase 경계 체크포인트가 있음 (build SKILL.md 참조)
4. 완료 후 `pipeline.json` 삭제
5. `ina_report_progress(in_progress="완료", completed="전체 파이프라인")`

## 크래시 복구

데몬이 에이전트를 재시작하면:

1. `.state/pipeline.json` 존재 확인
2. 존재하면 현재 `stage` 읽기
3. 해당 stage의 스킬부터 재개
4. 이전 stage의 산출물은 이미 파일로 존재하므로 재활용

## 오동작 방지

- 각 Stage 진입 시 `>>> Autopilot Stage: {name}` 출력
- Stage 전환 시 반드시 산출물 검증 (파일 존재 + 내용 유효성)
- 커밋 전 반드시 사용자 허락 (build Phase 3에서 처리)
- 데몬 MCP 호출 실패 시 무시하고 계속 진행

## 입출력

- **입력**: 자연어 태스크 설명 또는 스펙 파일 경로
- **출력**: 커밋된 코드 + 완료 처리된 TASKS.md
