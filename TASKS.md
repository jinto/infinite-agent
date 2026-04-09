# ina Tasks

## LLM-Judge Eval (Tier 3)

스펙: `.ina/specs/think-llm-judge-eval.md`
플랜: `.claude/plans/llm-judge-eval.md`

- [x] 시나리오 파싱 + 필터링 로직 (eval_scenarios.json 구조체 + EVAL_SKILLS 필터)
- [x] keyword rubric 채점 로직 (strings.Contains → 5/0점)
- [x] judge 프롬프트 생성 + JSON 파싱 (claude -p --model haiku, 재시도 1회)
- [x] 결과 저장 + regression 비교 (.state/eval/, ±1 tolerance, 부트스트랩)
- [x] fixture 파일 4개 작성 (review x2, plan x2)
- [x] eval_scenarios.json 작성 (4개 시나리오 + min_score + rubric)
- [x] TestSkillEval 통합 러너 (INFA_EVAL=1 게이팅, 10분 타임아웃)
- [x] pre-push.sh hook 스크립트 (diff 감지 → go test)
- [x] cmd/setup.go에 pre-push hook 설치 추가
- [x] Tier 1에 fixture 존재 검증 추가

## Hook Resilience

플랜: `.claude/plans/hook-resilience.md`
분석: `.ina/specs/20260409-0346-rethink-hook-resilience.md`

### Task 1: `cmd/hook.go` — ina hook 서브커맨드

- [x] 테스트: 소켓 없을 때 `ina hook post-tool-use` → exit 0, 에러 출력 없음
- [x] 테스트: 소켓 있을 때 stdin JSON이 daemon에 전달되고 exit 0
- [x] 테스트: 소켓 connect 500ms 초과 시 타임아웃 → exit 0
- [x] 구현: `cmd/hook.go` — stdin → unix 소켓 전달 → 실패 시 항상 exit 0
- [x] 구현: `daemon/protocol.go`에 `ActionHook` 상수 추가
- [x] 구현: `daemon/daemon.go`에 `handleHook` 핸들러 (기존 hook handler 위임)

### Task 2: `cmd/setup.go` — hook 등록 방식 전환

- [x] 테스트: 생성 hook이 `command` 타입 + `ina hook <event> 2>/dev/null || true`
- [x] 테스트: 기존 다른 도구의 PostToolUse hook이 보존됨 (append)
- [x] 테스트: 구 HTTP hook(`127.0.0.1:9111`) 감지 시 교체
- [x] 구현: `hookEntry` → command 타입, `|| true` 포함
- [x] 구현: hook 병합 — ina hook만 교체, 나머지 보존
- [x] 구현: 구 HTTP hook 감지 + 안내 메시지

### Task 3: `cmd/setup.go` — install 연동

- [x] 구현: setup 끝에 LaunchAgent plist 존재 확인 → 미등록 시 install 제안

### Task 4: `cmd/install.go` — cleanup 정밀화

- [x] 테스트: uninstall이 `ina hook` command만 제거, 다른 hook 보존
- [x] 구현: cleanup → `ina hook` 식별자 기반 매칭 (+ 구 HTTP hook 하위 호환)

### Task 5: 통합 검증

- [x] `go test ./... -count=1 -race` 통과
- [x] `ina hook post-tool-use` — 데몬 없이 exit 0 확인
- [ ] `ina setup` → settings.json에 command hook 등록 확인 (배포 후)

## Version Sync Check ← 현재

스펙: `.ina/specs/20260409-1200-think-version-sync-check.md`
플랜: `.claude/plans/version-sync-check.md`

- [x] Task 1: `maxVersion()` — 디렉토리 목록에서 최고 semver 반환 + 테스트
- [x] Task 2: `checkPluginVersion()` — 캐시 스캔 → 비교 → stdout 경고 + 테스트 (5개 시나리오)
- [x] Task 3: `cmd/hook.go` 통합 — session-start 이벤트에 체크 호출 추가

## Daemon Safety (예정)

- [ ] handleStop에서 watcher 중지
- [ ] 데몬 배타성 보장 (pid+flock)
- [ ] StopRunning pid 검증
- [ ] mostRecentlyActive → cwd/session 기반 매칭
- [ ] SessionStart auto-register 개선

## Quality (예정)

- [ ] daemon/cmd/mcp/notify 테스트 추가
- [ ] dead code 제거 (state/injector.go)
- [ ] repoSlug 충돌 방지
- [ ] CheckForUpdate 비동기화

## Completed

- [x] `ina log` command — tail agent or daemon logs
- [x] `--fresh` restart flag
- [x] `ina install` command — launchd auto-start
- [x] Unit tests — state parser, agent registry, config
- [x] Git worktree isolation
- [x] Log rotation
- [x] HTTP hook listener
- [x] `ina setup` command — hooks + MCP
- [x] `--continue` restart
- [x] MCP server — report_progress, mark_blocked, check_agents
- [x] `ina attach` command
