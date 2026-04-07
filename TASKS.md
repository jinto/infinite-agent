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
