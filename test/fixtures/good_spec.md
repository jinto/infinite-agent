# EVAL-FIXTURE: good-spec
# This spec is well-defined for eval testing.

# OAuth2 Authentication with Google

## Goal

사용자가 Google 계정으로 로그인할 수 있는 OAuth2 인증 시스템을 추가한다. 기존 세션 기반 인증은 유지하되, OAuth2를 대안 로그인 방법으로 제공한다.

## Constraints

- Go 1.23 + `golang.org/x/oauth2` 패키지 사용
- Google OAuth2 provider만 (추후 GitHub 확장 가능하도록 인터페이스 설계)
- 기존 `users` 테이블에 `oauth_provider`, `oauth_id` 컬럼 추가
- 환경변수: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `OAUTH_REDIRECT_URL`
- PKCE flow 사용 (public client)
- 토큰은 서버 사이드 세션에 저장 (클라이언트에 노출 금지)

## Non-Goals

- 소셜 프로필 동기화 (이름/사진 가져오기 안 함)
- 다중 OAuth provider (Google만)
- 기존 패스워드 인증 제거

## Acceptance Criteria

1. `/auth/google` 엔드포인트로 OAuth2 flow 시작
2. Google 인증 후 `/auth/google/callback`으로 리다이렉트, 세션 생성
3. 신규 사용자: 자동 계정 생성 (oauth_provider="google", oauth_id=sub)
4. 기존 사용자 (이메일 매칭): 기존 계정에 OAuth 연결
5. 환경변수 미설정 시 OAuth 라우트 비활성화 (에러 아닌 graceful skip)
6. CSRF 방지: state 파라미터 검증
7. 단위 테스트: provider 인터페이스 mock으로 flow 검증
