---
name: reference
description: 스킬 카탈로그 + 시나리오별 라우팅 가이드. 스킬 선택이 필요할 때 자동 로딩.
user-invocable: false
---

# ina Reference

사용자의 요청을 분석하여 적절한 스킬을 선택하는 가이드.

## 스킬 카탈로그

| 스킬 | 호출 | 용도 |
|------|------|------|
| **autopilot** | `/ina:autopilot` | 아이디어 → 커밋 전자동 파이프라인 |
| **explore** | `/ina:explore` | "만들까 말까?" 탐색 — 시장조사 + GO/NO-GO/PIVOT 판정 |
| **think** | `/ina:think` | 아이디어 구체화 (기술/사업/개선) |
| **rethink** | `/ina:rethink` | 코드베이스 종합검진 + 수정 계획 |
| **plan** | `/ina:plan` | 합의 기반 플래닝 + TDD 태스크 분해 |
| **build** | `/ina:build` | TASKS.md 태스크 실행 (직접/subagent/team) |
| **review** | `/ina:review` | 커밋 전 코드 리뷰 + 자동 수정 |
| **research** | `/ina:research` | 다각도 분해 + 병렬 검색 |
| **design** | `/ina:design` | UI 구현 + 시각적 검증 |
| **test** | `/ina:test` | 독립 테스트 실행 + 실패 수정 |
| **ship** | `/ina:ship` | PR 생성 |
| **guard** | `/ina:guard` | 안전장치 (자동 적용, 직접 호출 불필요) |

## 시나리오별 라우팅

사용자의 요청을 읽고 아래 테이블에서 가장 적합한 스킬을 선택한다.

### "알아서 해줘" / 전자동 요청

| 시그널 | 스킬 |
|--------|------|
| "처음부터 끝까지", "알아서", "전자동", "autopilot" | `/ina:autopilot` |
| "만들어줘" + 모호한 요구사항 | `/ina:autopilot` |

### 탐색 단계 ("만들까 말까?")

| 시그널 | 스킬 |
|--------|------|
| "이거 어때?", "해볼까?", "할 가치 있어?" | `/ina:explore` |
| "이런 게 이미 있나?", "시장이 있나?" | `/ina:explore` |
| 스타트업 아이디어 현실성 진단 | `/ina:explore` (Startup 모드) |
| 사이드프로젝트 방향 탐색 | `/ina:explore` (Builder 모드) |

> **explore vs research**: explore는 "해야 하나?" 결정을 위한 질문+조사 통합. research는 결정 이후 순수 정보 수집.
> **explore vs think**: explore는 GO/NO-GO 판정이 정상 출력. think는 이미 만들기로 결정한 후 스펙 작성.
> **rethink vs think improve**: rethink는 전체 코드베이스 자동 전수조사 + codex 병렬 검토. think improve는 사용자 인터뷰 기반 개선 방향 탐색.

### 아이디어 / 요구사항 단계

| 시그널 | 스킬 |
|--------|------|
| "~하고 싶어", "~하면 어떨까", 모호한 아이디어 | `/ina:think` |
| "스타트업", "사업", "제품", 시장/수익 관련 | `/ina:think` (business 모드) |
| "리팩터", "개선", "정리", "성능" + 기존 코드 | `/ina:think` (improve 모드) |
| "전체 검토", "종합검진", "처음부터 다시", "코드 점검" | `/ina:rethink` |
| "조사해줘", "비교해줘", "찾아봐줘" | `/ina:research` |

### 플래닝 단계

| 시그널 | 스킬 |
|--------|------|
| "플랜", "계획", "어떻게 구현할지" | `/ina:plan` |
| 스펙 파일 경로 제공 + 구현 요청 | `/ina:plan` |
| TASKS.md가 없고 복잡한 기능 | `/ina:plan` |

### 실행 단계

| 시그널 | 스킬 |
|--------|------|
| TASKS.md가 있고 구현 요청 | `/ina:build` |
| 구체적 파일/라인 언급 + 버그 수정 | `/ina:build` |
| 여러 독립 태스크 동시 수정 | `/ina:build` (병렬 위임) |
| UI/프론트엔드 구현 | `/ina:design` |

### 검증 / 품질 단계

| 시그널 | 스킬 |
|--------|------|
| "테스트 돌려봐", "테스트 수정" | `/ina:test` |
| "리뷰해줘", "커밋 전에 확인" | `/ina:review` |
| "PR 만들어줘", "PR 생성" | `/ina:ship` |

## 파이프라인 흐름

```
explore → think → plan → build → review → commit
  ↑          ↑                      ↑         │
  │          └──────────────────────┘         │
  │          (루프백 최대 3회)                 │
  │                                           │
  NO-GO/PIVOT (종료 또는 재탐색)              │
                                              │
  └─ 어디서든 시작 가능 (중간부터 가능)  ←───┘
```

- "만들까 말까?" 모를 때 → explore부터
- 만들기로 결정했지만 스펙이 모호 → think부터
- 스펙이 이미 있으면 → plan부터
- TASKS.md가 있으면 → build부터
- 코드가 있고 리뷰만 필요 → review부터
- 전부 자동 → autopilot

## 스킬 선택 우선순위

1. 사용자가 명시적으로 스킬을 지정하면 그것을 사용
2. 지정하지 않으면 요청의 **모호성 수준**으로 판단:
   - 모호함 (뭘 만들지 불명확) → `think`
   - 명확하지만 플랜 없음 → `plan`
   - 플랜 있고 구현만 남음 → `build`
   - 구현 완료, 리뷰 필요 → `review`
   - 전부 자동 → `autopilot`
3. 판단이 어려우면 사용자에게 물어본다

## 스킬 간 연결

| 스킬 완료 후 | 다음 추천 |
|-------------|----------|
| explore (GO) | `/ina:think` (스펙 구체화) |
| explore (PIVOT) | `/ina:explore` (새 방향으로 재실행) |
| explore (NO-GO) | 종료 또는 새 아이디어로 재시작 |
| think | `/ina:plan` |
| rethink | `/ina:plan` (수정 계획을 태스크로 분해) |
| plan | `/ina:build` |
| build | `/ina:review` |
| design | `/ina:review` |
| review (CLEAN) | 문서 업데이트 → 커밋 → `/ina:ship` |
| review (MECHANICAL FIX) | 문서 업데이트 → 커밋 → `/ina:ship` |
| review (CODE CHANGE REQUIRED) | `/ina:build` (이슈 수정) |
| research | `/ina:think` 또는 `/ina:plan` |

## MCP 도구

| 도구 | 용도 |
|------|------|
| `ina_report_progress` | 진행 상황 보고 (데몬 연동) |
| `ina_mark_blocked` | 차단 상태 보고 |
| `ina_check_agents` | 다른 에이전트 상태 조회 |

데몬 MCP 호출이 실패하면 무시하고 계속 진행.

## 입출력

- **입력**: 없음 (자동 로딩)
- **출력**: 없음 (가이드 참조용)
