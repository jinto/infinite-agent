# ina — Infinite Angel

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README.md)

**멈추지 않는 코딩 천사.**

_잠든 사이, 코드가 완성됩니다._

[시작하기](#시작하기) • [스킬](#스킬) • [파이프라인](#파이프라인) • [참고](#참고)

---

## 시작하기

```bash
# 1. 설치 (바이너리 + 설정 자동 완료)
curl -sSL https://raw.githubusercontent.com/jinto/infinite-angel/main/install.sh | sh
source ~/.zshrc  # 또는 새 터미널 열기

# 2. 데몬 등록 (로그인 시 자동 시작)
ina install

# 3. 스킬 설치 (Claude Code에서)
/plugin marketplace add https://github.com/jinto/infinite-angel
/plugin install ina

# 4. 사용예
/ina:autopilot 로그인 기능을 구현하시오.

# 삭제 (데몬, hooks, HUD, MCP 서버 제거)
ina uninstall          # 설정 유지 (~/.ina)
ina uninstall --purge  # 전부 삭제
```

---

## 뭐가 다른가

다른 Claude Code 플러그인은 **프롬프트 모음**이다. ina는 **인프라**다.

| | 프롬프트 모음 | ina |
|---|---|---|
| 에이전트가 죽으면 | 그걸로 끝 | **데몬이 살려서 이어감** |
| 컨텍스트가 날아가면 | 처음부터 다시 | **파이프라인 상태가 남아 있음** |
| 오래 걸리는 작업 | 잘 되고 있는지 알 수 없음 | **실시간 모니터링 + Discord 알림** |
| 에이전트 여러 개 | 직접 관리 | **레지스트리에서 자동 조율** |

---

## 스킬

```
  EXPLORE        THINK         PLAN          BUILD (구현→리뷰→커밋)       SHIP
 ┌─────────┐   ┌──────┐     ┌──────┐     ┌─────────────────────┐     ┌──────┐
 │ 만들까? │   │아이디어│    │ 플랜 │     │ 구현 → 3-lane 리뷰  │     │  PR  │
 │         │──▶│  스펙  │──▶ │태스크│ ──▶ │ → fix-first → 커밋  │ ──▶ │ 머지 │
 └─────────┘   └──────┘     └──────┘     └─────────────────────┘     └──────┘
GO/NO-GO/
  PIVOT
```

| 스킬 | 설명 |
|------|------|
| `explore` | "이거 해볼까?" — 시장조사 인라인 + GO / NO-GO / PIVOT 판정 |
| `autopilot` | 전체 파이프라인 — think → plan → build |
| `think` | 아이디어를 스펙으로 (기술 / 사업 / 개선) |
| `rethink` | 코드베이스 종합검진 — 전수조사 + codex 병렬 검토 + 수정 계획 |
| `plan` | 합의 기반 플래닝 + TDD 태스크 분해 |
| `build` | 구현 → 리뷰 → 커밋 한방 (3-lane 리뷰 내장) |
| `review` | 병렬 3-lane 리뷰 단독 실행 (adversarial + security + simplify) |
| `research` | 질문을 여러 각도로 나눠서 병렬 조사 |
| `design` | UI 구현 + 눈으로 확인 |
| `test` | 테스트 돌리고, 실패하면 원인 찾아서 수정 |
| `ship` | PR 만들기 — 요약은 자동 작성 |
| `guard` | 무인 실행 안전장치 — 위험한 건 알아서 막아줌 |

뭘 써야 할지 모르겠으면 그냥 하고 싶은 말만 하세요. 알아서 골라줍니다.

---

## 이런 식으로 씁니다

### "이거 만들 가치가 있을까?"

```
/ina:explore 코드 리뷰 비동기화 SaaS를 만들어볼까 하는데
```

Startup / Builder 모드 자동 감지 → 질문하면서 실시간 경쟁사 검색 → 시장 조사 → 핵심 전제 도전 → GO / NO-GO / PIVOT 판정 문서를 `.ina/specs/`에 저장.

### "아이디어가 모호해요"

```
/ina:think 사용자 인증을 추가하고 싶어
```

질문하면서 요구사항을 정리하고, 세 관점(Architect/Critic/CEO)에서 검증한 뒤 스펙 문서를 `.ina/specs/`에 저장.

### "이 코드 괜찮은 건가?"

```
/ina:rethink
```

전체 코드 전수조사 + codex 병렬 검토 → "처음부터 다시 만든다면?" → 수정 계획을 `.ina/specs/`에 저장. 코드는 건드리지 않습니다.

### "스펙은 있는데 어떻게 만들지 모르겠어요"

```
/ina:plan .ina/specs/20260405-1000-think-auth.md
```

Planner → Architect → Critic이 합의할 때까지 플랜을 다듬고, TDD 태스크로 쪼개서 TASKS.md로 저장.

### "할 일은 정해졌으니 만들어줘"

```
/ina:build
```

구현 → 3-lane 리뷰 → 커밋까지 한방으로. 태스크가 1개면 직접, 2-3개면 서브에이전트, 4개 이상이면 팀으로.

### "처음부터 끝까지 알아서"

```
/ina:autopilot OAuth2로 사용자 인증 추가
```

아이디어 정리부터 커밋까지 전자동. 중간에 죽어도 `.state/pipeline.json`에서 이어갑니다.

### "커밋하기 전에 한번 봐줘"

```
/ina:review
```

Adversarial + Security + Simplify 3개 레인으로 병렬 리뷰. 고칠 수 있는 건 바로 고칩니다.

### "테스트 좀 돌려봐"

```
/ina:test
```

실패하면 원인을 찾아서 고치고 다시 돌립니다. 최대 5번.

### "PR 올려줘"

```
/ina:ship
```

변경 내용을 요약하고, 테스트 확인한 뒤 PR을 만듭니다.

---

## 파이프라인

```
explore ──GO──▶ think → plan → build → review → commit
   │                              ↑         │
PIVOT/NO-GO                       └─────────┘ (루프백 최대 3회)
   │
   ▼
 종료 또는
재탐색
```

**죽어도 이어감** — `.state/pipeline.json`에 어디까지 했는지 기록해둡니다. 데몬이 에이전트를 다시 띄우면 그 지점부터 계속합니다.

**세 관점에서 검증** — think와 plan 단계에서 Architect, Critic, CEO 역할의 서브에이전트 3개가 동시에 검토합니다.

**알아서 나눠서 실행** — build가 태스크 수를 보고 직접 실행할지, 서브에이전트로 나눌지, 팀으로 돌릴지 판단합니다.

**플랜 크기 조절** — 플랜은 3~7개 태스크를 목표로 합니다. 더 큰 작업은 여러 플랜으로 나눠서 TASKS.md에 섹션으로 관리합니다.

---

## 생성 파일

```
.ina/specs/                                ← 판정, 스펙, 분석
├── {YYYYMMDD-HHMM}-explore-{slug}.md
├── {YYYYMMDD-HHMM}-think-{slug}.md
└── {YYYYMMDD-HHMM}-rethink-{slug}.md

.claude/plans/{slug}.md                    ← 실행 계획
TASKS.md                                   ← 태스크 체크리스트 (플랜별 섹션)

docs/{YYYYMMDD-HHMM}-research-{slug}.md    ← 참고 자료

.state/pipeline.json                       ← autopilot 크래시 복구
```

---

## 구조

```
┌─────────────────────────────────────────────┐
│  스킬 레이어 (SKILL.md)                      │
│  세션 안에서 Claude Code 도구로 조율           │
│                                             │
│  explore / autopilot / think / rethink /    │
│  plan / build / review / research /         │
│  design / test / ship / guard               │
├─────────────────────────────────────────────┤
│  데몬 레이어 (Go 바이너리)                     │
│  세션 밖에서 프로세스를 지켜봄                  │
│  죽으면 살리고, 막히면 알려줌                   │
│                                             │
│  ina daemon / watcher / hooks / MCP         │
└─────────────────────────────────────────────┘
```

---

## CLI 명령어

| 명령어 | 설명 |
|--------|------|
| `ina setup` | Claude Code hooks + MCP 서버 설치 |
| `ina install` | 데몬을 macOS 시작 프로그램으로 등록 |
| `ina uninstall` | 데몬, hooks, HUD, MCP 서버 제거 |
| `ina upgrade` | ina 최신 버전으로 업그레이드 |
| `ina status` | 전체 에이전트 상태 조회 |
| `ina launch <경로> <태스크>` | 프로젝트에 새 에이전트 실행 |
| `ina restart <이름\|ID>` | 죽거나 멈춘 에이전트 재시작 |
| `ina stop <이름\|ID>` | 에이전트 중지 |
| `ina attach <이름>` | 에이전트 로그에 실시간 연결 |
| `ina log [에이전트명]` | 에이전트 또는 데몬 로그 보기 |
| `ina hud on\|off` | Claude Code HUD 상태바 켜기/끄기 |
| `ina daemon` | 워치독 데몬 직접 실행 (보통 launchd가 관리) |

---

## 빌드 & 테스트

```bash
go build -o ina .
go build -o ina-mcp ./mcp/
go test ./... -count=1 -race

# E2E 스킬 라우팅 테스트 (API 크레딧 소모)
INFA_E2E=1 go test ./test/ -run TestSkillRouting -v

# LLM-Judge eval — 스킬 품질 검증 (API 크레딧 소모)
INFA_EVAL=1 go test ./test/ -run TestSkillEval -v -timeout 600s
```

---

## 참고

- [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) — 멀티에이전트 오케스트레이션
- [superpowers](https://github.com/obra/superpowers) — 프로세스 규율 + TDD 강제
- [gstack](https://github.com/garrytan/gstack) — Solo builder 소프트웨어 공장
- [agent-skills](https://github.com/addyosmani/agent-skills) — Google SWE 문화 인코딩

---

## 라이센스

[MIT](LICENSE)
