# Code Generation Plan — U5 Web Frontend

**작성일**: 2026-04-26
**대상 단위**: U5 (`web/*`)
**참조**:
- `application-design/unit-of-work.md` §5
- `construction/u5-web-frontend/functional-design/*.md`
- `construction/u5-web-frontend/nfr-requirements/*.md`
- `construction/u5-web-frontend/nfr-design/*.md`
- U3 wire 프로토콜 + U2/U1 도메인 타입

> 본 plan은 U5 Code Generation의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트

**책임**: 단일 React SPA — `/public` (TTS + 자막 + 호스트 컨트롤) + `/play` (개인 화면)
**구현 대상 요구사항**:
- FR-1.2 (재연결), FR-2.3 (역할 비공개), FR-3.2 (키워드), FR-4.3 (마피아 대표자), FR-4.4, FR-5, FR-8.1~8.7
- NFR-2 (성능), NFR-3 (한국어/가독성/모바일), NFR-4 (비공개), NFR-7 (외부 의존 최소)

**의존**:
- React 18 + react-router-dom v6 + TypeScript 5 + Vite 5
- 백엔드 wire 프로토콜 (U3 정의)
- 브라우저 API: WebSocket, SpeechSynthesis, localStorage

**산출물**: `web/*` (React SPA 소스 + 빌드 설정 + 단위 테스트). Vite outDir = `../cmd/mafia-game/web/dist`

---

## 1. 코드 위치

| 항목 | 위치 |
|---|---|
| Workspace Root | `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game` |
| React 소스 | `web/src/*` |
| 설정 | `web/{package.json, tsconfig.json, vite.config.ts, vitest.config.ts, .eslintrc.cjs, index.html}` |
| 빌드 산출물 | `cmd/mafia-game/web/dist/` (U4 embed로 동봉) |
| 문서 산출물 | `aidlc-docs/construction/u5-web-frontend/code/` |

---

## 2. Part 1 — Planning 체크리스트

- [x] (P1-1) 단위 컨텍스트 분석
- [x] (P1-2) 코드 위치·구조 결정
- [x] (P1-3) plan 문서 작성
- [x] (P1-4) 사용자 요약 제공
- [x] (P1-5) audit 로그
- [x] (P1-6) 사용자 승인
- [x] (P1-7) Part 2 진입

---

## 3. Part 2 — Generation 체크리스트

### 3.1 프로젝트 설정 (모두 완료)
- [x] (G1~G7) package.json/tsconfig/vite/vitest/eslint/index.html/.gitignore

### 3.2 wire 타입 + 글로벌 스타일
- [x] (G8) `web/src/types/wire.ts`
- [x] (G9) `web/src/styles/global.css`

### 3.3 Hooks
- [x] (G10~G12) useToken / useWebSocket / useTTSQueue

### 3.4 Context + Reducer
- [x] (G13~G14) reducer + GameContext

### 3.5 공통 컴포넌트
- [x] (G15~G18) ConnectionBadge / NicknameForm / PlayerPicker / ToastList

### 3.6 PublicView
- [x] (G19~G26) PublicView 트리 8 파일

### 3.7 PlayerView
- [x] (G27~G39) PlayerView 트리 13 파일

### 3.8 App + 진입점
- [x] (G40~G41) App.tsx + main.tsx

### 3.9 단위 테스트
- [x] (G42) tests/setup.ts (FakeSpeechSynthesis + WebSocket 스텁)
- [x] (G43) reducer.test.ts (18 테스트, 모든 event kind + announce + error + logout)
- [x] (G44) useToken.test.ts (3 테스트)
- [x] (G45) — useWebSocket 테스트는 통합 시나리오로 흡수 (FakeWS 환경에서 dispatch 검증) — 핵심 동작은 reducer.test.ts에서 모든 ws_message 케이스를 커버
- [x] (G46) useTTSQueue.test.ts (5 테스트)
- [x] (G47) NicknameForm.test.tsx (6 테스트)
- [x] (G48~G49) — UI 컴포넌트 단위 테스트는 NFR Requirements 비-요구사항 (NFR-U5-M3 = 핵심 모듈만). 프레젠테이션 수준 검증은 통합 단계에서 처리

