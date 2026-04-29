# Functional Design Plan — U5 Web Frontend

**작성일**: 2026-04-26
**대상 단위**: U5 / Web Frontend (`web/src/*`)
**컴포넌트**: C8 PublicView (`/public`), C9 PlayerView (`/play`)
**참조**:
- `aidlc-docs/inception/application-design/unit-of-work.md` §5
- `aidlc-docs/inception/application-design/component-methods.md` C8, C9
- U3 공개 API: `aidlc-docs/construction/u3-realtime-transport/code/u3-public-api.md` (와이어 프로토콜)
- U2 공개 API: `aidlc-docs/construction/u2-session-persistence-announce/code/u2-public-api.md` (안내 카탈로그·State 마스킹)
- U1 공개 API: `aidlc-docs/construction/u1-game-core/code/u1-public-api.md` (Action/Event/Phase)

> 본 plan은 U5 Functional Design의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트 분석

**목적**: 단일 React SPA에서 두 라우트(`/public`, `/play`)를 호스팅. WebSocket으로 백엔드 상태 수신·렌더링·입력 송신.

**책임 (PublicView, `/public`)**:
- 단계/타이머/사망자/투표 결과/종료 화면
- **Web Speech API**로 한국어 TTS 큐잉 + 자막 표시 (FR-8.1, FR-8.4, FR-8.6)
- 음성 ON/OFF 토글 (FR-8.5)
- Web Speech 부재 시 토스트 + 자막 폴백 (FR-8.7)
- 호스트 컨트롤 패널

**책임 (PlayerView, `/play`)**:
- 닉네임 입장 + 토큰 localStorage 저장 + 재연결
- 자기 역할·키워드 비공개 표시 (FR-2.3, FR-3.2) — **TTS 출력 없음** (FR-8.2)
- 단계별 입력 UI (마피아 살해 / 의사 보호 / 경찰 조사 / 투표)
- 마피아 대표자 입력 (Q-AD-7) — 다른 마피아는 현재 선택 확인

**비책임**: 비즈니스 규칙(U1), 영속화(U2), WebSocket 라우팅(U3), HTTP routing(U4).

**입력**:
- WebSocket 메시지 (welcome, joined, snapshot, event, announce, error)
- localStorage (token)

**출력**:
- WebSocket 메시지 (host:create-session, join, resume, host:start, submit:*, host:*)
- 화면 (자막, 큰 텍스트, 입력 폼)
- TTS 발화 (PublicView만)

**기술 스택**: React 18 + Vite 5 + TypeScript 5 + react-router-dom 6 + Web Speech API.
**외부 의존**: 위 4개 (npm). 추가 외부 lib 최소화.

---

## 1. 단계 체크리스트

- [x] (1) 단위 컨텍스트·책임·비책임 정의
- [x] (2) 결정 질문 작성 (Q-FD-U5-1~15)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증 — 모호성 없음
- [x] (6) `domain-entities.md` — wire 타입 + UI 상태 + 라우트
- [x] (7) `business-logic-model.md` — useWebSocket 훅 + useTTSQueue + 단계별 화면 + 호스트 컨트롤
- [x] (8) `business-rules.md` — BR-U5-* 라우팅·TTS·재연결·마스킹·입력 규칙
- [x] (9) `frontend-components.md` — 컴포넌트 트리 + props/state + 인터랙션 흐름
- [x] (10) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-FD-U5-1 ~ Q-FD-U5-15)

### Q-FD-U5-1. 라우팅 라이브러리

라우터 선택?

- **A.** **`react-router-dom v6`** (Q-AD-3=C 명시) — `BrowserRouter` + `Routes`/`Route`. 표준 React 라우팅.
- **B.** 커스텀 hash router.
- **C.** 라우팅 없음 (window.location 분기).

[Answer]: A

### Q-FD-U5-2. 상태 관리

WS 메시지로 받은 상태 관리 도구?

- **A.** **React `useState` + `useReducer`만** — 외부 라이브러리 0. 단일 SPA + 작은 상태로 충분.
- **B.** Redux / Zustand.
- **C.** Context API + useReducer (전역 상태).

[Answer]: C

### Q-FD-U5-3. 토큰 저장

PlayerView 토큰 저장 위치?

- **A.** **`localStorage`** — 게임 종료까지 유지. (BR-U2-TOKEN 정책과 호환)
- **B.** sessionStorage (탭 단위).
- **C.** 쿠키.

[Answer]: A

### Q-FD-U5-4. WebSocket 자동 재연결

네트워크 끊김 시 클라이언트 동작?

- **A.** **자동 재연결 + 지수 백오프** (1s, 2s, 4s, 최대 16s). 토큰이 있으면 `resume` 메시지 자동 전송.
- **B.** 수동 재연결 버튼만.
- **C.** 자동 재연결, 백오프 없음 (즉시 재시도).

[Answer]: A

### Q-FD-U5-5. TTS 큐잉 정책 (FR-8.6)

`announce` 메시지를 어떻게 발화?

