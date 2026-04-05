# ina — Infinite Agent

멈추지 않는 코딩 에이전트.

Go 데몬(프로세스 감시, 크래시 복구, Discord 알림) + Claude Code 플러그인 스킬 10개 + autopilot 파이프라인 오케스트레이션.

## Skills

| 스킬 | 설명 |
|------|------|
| `autopilot` | 전체 파이프라인: think → plan → build → review → commit |
| `think` | 아이디어 → 스펙 (기술 / 사업 / 개선) |
| `plan` | 합의 기반 플래닝 + TDD 태스크 분해 |
| `build` | 태스크 실행 (직접 / 서브에이전트 / 팀) |
| `review` | 멀티모델 코드 리뷰 + 자동 수정 |
| `research` | 패싯 분해 + 병렬 검색 |
| `design` | UI 구현 + 시각적 검증 |
| `test` | 테스트 실행 + 실패 분석 + 수정 |
| `ship` | PR 생성 (변경 요약 자동 작성) |
| `guard` | 무인 실행 안전장치 |

## 파이프라인

```
autopilot: think → plan → build → review → commit
                                    ↑         │
                                    └─────────┘ (루프백, 최대 3회)
```

`.state/pipeline.json`으로 크래시 복구 — 데몬이 에이전트를 재시작하고 기록된 단계부터 재개.

## 빌드

```bash
go build -o ina .
go build -o ina-mcp ./mcp/
go test ./... -count=1 -race
```

## 참고

- [oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) — 멀티에이전트 오케스트레이션
- [superpowers](https://github.com/obra/superpowers) — 프로세스 규율, TDD 강제
- [gstack](https://github.com/garrytan/gstack) — Solo builder 소프트웨어 공장
- [agent-skills](https://github.com/addyosmani/agent-skills) — Google SWE 문화 인코딩

## 라이센스

MIT