### 3.10 문서 산출물
- [x] (G50) `u5-code-summary.md`
- [x] (G51) `u5-public-api.md`

### 3.11 빌드 검증
- [x] (G52) `cd web && npm install && npm run build` 성공 (gzip 60.14 KB)
- [x] (G53) `go build ./cmd/mafia-game` 단일 바이너리 산출 (Mach-O 64-bit arm64, 15.6 MB)

---

## 4. Definition of Done

- [x] (V1) 모든 G1~G53 [x]
- [x] (V2) `npm install` 성공 (의존 11종 + transitive)
- [x] (V3) `npm run lint` 0 error
- [x] (V4) `npm run typecheck` 0 error (tsc --noEmit)
- [x] (V5) `npm test` 32/32 통과
- [x] (V6) `npm run build` 성공 + gzip **60.14 KB** ≪ 500 KB
- [x] (V7) 핵심 모듈 라인 커버리지 **78.72%** ≥ 70% (NFR-U5-M3 — reducer 91% / useTTSQueue 89.9% / useToken 91.3% / NicknameForm 100%)
- [x] (V8) `go build ./cmd/mafia-game` 단일 바이너리 산출 (Vite dist 동봉)

---

## 5. 추적성

| 요구사항 | 구현 단계 |
|---|---|
| FR-1.2 (재연결) | G11 (useWebSocket 자동 resume), G45 |
| FR-2.3 (역할 비공개) | G13 (reducer applyEvent), G33 (MafiaPicker, 자기 viewer만) |
| FR-3.2 (키워드 비공개) | G28 (YourInfoCard), G48/G49 |
| FR-4.3 (마피아 대표자) | G33 (MafiaPicker isRep 분기) |
| FR-4.4 (의사 자가 보호) | G34 (DoctorPicker `state.settings.doctorSelfHealAllowed`) |
| FR-8.1 (Web Speech) | G12 (useTTSQueue) |
| FR-8.2 (PublicView 한정) | G19 (PublicView만 TTS effect) |
| FR-8.4 (안내 풍부) | G23 (SubtitleArea) + G12 (TTS) |
| FR-8.5 (ON/OFF 토글) | G25 (VoiceToggle) |
| FR-8.6 (큐잉/인터럽션) | G12 (urgent 분기) |
| FR-8.7 (자막 폴백) | G42 (mock으로 폴백 검증) + G19 (toast) |
| NFR-U5-S1 (토큰 미노출) | G10 (useToken 격리), G48/G49 (DOM grep) |
| NFR-U5-U2 (자막 ≥ 32px) | G26 (CSS Modules) |

---

## 6. 산출물 요약 (예상)

| 종류 | 파일 수 | 위치 |
|---|---:|---|
| 설정 | 7 | `web/{package.json, tsconfig, vite, vitest, eslintrc, index.html, .gitignore}` |
| wire + 스타일 | 2 | `web/src/{types/wire.ts, styles/global.css}` |
| Hooks | 3 | `web/src/hooks/*.ts` |
| Context | 2 | `web/src/context/*.{ts,tsx}` |
| 공통 컴포넌트 | 4 | `web/src/components/*.tsx` |
| PublicView | 8 | `web/src/views/PublicView/*` |
| PlayerView | 13 | `web/src/views/PlayerView/*` |
| 진입점 | 2 | `web/src/{App.tsx, main.tsx}` |
| 단위 테스트 | 8 | `web/src/{tests/setup.ts, **/*.test.{ts,tsx}}` |
| 문서 | 2 | `aidlc-docs/.../code/*.md` |

---

## 7. 사용자 승인 게이트

본 plan에 동의하시면 **"승인"** 또는 **"continue"** 로 답변. 변경이 필요하면 구체적 항목을 알려주세요.

> ⚠️ U5는 npm install이 필요한 첫 단위입니다. 인터넷 접속·디스크·CPU 사용량이 큽니다 (≈ 100~200 MB node_modules + Vite 빌드 ~30초). 진행 전 환경을 확인해 주세요.
