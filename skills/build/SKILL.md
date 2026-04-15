---
name: build
description: 태스크 구현 → 리뷰 → 커밋 (plan 이후 한방 실행기)
argument-hint: [--no-review]
---

# Build

TASKS.md의 미완료 태스크를 구현하고, 리뷰하고, 커밋까지 한다. plan 이후 실행만 남았을 때 이 스킬 하나로 끝낸다.

## 언제 사용

- TASKS.md에 할 일이 정의되어 있고 구현 → 리뷰 → 커밋까지 진행할 때
- autopilot 파이프라인의 실행 단계로 호출될 때

## 사용하지 말 것

- 아이디어가 모호할 때 → `/ina:think`
- 플랜이 없을 때 → `/ina:plan`
- 처음부터 끝까지 자동 → `/ina:autopilot`

## 인자

- (없음): 인터랙티브 모드 (단계마다 확인)
- `--no-review`: 구현만 하고 끝 (리뷰/커밋 생략, review-gate 생성)
- `--non-interactive`: 자동 진행 (문제 발생 시에만 멈춤)

## ina 연동

- 태스크 시작: `ina_report_progress(in_progress="BUILD: {태스크}", remaining="{남은 수}")`
- 리뷰 시작: `ina_report_progress(in_progress="REVIEW: 병렬 리뷰 ({tier})", completed="구현")`
- 막히면: `ina_mark_blocked(reason="BUILD: {이유}")`
- 데몬 MCP 호출 실패 시 무시하고 계속 진행

## 전체 흐름

```
Phase 1: 구현 (태스크 식별 → 위임 → 실행 → 검증 → 체크박스)
Phase 2: 리뷰 (티어별 1-3 lane → findings 집계 → fix-first)
Phase 3: 커밋 (문서 확인 → 사용자 허락 → commit)

--no-review 시: Phase 1만 실행 후 review-gate 생성
```

---

## Phase 1: 구현

### >>> Stage 1: 태스크 식별

> `ina_report_progress(in_progress="태스크 파악", remaining="{전체 미완료 수}")`

- `TASKS.md` 읽고 미완료(`- [ ]`) 항목 식별
- **섹션이 여러 개면 `← 현재` 표시된 섹션만 처리** (다른 섹션은 무시)
- 현재 섹션 완료 시: ✅ 표시하고, 다음 섹션에 `← 현재` 이동
- 관련 플랜 파일(`.claude/plans/*.md`) 읽어서 컨텍스트 확보
- 루프백 재진입 시: `.state/review-issues.md` 존재하면 해당 이슈 우선 처리
- TASKS.md 파싱 실패 시: 사용자에게 경로 확인 요청

### >>> Stage 2: 실행 위임 판단

미완료 태스크들을 분석하여 실행 방식을 결정한다:

```
1. 독립성 판단 (LLM)
   - 공유 파일이 있는가? → 순차 실행
   - 다른 디렉토리/모듈인가? → 병렬 후보

2. 태스크 수에 따른 방식 선택
   - 1개 또는 의존성 있는 태스크 → 직접 실행
   - 2-3개 독립 태스크 → Agent 병렬 (subagent)
   - 4개+ 독립 태스크 → Team

3. 도메인 라우팅
   - UI/프론트엔드 태스크 → /ina:design 스킬 참조
   - 외부 리뷰/분석 필요 → codex 위임 (Agent)
```

### >>> Stage 3: 구현 (RED → GREEN)

> `ina_report_progress(in_progress="구현: {태스크}", context="위임 방식: {method}")`

`/ina:plan`이 TDD 구조로 분해한 태스크를 그대로 따른다:

```
1. RED   — 의도를 검증하는 실패 테스트 작성 → 실행하여 실패 확인
2. GREEN — 테스트 통과할 최소 코드 구현
3. 다음 태스크로 (리팩터는 리뷰 Phase에서)
```

- 버그 수정 태스크: 반드시 재현 테스트 먼저 (Prove-It Pattern)
- 설정/문서 등 행위 변경 없는 태스크: 테스트 생략 가능
- `/ina:guard` 규칙 준수 (위험 명령 차단, blast radius 체크)

#### 위임 시에도 TDD 유지

서브에이전트/Team에 위임할 때 **RED→GREEN 원칙을 프롬프트에 포함**:

```
Agent: "TASKS.md의 태스크 A를 구현.
  1) 실패 테스트 먼저 작성  2) 최소 구현으로 통과
  플랜: {plan_excerpt}. 관련 파일: {files}"
```

- 2-3개 독립 태스크 → Agent 병렬 (하나의 메시지에서)
- 4개+ 독립 태스크 → Team
- **컨텍스트 격리 원칙:** 각 에이전트에 플랜 발췌 + 해당 태스크 + 관련 파일만 전달

### >>> Stage 4: 전체 검증

> `ina_report_progress(in_progress="검증: 전체 테스트 스위트")`

1. 전체 테스트 스위트 실행 (CLAUDE.md의 테스트 명령 우선)
2. 실패 시: 분석 → 수정 → 재실행 (최대 3회)
3. 3회 내 미해결 시: `ina_mark_blocked` + 사용자 보고
4. guard 규칙 확인: 변경 파일 수, 테스트 회귀
5. **스모크 테스트** — 프로젝트 타입을 감지하여 실제 실행 확인:
   - **CLI** → 기본 명령 실행 (예: `ina status`, `--help`)
   - **웹** → 개발 서버 띄우고 브라우저로 주요 페이지 확인
   - **앱** → 실행하여 기본 동작 확인
   - **라이브러리** → 생략
   - 감지 불가 시: 사용자에게 확인 요청

