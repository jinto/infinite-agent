---
name: review
description: 병렬 3-lane 코드 리뷰 + fix-first 자동 수정
argument-hint: [file-or-diff-target]
---

# Review

커밋 전 최종 코드 리뷰. 스펙 준수 → 병렬 3-lane 리뷰 → findings 집계 → fix-first → 판정.

## 언제 사용

- 커밋 전 최종 코드 리뷰
- PR 생성 전 셀프 리뷰
- autopilot 파이프라인의 review 단계로 호출될 때

## 사용하지 말 것

- 플랜 리뷰 → `/ina:plan`
- 아키텍처 리뷰 → 별도 architect agent 사용

## ina 연동

ina 데몬에 의해 실행된 경우:

- 각 Stage 진입 시 `ina_report_progress` 호출
- 내부 3회 재리뷰 후에도 ISSUE 남으면: `ina_mark_blocked(reason="리뷰 이슈 미해결: {issues}")`
- 데몬 MCP 호출 실패 시 무시하고 계속 진행

## Findings 포맷

모든 리뷰 레인은 아래 구조로 findings를 반환한다. 이 포맷은 fix-first 및 루프백에서 공통으로 사용된다.

```
FINDING: {severity} | {confidence} | {file}:{line_start}-{line_end}
{title}
{body — 무엇이 잘못되었고, 왜 위험하고, 어떻게 고쳐야 하는지}
```

- **severity**: `critical`, `high`, `medium`, `low`
- **confidence**: `0.0`~`1.0` (0.7 미만은 최종 집계에서 제외)
- **필터링 기준**: confidence ≥ 0.7 AND severity ∈ {critical, high, medium}만 fix-first 대상

## 흐름

### >>> Stage 0: 변경 감지

`git diff HEAD`와 `git diff --cached`를 실행한다. 둘 다 비어있으면 리뷰할 변경이 없다.

- 변경 없음 → **CLEAN — no changes to review.** 출력 후 즉시 종료
- 변경 있음 → Stage 1로 진행

### >>> Stage 1: 스펙 준수 확인

> `ina_report_progress(in_progress="스펙 준수 확인", remaining="병렬 리뷰, findings 집계, fix-first, 판정")`

- `TASKS.md` 또는 `.claude/plans/*.md`가 있으면 읽는다
- 현재 변경사항(`git diff`)이 스펙/태스크와 일치하는지 확인
- 누락된 요구사항이 있으면 목록으로 제시

### >>> Stage 2: 병렬 3-Lane 리뷰

> `ina_report_progress(in_progress="병렬 3-lane 리뷰", completed="스펙 확인")`

**3개 Agent를 동시에 실행한다.** 각 Agent는 독립적으로 `git diff`를 분석하고, 위의 Findings 포맷으로 결과를 반환한다.

#### Lane A: Adversarial Review (Codex CLI)

Codex CLI로 적대적 리뷰 실행:

```bash
codex exec -C . --full-auto -s read-only -c model_reasoning_effort="high" \
  "You are performing an adversarial code review.
  Your job is to break confidence in the change, not to validate it.
  Default to skepticism.

  Review git diff and git diff --cached.

  Attack surface priorities:
  - auth, permissions, tenant isolation, trust boundaries
  - data loss, corruption, irreversible state changes
  - race conditions, ordering assumptions, stale state
  - rollback safety, retries, partial failure, idempotency gaps
  - version skew, schema drift, migration hazards

  Finding bar: only material findings. No style feedback or speculative concerns.
  Each finding must answer: what can go wrong, why vulnerable, likely impact, concrete fix.

  Output format per finding:
  FINDING: {severity} | {confidence 0-1} | {file}:{line_start}-{line_end}
  {title}
  {body}

  If no material issues: output CLEAN.
  한국어로 응답."
```

**Codex CLI 실패 시 fallback**: Claude Agent (subagent)를 대신 실행하여 동일한 adversarial 관점으로 `git diff`를 리뷰. 결과에 `[degraded: self-review]` 태그를 붙인다.

#### Lane B: Security Review (Agent)

Claude Agent를 실행하여 보안 중심 리뷰:

```
보안 리뷰어로서 git diff를 분석하라.

검증 항목 (OWASP Top 10 기반):
- Injection (SQL, NoSQL, Command, XSS)
- 인증/인가 결함
- 민감 데이터 노출 (하드코딩된 키, 토큰, 시크릿)
- 입력 검증 누락 / 출력 인코딩 누락
- CORS / CSRF 설정
- 의존성 취약점 (알려진 CVE)

Severity × Exploitability × Blast Radius로 우선순위 결정.

Output format per finding:
FINDING: {severity} | {confidence 0-1} | {file}:{line_start}-{line_end}
{title}
{body — 취약점 설명 + 안전한 코드 예시}

이슈 없으면: CLEAN.
한국어로 응답.
```

#### Lane C: Simplify Review (Agent)

Claude Agent를 실행하여 코드 간결화 리뷰:

```
코드 간결화 전문가로서 git diff의 변경된 코드만 분석하라.

검증 항목:
- 불필요한 복잡성, 중첩 (nested ternary 등)
- 중복 코드, 불필요한 추상화
- 명확하지 않은 변수/함수명
- 사용하지 않는 import, 변수
- 프로젝트 CLAUDE.md 코딩 컨벤션 위반
- 가독성을 해치는 과도한 압축

원칙: 기능 변경 없이 clarity만 개선. 간결함보다 명확함 우선.

Output format per finding:
FINDING: {severity} | {confidence 0-1} | {file}:{line_start}-{line_end}
{title}
{body — 현재 문제 + 개선 제안}

이슈 없으면: CLEAN.
한국어로 응답.
```

### >>> Stage 3: Findings 집계

> `ina_report_progress(in_progress="findings 집계", completed="스펙 확인, 병렬 리뷰")`

3개 레인의 결과를 합친다:

1. **필터링**: confidence < 0.7 또는 severity == `low` → 제외
2. **중복 제거**: 동일 파일:라인 범위를 가리키는 findings → severity 높은 쪽 유지
3. **정렬**: critical → high → medium 순

집계 결과를 사용자에게 테이블로 보고:

```
## Review Findings

| # | Sev | Lane | File:Line | Title | Conf |
|---|-----|------|-----------|-------|------|
| 1 | critical | adversarial | auth.go:42-50 | 토큰 검증 누락 | 0.9 |
| 2 | high | security | db.go:15-20 | SQL injection | 0.8 |
| 3 | medium | simplify | util.go:30-35 | 중첩 삼항 연산자 | 0.7 |

Total: 3 findings (1 critical, 1 high, 1 medium)
Lanes: adversarial ✓ | security ✓ | simplify ✓
```

findings가 0개면 → Stage 5로 직행 (CLEAN).

### >>> Stage 4: Fix-First 자동 수정

> `ina_report_progress(in_progress="fix-first 자동 수정", completed="스펙 확인, 병렬 리뷰, findings 집계")`

집계된 findings를 분류하여 처리:

**MECHANICAL FIX (자동 적용)** — simplify 레인의 findings 중심:
- 포맷팅, 임포트 정리, 사용하지 않는 변수 제거
- 타입 힌트 누락 보완
- 명백한 오타 수정
- 중첩 삼항 → if/else 변환
- 명확하지 않은 변수명 개선

**CODE CHANGE REQUIRED (코드 변경 필요)** — adversarial/security 레인 중심:
- 로직 변경이 필요한 버그
- 보안 취약점 (injection, 인증 결함 등)
- 아키텍처 관련 이슈
- 성능 관련 트레이드오프

자동 수정 후 어떤 findings를 수정했고, 어떤 것이 남았는지 보고.

### >>> Stage 5: 재리뷰 (최대 3회)

- Stage 4에서 수정이 있었으면 Stage 2로 돌아가 재리뷰
- CLEAN 판정 시 완료
- 3회 반복 후에도 ISSUE 남으면: `ina_mark_blocked` + 남은 이슈 요약

### >>> Stage 6: 최종 판정

리뷰 결과를 3가지 중 하나로 판정:

| 판정 | 의미 | autopilot 동작 |
|------|------|---------------|
| **CLEAN** | 이슈 없음 | → 리뷰 게이트 해제 → 문서 업데이트 확인 → commit |
| **MECHANICAL FIX** | 기계적 수정 완료, 추가 이슈 없음 | → 리뷰 게이트 해제 → 문서 업데이트 확인 → commit |
| **CODE CHANGE REQUIRED** | 코드 변경 필요 | → build 단계로 루프백 |

**리뷰 게이트 해제:** CLEAN 또는 MECHANICAL FIX 판정 시 `.state/review-gate.md`를 삭제한다. 이 파일이 삭제되어야 커밋이 가능하다 (guard 규칙 5 참조).

## autopilot 루프백 프로토콜

autopilot 파이프라인에서 호출된 경우:

1. **CODE CHANGE REQUIRED** 판정 시:
   - 이슈 목록을 `.state/review-issues.md`에 구조화된 findings 포맷으로 기록:
     ```markdown
     # Review Issues (Loop N)
     | # | Sev | Lane | File:Line | Title | Conf |
     |---|-----|------|-----------|-------|------|
     | 1 | critical | adversarial | auth.go:42-50 | 토큰 검증 누락 | 0.9 |
     ```
   - autopilot에 "CODE CHANGE REQUIRED" 반환
   - autopilot이 `review_loops`를 증가시키고 build 재실행

2. **루프백 제한:**
   - autopilot 레벨에서 최대 3회 (pipeline.json의 `review_loops`)
   - 3회 초과 시: `ina_mark_blocked(reason="review 3회 루프백 초과 — 미해결 이슈: {issues}")`
   - 누적 이슈 목록을 사용자에게 보고하고 **커밋하지 않음**

3. **build 재실행 시:**
   - build가 `.state/review-issues.md`를 읽어서 해당 이슈만 수정
   - 전체 재구현이 아닌 targeted fix

## 입출력

- **입력**: uncommitted changes (`git diff`)
- **출력**: 판정 (CLEAN / MECHANICAL FIX / CODE CHANGE REQUIRED) + findings 테이블 + 자동 수정된 코드
