# NFR Requirements — U5 Web Frontend

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 NFR-2/3/4/6/7, `construction/u5-web-frontend/functional-design/*.md`, `plans/u5-web-frontend-nfr-requirements-plan.md`

---

## 1. NFR 영역별 요구사항

### 1.1 Performance (NFR-2)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U5-P1 | wire 메시지 수신 → DOM 갱신 지연 | **p99 < 100 ms** (LAN, 12 PLAYER) | DevTools Performance 측정 |
| NFR-U5-P2 | `announce` 수신 → SpeechSynthesis.speak 호출 | **p99 < 200 ms** | console.time 측정 |
| NFR-U5-P3 | 첫 페이지 로드 (FCP) | < 1초 (LAN, 캐시 미사용) | Lighthouse |
| NFR-U5-P4 | 빌드 산출물 크기 (gzip) | **< 500 KB** | `vite build` 출력 |
| NFR-U5-P5 | 브라우저 메인 스레드 freeze 없음 | 모든 핸들러 < 50ms | React Profiler |

### 1.2 Usability (NFR-3)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U5-U1 | 한국어 UI / 안내 | 100% (영어 fallback 없음) | 코드 리뷰 |
| NFR-U5-U2 | PublicView 자막 폰트 | **≥ 32px** | CSS 모듈 검증 |
| NFR-U5-U3 | PublicView 단계 헤더 폰트 | **≥ 48px** | CSS 검증 |
| NFR-U5-U4 | PlayerView 모바일 반응형 | 320px (iPhone SE) ~ 1920px 정상 표시 | Chrome DevTools 디바이스 시뮬레이션 |
| NFR-U5-U5 | 사망 플레이어 시각 표시 | X 표식 + 회색 처리 | 단위 테스트 |
| NFR-U5-U6 | TimerBar는 마지막 10초 빨강 강조 | 코드 검증 | — |

### 1.3 Reliability

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U5-R1 | WS 끊김 자동 재연결 | 지수 백오프 1/2/4/8/16s, 16초 후 무한 재시도 (Q-NFR-U5-12=A) | 단위 테스트 |
| NFR-U5-R2 | 재연결 직후 토큰 자동 resume | localStorage 토큰 있으면 100% 자동 호출 | 단위 테스트 |
| NFR-U5-R3 | TTS 부재 환경 자막 폴백 | window.speechSynthesis === undefined 시 토스트 + 자막만 | 단위 테스트 (mock) |
| NFR-U5-R4 | 알 수 없는 wire kind 무시 | console.warn + 페이지 정상 작동 | 단위 테스트 |

### 1.4 Maintainability (NFR-6)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U5-M1 | TypeScript strict + noUncheckedIndexedAccess | tsconfig 설정 | 코드 리뷰 |
| NFR-U5-M2 | ESLint 통과 | `eslint .` 0 error | CI 게이트 |
| NFR-U5-M3 | 핵심 모듈 단위 테스트 라인 커버리지 | **≥ 70%** (Q-NFR-U5-5=A — reducer, useTTSQueue, NicknameForm 등) | `vitest --coverage` |
| NFR-U5-M4 | npm 직접 의존 한정 | 7종 (react/react-dom/react-router-dom/vite/typescript/@types/react/@types/react-dom) + 테스트 4종 | `package.json` 리뷰 |
| NFR-U5-M5 | wire 타입 단일 진실 소스 | `web/src/types/wire.ts` | 코드 리뷰 |
| NFR-U5-M6 | 컴포넌트당 ≤ 200 LOC | 큰 컴포넌트는 분할 | 코드 리뷰 |

### 1.5 Security (NFR-4)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U5-S1 | 토큰을 화면/HTML/console.log에 노출 안 함 | 단위 테스트로 grep | (Q-NFR-U5-11=A) |
| NFR-U5-S2 | Role/Keyword 마스킹은 백엔드 신뢰 — 자체 마스킹 안 함 | 코드 리뷰 |
| NFR-U5-S3 | XSS 방어는 React 기본 (innerHTML 미사용) | 코드 리뷰 |
| NFR-U5-S4 | 토큰 무효화 시 localStorage 즉시 삭제 (`UNKNOWN_PLAYER_ERROR`) | 단위 테스트 |

