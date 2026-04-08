---
name: plan
description: 합의 기반 플래닝 + 다관점 검증 + TDD 태스크 분해
argument-hint: [spec-file-path | description]
---

# Plan

스펙을 합의 기반으로 검증하고, 다관점 서브에이전트 검증을 거쳐, TDD 태스크 리스트로 분해한다.

## 언제 사용

- 스펙이 확정되고 구현 전 계획이 필요할 때
- 복잡한 기능을 체계적으로 분해하고 싶을 때
- `/ina:think` 완료 후

## 사용하지 말 것

- 아이디어가 아직 모호할 때 → `/ina:think`
- 단순한 일감이라 플래닝 불필요 → 직접 구현 또는 `/ina:build`
- 이미 TASKS.md가 잘 정의되어 있을 때 → `/ina:build`

## 인자

- 스펙 파일 경로 (예: `.ina/specs/20260405-1000-think-auth.md`)
- 또는 자연어 설명
- `--quick`: Phase 1-2 스킵, 바로 태스크 분해만. 사용 시 로그에 "quick 모드: 합의/검증 생략" 기록

## ina 연동

ina 데몬에 의해 실행된 경우:

- Phase 1: `ina_report_progress(in_progress="아키텍처 합의 (시도 N/5)", remaining="다관점 검증, 태스크 분해")`
- Phase 2: `ina_report_progress(in_progress="다관점 검증", completed="아키텍처 합의")`
- Phase 3: `ina_report_progress(in_progress="TDD 태스크 분해", completed="아키텍처 합의, 다관점 검증")`
- 합의 5회 실패: `ina_mark_blocked(reason="아키텍처 합의 5회 도달 — 핵심 분기점: {issue}")`

## 전체 흐름

```
Phase 0: 입력 검증
Phase 1: 아키텍처 합의 (Planner → Architect → Critic, 최대 5회)
Phase 2: 다관점 검증 (Architect + Critic + CEO 병렬)
Phase 3: TDD 태스크 분해
```

## Phase 0: 입력 검증

스펙 파일을 검증:
- 파일 존재 확인
- Goal, Constraints, Acceptance Criteria 섹션이 비어있지 않은지 확인
- 검증 실패 시: "스펙이 불완전합니다. `/ina:think`로 먼저 스펙을 작성하세요." 안내

자연어 입력의 경우: 검증 없이 Phase 1 진행.

## Phase 1: 아키텍처 합의

### Step 1. Planner (메인 Claude)
- 스펙을 읽고 초기 플랜 작성
- **필수 포함**:
  - 파일 구조
  - 핵심 결정
  - **ASCII 다이어그램** (데이터 흐름, 상태 머신 등)
  - **Blast Radius**: 영향받는 파일/모듈/API 목록 + 변경 범위 평가
- `.claude/plans/{slug}.md`에 저장

### Step 2. Architect (역할 전환)
- 플랜에 대해 아키텍처 리뷰 수행
- 반드시 포함: steelman 반론 1개 + 실질적 트레이드오프 1개 이상
- 의식적으로 "나는 지금 Architect 역할이다"라고 선언 후 리뷰

> Agent 도구 사용 가능 시: 별도 Task로 분리하여 관점 편향 방지

### Step 3. Critic (역할 전환)
- 플랜과 Architect 리뷰를 읽고 품질 평가
- 판정: APPROVE / ITERATE / REJECT
- ITERATE/REJECT 시 구체적 수정 요청 포함

### Step 4. 반복 (최대 5회)
- APPROVE가 아니면: 피드백 반영 → Step 1 재작성 → Step 2 → Step 3
- 5회 도달 시: 최선 버전을 사용자에게 제시

## Phase 2: 다관점 검증

합의된 플랜에 대해 3개 Agent를 **하나의 메시지에서 병렬로** 실행:

### Agent 1 — Architect (플랜 아키텍처 리뷰)

```
프롬프트: "다음 구현 플랜을 아키텍처 관점에서 리뷰하세요.
플랜: {plan_content}
스펙: {spec_content}
프로젝트: {CLAUDE.md 발췌}

필수: 기술적 타당성, 실패 모드, 확장성 리스크
판정: APPROVED / ITERATE"
```

### Agent 2 — Critic (태스크 분해 적절성)

```
프롬프트: "다음 구현 플랜을 프로세스 관점에서 리뷰하세요.
플랜: {plan_content}

필수: 
- 각 결정의 수락 기준이 테스트 가능한가?
- TDD로 분해 가능한 구조인가?
- 빠진 엣지 케이스, 에러 핸들링
판정: APPROVED / ITERATE"
```

### Agent 3 — CEO (스코프 판단)

```
프롬프트: "다음 구현 플랜을 전략 관점에서 리뷰하세요.
플랜: {plan_content}
스펙: {spec_content}

필수: 스코프 적절성, 과설계 여부, MVP 기준 충족
판정: APPROVED / ITERATE"
```

**결과 처리:**
- 3명 모두 APPROVED → Phase 3
- ITERATE → 피드백 반영하여 플랜 수정 후 Phase 1 Step 1로 (합의 루프 횟수에 합산)
- 전체 5회 도달 시: 사용자에게 판단 위임

## Phase 3: TDD 태스크 분해

합의+검증된 플랜을 2-5분 단위 태스크로 분해:

- 각 태스크는 **하나의 액션**:
  1. 실패 테스트 작성
  2. 테스트 실행하여 실패 확인
  3. 최소 구현
  4. 테스트 통과 확인
  5. 커밋

- 실제 코드 블록 포함 (pseudocode 금지)
- `TASKS.md`에 체크박스 형태로 저장

### 플랜 크기 가이드

플랜이 너무 크면 에이전트가 한 세션에서 완료할 수 없고, 리뷰와 롤백이 어려워진다.

| 기준 | 적정 | 초과 시 |
|------|------|---------|
| 태스크 수 | 3~7개 | 여러 플랜으로 분리 |
| 변경 파일 수 | ≤10개 | guard blast radius와 연동 |
| 한 세션 완료 | 필수 | 분리 필수 |

**예외**: 원자적 변경이 필요한 경우(DB 마이그레이션, 스키마 변경)는 크더라도 쪼개지 않는다.

태스크가 7개를 초과하면 사용자에게 "플랜을 나눌까요?"라고 확인한다.

### 플랜 분리 시 TASKS.md 구조

여러 플랜으로 나뉘면 TASKS.md에 섹션으로 구분한다. 에이전트가 TASKS.md만 읽으면 전체 상황과 현재 위치를 파악할 수 있어야 한다.

```markdown
## Plan 1: DB 스키마 ✅
- [x] 마이그레이션 파일 작성
- [x] 모델 정의

## Plan 2: API ← 현재
- [ ] 엔드포인트 구현
- [ ] 인증 미들웨어

## Plan 3: UI
- [ ] 로그인 폼
- [ ] 에러 처리
```

- 완료된 섹션에 ✅ 표시
- 현재 진행 중인 섹션에 `← 현재` 표시
- build는 현재 활성 섹션만 처리

## 입출력

- **입력**: `.ina/specs/{YYYYMMDD-HHMM}-think-{slug}.md` 또는 자연어
- **출력**: `.claude/plans/{slug}.md` + `TASKS.md` 갱신