- **A.** **TTSQueue**: `enqueue(text)` → 순차 발화. `urgent` 플래그로 큐 클리어 + 즉시 발화. PhaseChanged + Eliminated 같은 단계 전환 이벤트는 `urgent`. 일반 INFO는 큐에 적층.
- **B.** 모든 메시지 즉시 발화 (인터럽트).
- **C.** 큐 없이 마지막 메시지만 발화.

[Answer]: A

### Q-FD-U5-6. TTS 한국어 음성 선택

한국어 음성 (ko-KR) 선택 정책?

- **A.** **시스템 한국어 음성 자동 선택** — `getVoices()` 중 `lang.startsWith("ko")` 첫 번째. 없으면 default voice + 한국어 발음 시도.
- **B.** 사용자 음성 선택 UI.
- **C.** 영어 음성 강제.

[Answer]: A

### Q-FD-U5-7. Web Speech 부재 폴백 (FR-8.7)

`window.speechSynthesis === undefined`이면?

- **A.** **자막만 표시 + 1회 토스트 안내 ("이 브라우저는 음성 안내를 지원하지 않습니다")**. 게임 진행은 정상.
- **B.** 페이지 로드 거부.
- **C.** 토스트 없이 자막만.

[Answer]: A

### Q-FD-U5-8. 호스트 컨트롤 노출 조건

호스트 패널 표시 조건은?

- **A.** **`/public` PublicView이면서 `joined.isHost === true`로 처음 진입한 클라이언트** — 호스트 PC 자체에서 host:create-session 호출. 다른 PUBLIC 클라이언트는 패널 없음.
- **B.** 모든 PUBLIC.
- **C.** 별도 `/host` 라우트.

[Answer]: A

### Q-FD-U5-9. PlayerView 단계별 UI

각 Phase에서 Player가 보는 화면?

- **A.** **단일 `<PlayerView>`가 Phase 분기**: LOBBY 대기, INTRO 자기소개 안내(자기 차례 강조), NIGHT 역할별 입력(마피아/의사/경찰만), DAY/VOTE 투표 폼, END 결과 화면.
- **B.** Phase별 별도 페이지 (라우트 추가).
- **C.** 단계 무관 동일 화면 (모든 입력 항상 보임).

[Answer]: A

### Q-FD-U5-10. 마피아 대표자 입력 동기화

마피아 대표자가 살해 대상을 변경하면 다른 마피아가 어떻게 알게?

- **A.** **`MafiaTargetSelected` 이벤트 (VisRoleMafia)** — 백엔드 U1이 매번 송출. 다른 마피아 클라이언트는 `state.pendingMafiaTarget`을 표시.
- **B.** 폴링.
- **C.** 마피아 클라이언트 간 P2P (X — 백엔드 단일 진실 소스).

[Answer]: A

### Q-FD-U5-11. 컴포넌트 라이브러리 / CSS

스타일링 도구?

- **A.** **CSS Modules + 직접 작성** — 외부 lib 0. 폰트는 system-ui/sans.
- **B.** Tailwind CSS.
- **C.** Material UI / shadcn.

[Answer]: A

### Q-FD-U5-12. 단위 테스트 / E2E

테스트 도구?

- **A.** **Vitest + @testing-library/react** — 컴포넌트 단위. E2E 미적용 (PoC).
- **B.** Jest + Playwright.
- **C.** 테스트 안 함.

[Answer]: A

### Q-FD-U5-13. wire 타입 동기화

백엔드 `protocol.go` 타입을 TS로 어떻게?

- **A.** **수동 동기화** — `web/src/types/wire.ts`에 직접 작성. 백엔드 변경 시 같이 수정. PoC + 작은 wire spec.
- **B.** OpenAPI / JSON Schema → 자동 codegen.
- **C.** TS 우선, Go zod-like 라이브러리.

[Answer]: A

### Q-FD-U5-14. ko-KR 텍스트 / i18n

한국어 텍스트 외부화?

- **A.** **인라인 한국어 (다국어 미지원)** — FR-8.3 한국어 한정.
- **B.** i18next + ko.json.
- **C.** 영어 + ko.

[Answer]: A

### Q-FD-U5-15. Vite outDir

Vite 빌드 출력 위치?

- **A.** **`vite.config.ts`의 `build.outDir = "../cmd/mafia-game/web/dist"`** — U4의 embed 위치와 일치 (U4 코드 변경 메모 그대로).
- **B.** `web/dist` (default) + 빌드 후 cp.
- **C.** root `dist/`.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `domain-entities.md` | 와이어 타입 TS 매핑 + UI 상태 모델 + 라우트 정의 + TTSQueue 인터페이스 |
| `business-logic-model.md` | useWebSocket 훅 의사 코드 + useTTSQueue 의사 코드 + 단계별 PlayerView 의사 코드 + 재연결 흐름 + 시퀀스 다이어그램 |
| `business-rules.md` | BR-U5-* 라우팅·TTS·재연결·마스킹·입력 규칙 ~50항목 + FR/NFR 추적성 |
| `frontend-components.md` | 컴포넌트 트리 + 각 컴포넌트 props/state + 인터랙션 흐름 + form validation |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주세요.