### 1.6 Accessibility (NFR-U5-A)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U5-A1 | TTS 부재 시 자막만으로 게임 진행 가능 | 단위 테스트 (시나리오 6) | (Q-NFR-U5-10=A) |
| NFR-U5-A2 | 키보드 네비게이션 (Tab으로 모든 입력 도달) | 수동 검증 | — |
| NFR-U5-A3 | 색상 대비 (자막 / 배경) | 기본 라이트/다크 테마 대비 ≥ 4.5:1 | DevTools 측정 |
| NFR-U5-A4 | WCAG AA 명시 추구 안 함 (PoC) | — | — |

---

## 2. 트레이드오프 결정

| 트레이드오프 | 본 단위의 결정 | 근거 |
|---|---|---|
| 상태 관리 (Redux vs Context) | **Context + useReducer** | 외부 lib 0, 작은 스펙 |
| CSS (Tailwind vs Modules) | **CSS Modules** | 외부 lib 0, 명시적 스타일 |
| 라우터 (커스텀 vs react-router) | **react-router-dom v6** | 표준, 사용자 명시 |
| TTS 큐잉 (즉시 vs 적층) | **적층 + urgent 인터럽트** | UX 자연스러움 |
| 토큰 저장 (localStorage vs cookie) | **localStorage** | LAN 환경, 단순성 |
| 단위 테스트 커버리지 (70 vs 85) | **≥ 70%** | UI 부분은 통합 테스트가 더 효과적 (PoC 비-요구) |

---

## 3. 추적성 (FR/NFR ↔ U5 NFR)

| 출처 | 본 문서 항목 |
|---|---|
| NFR-2 (성능) | NFR-U5-P1~P5 |
| NFR-3 (사용성) | NFR-U5-U1~U6 |
| NFR-4 (비공개) | NFR-U5-S1~S4 |
| NFR-6 (유지보수성) | NFR-U5-M1~M6 |
| NFR-7 (외부 서비스 0) | NFR-U5-M4 |
| FR-1.2 (재연결) | NFR-U5-R1~R2 |
| FR-2.3 (역할 비공개) | NFR-U5-S2 |
| FR-8.1 (Web Speech) | NFR-U5-P2, NFR-U5-R3 |
| FR-8.7 (자막 폴백) | NFR-U5-A1 |

---

## 4. 검증 게이트 (Build & Test 단계)

다음 모든 항목이 통과해야 U5가 출하 가능:

1. ✅ `tsc --noEmit` 0 error (TS strict)
2. ✅ `eslint .` 0 error
3. ✅ `vitest run` 모든 테스트 통과
4. ✅ `vitest --coverage` 라인 ≥ **70%**
5. ✅ `vite build` 성공 + gzip 산출물 < 500 KB
6. ✅ Lighthouse FCP < 1초 (LAN)
7. ✅ Chrome DevTools mobile (320px) 표시 정상
8. ✅ 토큰 노출 단위 테스트 통과 (DOM grep)

---

## 5. 명시적 비-요구사항 (Non-Goals)

- **Scalability**: 단일 호스트 PC, LAN 한정.
- **Availability SLA**: 미정의.
- **i18n / 다국어**: 한국어 한정 (FR-8.3).
- **WCAG AA 인증**: PoC 범위 외 (A1 자막 폴백만 보장).
- **Service Worker / PWA**: 비-요구.
- **모바일 네이티브 앱 (React Native)**: 비-요구.
- **Server-Side Rendering**: 클라이언트 SPA만.
- **E2E 자동화 (Playwright)**: 비-요구.
- **Bundle analyzer / 시각화 도구**: 비-요구.