### >>> Stage 5: 체크박스 업데이트 + 루프

- TASKS.md에서 완료된 항목을 `- [x]`로 변경
- 다음 미완료 태스크 있으면 → Stage 1로
- 전부 완료 → **Phase 경계 체크포인트** 후 Phase 2로 진행
- 태스크 10개 초과 처리 시: 사용자에게 "계속 진행할까요?" 확인
- `--no-review` 시: `.state/review-gate.md` 생성하고 끝

### >>> Phase 경계 체크포인트 (Phase 1 → Phase 2)

> **Phase 전환 전에 반드시 실행한다.**

1. `.state/progress.md`의 `## Context for Restart` 섹션을 **충실히** 작성:
   - 변경한 파일 목록 (경로 + 변경 요약)
   - 통과한 테스트 결과
   - 남은 작업 (Phase 2: 리뷰, Phase 3: 커밋)
   - 핵심 설계 결정 사항
2. `pipeline.json` 업데이트: `stage="build"`, `sub_phase="review"`
3. **context 확인**: HUD 또는 체감으로 context가 높다고 판단되면 (긴 대화, 많은 파일 수정):
   - 위 체크포인트가 완전한지 재확인
   - `exit(42)` — daemon이 fresh session으로 Phase 2를 시작함
   - 확신이 없으면 그냥 Phase 2로 계속 진행 (exit하지 않음)

---

## Phase 2: 리뷰

> `ina_report_progress(in_progress="REVIEW: 병렬 리뷰 ({tier})", completed="구현")`

### >>> Stage 6: 스펙 준수 확인

- `TASKS.md` 또는 `.claude/plans/*.md` 대비 변경사항 검증
- 누락된 요구사항 목록 제시

### >>> Stage 6.5: Lane 티어 결정

변경 파일 수로 리뷰 강도를 결정한다:

| 변경 파일 수 | 티어 | 실행 Lane |
|-------------|------|----------|
| ≤ 3 | **lite** | Simplify만 |
| 4-10 | **standard** | Security + Simplify |
| 11+ | **full** | Adversarial + Security + Simplify |

티어 로그: "Review tier: {tier} ({total_files} files) — {lane_names}"

### >>> Stage 7: 병렬 리뷰 (티어에 따라 1-3 Lane)

Stage 6.5에서 결정된 티어에 따라 해당 Lane만 실행:

- **Lane A: Adversarial** — Codex CLI로 적대적 리뷰 (실패 시 Claude fallback) — **full 티어만**
- **Lane B: Security** — OWASP Top 10 보안 검증 — **standard + full 티어**
- **Lane C: Simplify** — 코드 간결화/가독성 검증 — **모든 티어**

각 레인은 구조화된 Findings 포맷으로 반환:
```
FINDING: {severity} | {confidence 0-1} | {file}:{line_start}-{line_end}
{title}
{body}
```

### >>> Stage 8: Findings 집계

1. confidence < 0.7 또는 severity == `low` → 제외
2. 동일 파일:라인 → severity 높은 쪽 유지
3. critical → high → medium 순 정렬
4. 테이블로 사용자에게 보고

findings 0개면 → Phase 3으로 직행 (CLEAN).

### >>> Stage 9: Fix-First 자동 수정

- **MECHANICAL FIX**: simplify 레인 중심 — 자동 적용
- **CODE CHANGE REQUIRED**: adversarial/security 레인 — 코드 변경 필요

### >>> Stage 10: 재리뷰 루프 (최대 3회)

- 수정이 있었으면 Stage 7로 돌아가 재리뷰
- 재리뷰 시 티어는 변경하지 않음 — 초기 결정 유지
- CLEAN 시 **Phase 경계 체크포인트** 후 Phase 3으로
- 3회 후에도 ISSUE → `ina_mark_blocked` + 중단

### >>> Phase 경계 체크포인트 (Phase 2 → Phase 3)

> **Phase 전환 전에 반드시 실행한다.**

1. `.state/progress.md`의 `## Context for Restart` 섹션 업데이트:
   - 리뷰 결과 요약 (CLEAN / 수정 내역)
   - 남은 작업 (Phase 3: 문서 확인 + 커밋)
2. `pipeline.json` 업데이트: `sub_phase="commit"`
3. **context 확인**: 높으면 `exit(42)`, 아니면 계속 진행

---

## Phase 3: 커밋

> `ina_report_progress(in_progress="COMMIT", completed="구현, 리뷰")`

### >>> Stage 11: 문서 확인 + 커밋

1. 문서 업데이트 확인:
   - `CLAUDE.md` — 새 명령/스킬/규칙 반영 필요 여부
   - `README.md` / `README.ko.md` — 사용법 변경 여부
   - `TASKS.md` — 완료 체크박스 처리
   - 불일치 발견 시 수정
2. `.state/review-gate.md` 삭제 (리뷰 게이트 해제)
3. **커밋 전 반드시 사용자 허락**
4. 커밋 완료
5. `ina_report_progress(in_progress="완료", completed="구현, 리뷰, 커밋")`

## 오동작 방지

- 각 Stage 진입 시 `>>> Build: {name}` 출력
- guard 규칙 자동 적용 (위험 명령, blast radius, 비밀 정보)
- 태스크 10개 초과 시 사용자 확인
- 테스트 3회 실패 시 즉시 blocked
- 리뷰 루프 3회 초과 시 즉시 blocked

## 입출력

- **입력**: `TASKS.md` (미완료 항목) + `.claude/plans/*.md`
- **출력**: 커밋된 코드 + 완료 처리된 `TASKS.md`
- `--no-review` 시: 구현된 코드 + `.state/review-gate.md`
