---
name: ship
description: PR 생성 — 변경 요약 + 테스트 확인
argument-hint: [--draft]
---

# Ship

커밋된 변경사항으로 PR을 생성한다. 변경 요약을 자동 작성하고, 테스트 통과를 확인한다.

## 언제 사용

- 구현 완료 후 PR을 만들 때
- autopilot 파이프라인 후 수동으로 PR 생성할 때

## 사용하지 말 것

- 아직 커밋 전 → `/ina:review` 먼저
- 구현이 안 끝났을 때 → `/ina:build`

## 인자

- (없음): 기본 PR 생성
- `--draft`: 드래프트 PR로 생성

## 흐름

### >>> Stage 1: 사전 확인

1. 테스트 실행 — 실패 시 경고 (계속 진행 여부 사용자 확인)
2. 현재 브랜치 확인 — main/master이면 경고
3. 리모트 존재 확인 — 없으면 push 필요성 안내

### >>> Stage 2: 변경 분석

1. `git log main..HEAD` — 커밋 히스토리 분석
2. `git diff main...HEAD` — 전체 변경사항 파악
3. 변경 요약 자동 작성:
   - 무엇이 변경되었는지 (bullet points)
   - 왜 변경했는지 (커밋 메시지에서 추론)
   - 테스트 계획

### >>> Stage 3: PR 생성

```bash
gh pr create --title "{제목}" --body "$(cat <<'EOF'
## Summary
{변경 요약}

## Test plan
{테스트 계획}
EOF
)"
```

- `--draft` 플래그 시 `gh pr create --draft` 사용
- PR URL을 사용자에게 반환

## 입출력

- **입력**: 커밋된 변경사항 (git log/diff)
- **출력**: PR URL
