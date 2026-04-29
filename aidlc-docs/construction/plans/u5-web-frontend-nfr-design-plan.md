# NFR Design Plan — U5 Web Frontend

**작성일**: 2026-04-26
**대상 단위**: U5 / Web Frontend (`web/src/*`)
**참조**: NFR Requirements (U5), Functional Design (U5)

> 본 plan은 U5 NFR Design의 단일 진실 소스입니다.

---

## 0. 목적

NFR Req의 한도(p99 < 100ms / 200ms / 500 KB / 70% / 32px 등)를 만족시키기 위한 React 패턴 + 컴포넌트 트리.

---

## 1. 단계 체크리스트

- [x] (1) NFR Req → 패턴 매핑
- [x] (2) 결정 질문 작성 (Q-NFRD-U5-1~7)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "승인" (권장 답안)
- [x] (5) 답변 일관성 검증
- [x] (6) `nfr-design-patterns.md` 작성
- [x] (7) `logical-components.md` 작성
- [x] (8) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-NFRD-U5-1 ~ Q-NFRD-U5-7)

### Q-NFRD-U5-1. Context 분리

GameContext를 단일 객체로 둘지, 분리할지?

- **A.** **단일 GameContextValue** — 모든 게임 상태 + dispatch + ws + tts. 작은 SPA에 적합.
- **B.** WSContext / GameStateContext / TTSContext 분리 — 리렌더 최적화.

[Answer]: A

### Q-NFRD-U5-2. WS 메시지 처리 위치

WS onMessage에서 직접 dispatch vs 큐?

- **A.** **직접 dispatch** — useReducer가 batch 처리, React 18의 자동 batching 활용.
- **B.** 별도 큐 + requestAnimationFrame.

[Answer]: A

### Q-NFRD-U5-3. PlayerPicker 리렌더 최적화

플레이어가 많으면 PlayerPicker 리렌더가 비싼가?

- **A.** **React.memo + key=player.id** — 12명이면 충분히 빠름. 복잡한 메모화 불필요.
- **B.** useMemo로 후보 계산.
- **C.** virtualization.

[Answer]: A

### Q-NFRD-U5-4. TTS 음성 사전 로드

브라우저 음성 목록 로드 시점은?

- **A.** **페이지 로드 시 `voiceschanged` 이벤트 + 즉시 호출** — Chrome은 비동기, Safari는 동기. 두 경우 모두 처리.
- **B.** 첫 발화 시점에만.
- **C.** 서비스 워커.

[Answer]: A

### Q-NFRD-U5-5. localStorage 접근

토큰 read/write 패턴?

- **A.** **useWebSocket 훅 안에서 직접 localStorage** — 간단. SSR 없으니 안전.
- **B.** 별도 useToken 훅.

[Answer]: B

### Q-NFRD-U5-6. CSS 변수 vs 인라인 스타일

severity 색상 적용 방법?

- **A.** **CSS Modules + data-severity 속성 + 매처 셀렉터** — 재사용 가능, 테마 변경 용이.
- **B.** 인라인 스타일.

[Answer]: A

### Q-NFRD-U5-7. 단위 테스트 패턴

핵심 모듈 테스트 도구는?

- **A.** **Vitest + Testing Library** + jsdom 환경. SpeechSynthesis는 mock.
- **B.** Jest + Enzyme.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `nfr-design-patterns.md` | P-U5-1~7 패턴 (단일 Context / direct dispatch / React.memo / voiceschanged / useToken / CSS Modules data-attr / Vitest + jsdom + mocks) |
| `logical-components.md` | LC-U5-1~N + 패키지 레이아웃 + 책임 매트릭스 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주세요.
