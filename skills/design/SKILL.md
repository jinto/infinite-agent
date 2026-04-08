---
name: design
description: 프레임워크 감지 + UI 구현 + 시각적 검증
argument-hint: [design request]
---

# Design

���레임워크를 감지하고, 미적 방향을 설정한 후, 프로덕션 수준의 UI를 구현하고 검증한다.

## 언제 사용

- UI/UX 구현 또는 수정
- 디자인 품질 개선
- 시각적 일관성 점검
- "이거 예쁘게 만들어줘"

## 사용하지 말 것

- 코드 리뷰 → `/ina:review`
- 로직 구현 → `/ina:build`

## ina 연동

ina 데몬에 의해 실행된 경우:

- 각 Stage 진입 시: `ina_report_progress(in_progress="Stage N: {name}")`
- 검증 실패 ���: `ina_mark_blocked(reason="렌더링/접근성 검증 실패: {detail}")`

## 흐름

### >>> Stage 1: 프레임워크 감지

프로젝트의 프레임워크를 자동 감지:
- React / Next.js / Vue / Angular / Svelte / Solid 등
- CSS 방식: Tailwind / CSS Modules / styled-components / vanilla CSS
- 기존 디자인 패턴 파악 (컴포넌트 구조, 색상, 타이포��래피)

### >>> Stage 2: 미적 방향 ��정

구현 전에 방향을 명확히:
- **톤**: 미니멀 / 플레이풀 / 프로페셔널 / 대시보드 등
- **제약**: 기존 디자인 시스템과의 일관성
- **차별화**: 어떤 점에서 돋보여야 하는지

사용자와 방향 합의 후 진행.

### >>> Stage 3: 컴포넌트 구현

- 프레임워크 관용적인 코드 작성 (React면 React답게)
- 폰트: Arial, Inter, Roboto 같은 기본 폰트 지양 — 프로젝트에 맞는 폰트 선택
- CSS 변수 + 색상 조화 활용
- 애니메이션은 고임팩트 순간에만 (과하지 않게)

### >>> Stage 4: 검증

- 렌더링 확인 (빌드/실행)
- 반응형 레이아웃 확인
- 접��성 기본 점검 (contrast, alt text, keyboard nav)

### >>> Stage 5: Before/After 비교

- 변경 전후 상태를 비교하여 개선점 확인
- 스크린샷 가능 시 before/after 캡처

### >>> Stage 6: 원자적 커밋

- 변경 단위별로 원자적 커밋 (하나의 커밋에 하나의 개선)
- **커밋 전 반드시 사용자 허락**

## 출력물

- 구현된 디자인 코드 + 검증 결과

## 입출력

- **입력**: 디자인 요청 (자연어 또는 목업/스크린샷 참조)
- **출력**: 커밋된 디자인 코드
