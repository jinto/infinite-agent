---
name: review
description: ina 멀티모델 코드 리뷰 + fix-first 자동 수정 + 루프백
argument-hint: [file-or-diff-target]
---

# Review

커밋 전 최종 코드 리뷰. 스펙 준수 확인 → 외부 리뷰 → fix-first 자동 수정. autopilot 파이프라인에서 호출 시 루프백 프로토콜을 따른다.

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

## 흐름

### >>> Stage 1: 스펙 준수 확인

> `ina_report_progress(in_progress="스펙 준수 확인", remaining="외부 리뷰, fix-first, 판정")`

- `TASKS.md` 또는 `.claude/plans/*.md`가 있으면 읽는다
- 현재 변경사항(`git diff`)이 스펙/태스크와 일치하는지 확인
- 누락된 요구사항이 있으면 목록으로 제시

### >>> Stage 2: 외부 코드 리뷰

> `ina_report_progress(in_progress="외부 코드 리뷰", completed="스펙 확인")`

Codex CLI로 미커밋 변경사항 리뷰:

```bash
npx codex exec -C . --full-auto -s read-only -c model_reasoning_effort="xhigh" \
  "git diff와 git diff --cached를 리뷰해주세요.
  리뷰 기준: 코드 품질, 버그, 보안, LLM 출력 신뢰 경계 검증.
  문제가 있으면 'ISSUE:' 접두어로 나열. 없으면 'CLEAN'으로 시작.
  한국어로 응답."
```

**Codex CLI 실패 시 fallback**: Claude가 `git diff`를 직접 읽고 동일 기준으로 자체 리뷰. "Codex 대신 자체 리뷰 수행 (degraded mode)" 고지. **자체 리뷰는 yellow light** — 통과하지만 기록에 degraded로 남긴다.

### >>> Stage 3: Fix-First 자동 수정

> `ina_report_progress(in_progress="Fix-First 자동 수정", completed="스펙 확인, 외부 리뷰")`

발견된 이슈를 분류:

**MECHANICAL FIX (자동 적용)**:
- 포맷팅, 임포트 정리, 사용하지 않는 변수 제거
- 타입 힌트 누락 보완
- 명백한 오타 수정

**CODE CHANGE REQUIRED (코드 변경 필요)**:
- 로직 변경이 필요한 버그
- 아키텍처 관련 이슈
- 성능 관련 트레이드오프

자동 수정 후 결과를 사용자에게 보고.

### >>> Stage 4: 재리뷰 (최대 3회)

- Stage 3에서 수정이 있었으면 Stage 2로 돌아가 재리뷰
- CLEAN 판정 시 완료
- 3회 반복 후에도 ISSUE 남으면: `ina_mark_blocked` + 남은 이슈 요약

### >>> Stage 5: 최종 판정

리뷰 결과를 3가지 중 하나로 판정:

| 판정 | 의미 | autopilot 동작 |
|------|------|---------------|
| **CLEAN** | 이슈 없음 | → 문서 업데이트 확인 → commit |
| **MECHANICAL FIX** | 기계적 수정 완료, 추가 이슈 없음 | → 문서 업데이트 확인 → commit |
| **CODE CHANGE REQUIRED** | 코드 변경 필요 | → execute 단계로 루프백 |

## autopilot 루프백 프로토콜

autopilot 파이프라인에서 호출된 경우:

1. **CODE CHANGE REQUIRED** 판정 시:
   - 이슈 목록을 `.state/review-issues.md`에 기록:
     ```markdown
     # Review Issues (Loop N)
     - [ ] ISSUE: {설명} — {파일:라인}
     - [ ] ISSUE: {설명} — {파일:라인}
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
- **출력**: 판정 (CLEAN / MECHANICAL FIX / CODE CHANGE REQUIRED) + 자동 수정된 코드
