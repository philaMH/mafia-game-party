# Logical Components — U5 Web Frontend

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-design-patterns.md`, `tech-stack-decisions.md`, `functional-design/*.md`

---

## 1. 구성요소 카탈로그

| ID | 구성요소 | 위치 | 책임 | 적용 패턴 |
|---|---|---|---|---|
| LC-U5-1 | `App` | `src/App.tsx` | Router + GameProvider 와이어링 | — |
| LC-U5-2 | `GameProvider` + `useGameContext` | `src/context/GameContext.tsx` | 단일 Context + useReducer + ws/tts 와이어 | P-U5-1 |
| LC-U5-3 | `gameReducer` + `applyEvent` | `src/context/reducer.ts` | wire 메시지 → state 갱신 | — |
| LC-U5-4 | `useWebSocket` | `src/hooks/useWebSocket.ts` | 자동 재연결 + dispatch + 토큰 resume | P-U5-2, P-U5-5 |
| LC-U5-5 | `useTTSQueue` | `src/hooks/useTTSQueue.ts` | TTS 큐잉 + voiceschanged | P-U5-4 |
| LC-U5-6 | `useToken` | `src/hooks/useToken.ts` | localStorage 격리 | P-U5-5 |
| LC-U5-7 | `<PublicView>` 트리 | `src/views/PublicView/*` | 공용 화면 + 호스트 컨트롤 + TTS | P-U5-3, P-U5-6 |
| LC-U5-8 | `<PlayerView>` 트리 | `src/views/PlayerView/*` | 개인 화면 + 단계별 입력 | P-U5-3 |
| LC-U5-9 | 공통 컴포넌트 | `src/components/*` | NicknameForm / PlayerPicker / ConnectionBadge / ToastList | P-U5-3 |
| LC-U5-10 | wire 타입 | `src/types/wire.ts` | 단일 진실 소스 | — |
| LC-U5-11 | 글로벌 스타일 + 테마 | `src/styles/global.css` | CSS 변수 + reset | P-U5-6 |
| LC-U5-12 | 테스트 셋업 | `src/tests/setup.ts` | jsdom + SpeechSynthesis mock | P-U5-7 |

---

## 2. 패키지 / 파일 레이아웃 (확정)

```
web/
├── package.json
├── tsconfig.json
├── vite.config.ts
├── vitest.config.ts
├── .eslintrc.cjs
├── index.html
├── src/
│   ├── main.tsx                       # ReactDOM.createRoot
│   ├── App.tsx                        # LC-U5-1
│   ├── context/
│   │   ├── GameContext.tsx            # LC-U5-2
│   │   ├── reducer.ts                 # LC-U5-3
│   │   └── reducer.test.ts
│   ├── hooks/
│   │   ├── useWebSocket.ts            # LC-U5-4
│   │   ├── useWebSocket.test.ts
│   │   ├── useTTSQueue.ts             # LC-U5-5
│   │   ├── useTTSQueue.test.ts
│   │   ├── useToken.ts                # LC-U5-6
│   │   └── useToken.test.ts
│   ├── views/
│   │   ├── PublicView/
│   │   │   ├── PublicView.tsx
│   │   │   ├── PhaseHeader.tsx
│   │   │   ├── TimerBar.tsx
│   │   │   ├── PlayersGrid.tsx
│   │   │   ├── SubtitleArea.tsx
│   │   │   ├── HostControls.tsx
│   │   │   ├── VoiceToggle.tsx
│   │   │   ├── PublicView.module.css
│   │   │   └── PublicView.test.tsx
│   │   └── PlayerView/
│   │       ├── PlayerView.tsx
│   │       ├── PhaseInputs.tsx
│   │       ├── LobbyView.tsx
│   │       ├── IntroView.tsx
│   │       ├── NightInputs.tsx
│   │       ├── MafiaPicker.tsx
│   │       ├── DoctorPicker.tsx
│   │       ├── PolicePicker.tsx
│   │       ├── DiscussionView.tsx
│   │       ├── VoteForm.tsx
│   │       ├── EndScreen.tsx
│   │       ├── YourInfoCard.tsx
│   │       ├── PlayerView.module.css
│   │       └── PlayerView.test.tsx
│   ├── components/
│   │   ├── ConnectionBadge.tsx
│   │   ├── NicknameForm.tsx
│   │   ├── NicknameForm.test.tsx
│   │   ├── PlayerPicker.tsx
│   │   ├── ToastList.tsx
│   │   └── *.module.css
│   ├── styles/
│   │   └── global.css
│   ├── types/
│   │   └── wire.ts
│   └── tests/
│       └── setup.ts
└── (Vite outputs to ../cmd/mafia-game/web/dist)
```

---

## 3. 구성요소별 상세

### 3.1 LC-U5-2 GameProvider + useGameContext

```tsx
export interface GameContextValue {
  status: ConnectionStatus;
  clientId?: string;
  playerId?: PlayerID;
  token?: string;
  isHost: boolean;
  state?: State;
  your?: YourInfo;
  lastAnnounce?: { subtitle: string; severity: Severity; receivedAt: number };
  errors: { code: string; message: string }[];
  voiceOn: boolean;
  ttsAvailable: boolean;

  send: (msg: OutgoingMsg) => void;
  toggleVoice: (on: boolean) => void;
  ackError: (index: number) => void;
}
```

### 3.2 LC-U5-3 reducer

```ts
export const initialState: GameState = {
  status: "connecting",
  isHost: false,
  errors: [],
  voiceOn: true,
  ttsAvailable: typeof window !== "undefined" && !!window.speechSynthesis,
};

export function gameReducer(state: GameState, action: GameAction): GameState { /* ... */ }
function applyEvent(state: GameState, msg: { event: EventPayload; visibility: string }): GameState { /* ... */ }
```

### 3.3 LC-U5-4 useWebSocket

```ts
export interface UseWebSocketParams {
  url: string;
  dispatch: Dispatch<GameAction>;
  tokenIO: TokenIO;
}
export function useWebSocket(params: UseWebSocketParams): { send: (msg: OutgoingMsg) => void } {
  // ws lifecycle + 자동 재연결 + 지수 백오프 + onopen 시 token 있으면 resume
}
```

### 3.4 LC-U5-5 useTTSQueue

```ts
export interface TTSQueue {
  enqueue(text: string, opts?: TTSOpts): void;
  enqueueUrgent(text: string, opts?: TTSOpts): void;
  cancelAll(): void;
  available: boolean;
}
export function useTTSQueue(enabled: boolean): TTSQueue { /* ... */ }
```

### 3.5 LC-U5-6 useToken

```ts
export interface TokenIO {
  get(): string | null;
  set(token: string): void;
  clear(): void;
}
export function useToken(): TokenIO { /* ... */ }
```

### 3.6 LC-U5-9 공통 컴포넌트

| 컴포넌트 | Props 핵심 |
|---|---|
| `ConnectionBadge` | `status: ConnectionStatus` |
| `NicknameForm` | `prompt: string`, `onSubmit: (name) => void` |
| `PlayerPicker` | `players: Player[]`, `value?: PlayerID`, `disabled?: boolean`, `onChange: (id) => void` |
| `ToastList` | `errors[]`, `onDismiss: (i) => void` |

---

## 4. 책임 매트릭스 (NFR ↔ LC)

| NFR Req | 책임 LC |
|---|---|
| NFR-U5-P1 (DOM 갱신) | LC-U5-3 (reducer) + LC-U5-9 (memo PlayerPicker) |
| NFR-U5-P2 (TTS) | LC-U5-5 (useTTSQueue) |
| NFR-U5-P4 (빌드 < 500 KB) | 모든 LC + Vite 빌드 |
| NFR-U5-U1~U6 (Usability) | LC-U5-7 (PublicView), LC-U5-11 (CSS 변수) |
| NFR-U5-R1 (재연결) | LC-U5-4 (useWebSocket) |
| NFR-U5-R3 (TTS 폴백) | LC-U5-5 (available 분기) |
| NFR-U5-M1 (TS strict) | tsconfig.json |
| NFR-U5-M3 (커버리지) | LC-U5-12 (테스트 셋업) + 각 LC의 *.test.tsx |
| NFR-U5-M5 (wire 단일) | LC-U5-10 |
| NFR-U5-S1 (토큰 미노출) | LC-U5-6 (useToken 격리) |

---

## 5. Import 그래프

```
App
 ├─→ context/GameContext (Provider)
 │     ├─→ context/reducer
 │     ├─→ hooks/useWebSocket
 │     │     └─→ hooks/useToken
 │     └─→ hooks/useTTSQueue
 ├─→ views/PublicView/* (uses useGameContext)
 ├─→ views/PlayerView/* (uses useGameContext)
 └─→ components/* (uses useGameContext)

types/wire.ts ←─ 모든 곳 import
```

> 모든 import는 단방향. 컴포넌트가 hooks/context import. hooks/context는 컴포넌트 import 안 함.

---

## 6. 외부 인프라 / 의존

| 외부 | 사용처 |
|---|---|
| `react`, `react-dom`, `react-router-dom` | LC-U5-1, LC-U5-2 |
| `vite` (빌드 + dev server) | 모든 LC |
| `typescript` | 모든 .ts/.tsx |
| `vitest` + `@testing-library/react` | LC-U5-12 + 모든 *.test.* |
| 브라우저 API | LC-U5-4 (WebSocket), LC-U5-5 (SpeechSynthesis), LC-U5-6 (localStorage) |

---

## 7. 검증 체크리스트

- [x] 모든 LC 정확히 한 디렉터리에 위치
- [x] LC-U5-2 단일 Context 정책
- [x] LC-U5-6 useToken 격리 (NFR-U5-S1)
- [x] LC-U5-10 wire 타입 단일 진실 소스
- [x] NFR Req ↔ LC 매트릭스 모두 매핑
- [x] Import 그래프 단방향 (cycle 없음)
- [x] 외부 의존은 npm 11종 + 브라우저 API
