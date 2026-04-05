# ina — Infinite Agent

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[English](README.md)

**멈추지 않는 코딩 에이전트.**

_에이전트를 띄우고 잠들면, 아침에 코드가 완성되어 있다._

[시작하기](#시작하기) • [스킬](#스킬) • [파이프라인](#파이프라인) • [참고](#참고)

---

## 시작하기

```bash
# 1. 바이너리 설치
curl -sSL https://raw.githubusercontent.com/jinto/infinite-agent/main/install.sh | sh
source ~/.zshrc  # 또는 새 터미널 열기

# 2. 스킬 설치 (Claude Code에서)
/plugin marketplace add https://github.com/jinto/infinite-agent
/plugin install ina

# 3. 설정 (Claude Code 훅 + MCP)
ina setup

# 4. 데몬 시작 (택 1)
ina install   # 추천: 로그인 시 자동 시작 (macOS launchd)
ina daemon    # 또는: 포그라운드 실행

# 5. 사용
/ina:autopilot 사용자 인증 추가
```

---

## 뭐가 다른가

다른 Claude Code 플러그인은 **프롬프트 라이브러리**다. ina는 **인프라**다.

| | 프롬프트 라이브러리 | ina |
|---|---|---|
| 에이전트 크래시 | 끝 | **데몬이 재시작 + 재개** |
| 컨텍스트 리셋 | 처음부터 | **파이프라인 상태 보존** |
| 장시간 작업 | 기도 | **모니터링 + 알림** |
| 멀티 에이전트 | 수동 | **레지스트리 + 조율** |

---

## 스킬

```
  THINK         PLAN          BUILD         REVIEW        SHIP
 ┌──────┐     ┌──────┐     ┌──────┐     ┌──────┐     ┌──────┐
 │아이디어│ ──▶ │ 플랜 │ ──▶ │ 구현 │ ──▶ │ 리뷰 │ ──▶ │  PR  │
 │ 스펙  │     │태스크│     │테스트│     │ 수정 │     │ 머지 │
 └──────┘     └──────┘     └──────┘     └──────┘     └──────┘
```

| 스킬 | 설명 |
|------|------|
| `autopilot` | 전체 파이프라인: think → plan → build → review → commit |
| `think` | 아이디어 → 스펙 (기술 / 사업 / 개선) |
| `plan` | 합의 기반 플래닝 + TDD 태스크 분해 |
| `build` | 태스크 실행 (직접 / 서브에이전트 / 팀) |
| `review` | 코드 리뷰 + 자동 수정 |
| `research` | 패싯 분해 + 병렬 검색 |
| `design` | UI 구현 + 시각적 검증 |
| `test` | 테스트 실행 + 실패 분석 + 수정 |
| `ship` | PR 생성 (변경 요약 자동 작성) |
| `guard` | 무인 실행 안전장치 |

---

## 파이프라인

```
autopilot: think → plan → build → review → commit
                                    ↑         │
                                    └─────────┘ (루프백, 최대 3회)
```

**크래시 복구** — `.state/pipeline.json`으로 상태 추적. 데몬이 에이전트를 재시작하면 기록된 단계부터 재개.

**다관점 검증** — think와 plan에서 Architect, Critic, CEO 서브에이전트를 병렬로 실행하여 스펙과 플랜을 검증.

**실행 위임** — build가 태스크 독립성을 판단하여 직접 실행, 서브에이전트 병렬, 팀 조율을 자동 선택.

---

## 아키텍처

```
┌─────────────────────────────────────────┐
│  스킬 레이어 (SKILL.md)                  │
│  세션 내 오케스트레이션                     │
│  Claude Code 네이티브 도구 활용            │
│                                         │
│  autopilot / think / plan / build /     │
│  review / test / ship / guard           │
├─────────────────────────────────────────┤
│  데몬 레이어 (Go 바이너리)                 │
│  세션 외 프로세스 감시                     │
│  크래시 복구, 재시작, Discord 알림          │
│                                         │
│  ina daemon / watcher / hooks / MCP     │
└─────────────────────────────────────────┘
```

---

## 빌드 & 테스트

```bash
go build -o ina .
go build -o ina-mcp ./mcp/
go test ./... -count=1 -race

# E2E 스킬 라우팅 테스트 (API 크레딧 소모)
INFA_E2E=1 go test ./test/ -run TestSkillRouting -v
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
