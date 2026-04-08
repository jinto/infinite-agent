---
name: research
description: 다각도 분해 + 병렬 검색으로 외부 정보 조사
argument-hint: [research question]
---

# Research

질문을 독립 관점으로 분해하고 병렬 검색하여 소스 인용 포함 종합 보고서를 생성한다.

## 언제 사용

- 기술 비교 (예: "Prisma vs Drizzle ORM")
- 라이브러리/프레임워크 선택
- 경쟁사/시장 분석
- 디버깅 시 근본 원인 조사
- 외부 정보가 필요한 모든 상황

## 사용하지 말 것

- 코드베이스 내부 조사 → Grep/Glob 직접 사용
- 아이디어 정리 → `/ina:think`
- 단순 질문 → 직접 대화

## ina 연동

ina 데몬에 의해 실행된 경우:

- 다각도 분해 후: `ina_report_progress(in_progress="관점 N개 병렬 검색 중", remaining="종합")`
- 종합 시작: `ina_report_progress(in_progress="종합 보고서 작성", completed="관점 검색")`
- 3-strike: `ina_mark_blocked(reason="검색 3회 연속 실패 — 검색어 조정 필요")`

## 흐름

### >>> Stage 1: 다각도 분해

질문을 2-5개 독립적인 검색 관점으로 분해한다.

예시: "Next.js vs Remix for our project"
→ 관점 1: Next.js 핵심 기능 및 장단점
→ 관점 2: Remix 핵심 기능 및 장단점
→ 관점 3: 성능 벤치마크 비교
→ 관점 4: 커뮤니티 크기 및 생태계

### >>> Stage 2: 병렬 검색

각 관점을 별도 Agent로 병렬 실행:

```
Agent(prompt="관점 N에 대해 조사: ...", description="Research facet N")
```

각 Agent는:
- WebSearch로 관련 정보 검색
- WebFetch로 공식 문서, 블로그, GitHub 확인
- 소스 URL 인용 필수

### >>> Stage 3: 종합

모든 관점 결과를 종합하여 보고서 작성:
- 관점별 핵심 발견
- 비교 테이블 (해당 시)
- 추천 및 근거
- 모든 소스 URL 인용

### >>> 3-Strike Rule

검색이 3회 연속 유용한 결과를 찾지 못하면:
- `ina_mark_blocked(reason="검색 3회 연속 실패")` (데몬 연동 시)
- 즉시 멈추고 현재까지의 결과를 보고
- "추가 검색이 필요하면 검색어를 조정해주세요"라고 제안

## 출력물

- 종합 보고서: `docs/{YYYYMMDD-HHMM}-research-{slug}.md`

## 입출력

- **입력**: 자연어 조사 질문
- **출력**: `docs/{YYYYMMDD-HHMM}-research-{slug}.md`
