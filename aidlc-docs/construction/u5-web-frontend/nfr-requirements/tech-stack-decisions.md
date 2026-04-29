# Tech Stack Decisions — U5 Web Frontend

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-requirements.md`, `requirements.md` v1.1 NFR-7

---

## 1. npm 직접 의존 (Q-NFR-U5-1=A)

### 1.1 Runtime

| 패키지 | 사용처 |
|---|---|
| `react` ^18 | 컴포넌트 |
| `react-dom` ^18 | 렌더링 |
| `react-router-dom` ^6 | `/public` ↔ `/play` 라우팅 |

### 1.2 Build / Tooling

| 패키지 | 사용처 |
|---|---|
| `vite` ^5 | 개발 서버 + 번들러 |
| `typescript` ^5 | TS 컴파일 (strict 모드) |
| `@types/react` ^18 | React 타입 정의 |
| `@types/react-dom` ^18 | React DOM 타입 |
| `@vitejs/plugin-react` ^4 | React Vite 플러그인 |

### 1.3 Test

| 패키지 | 사용처 |
|---|---|
| `vitest` | 테스트 러너 |
| `@testing-library/react` | 컴포넌트 단위 테스트 |
| `@testing-library/jest-dom` | DOM 매처 |
| `jsdom` | 테스트 환경 |

### 1.4 Lint

| 패키지 | 사용처 |
|---|---|
| `eslint` | 코드 린트 |
| `@typescript-eslint/parser` | TS 파서 |
| `@typescript-eslint/eslint-plugin` | TS 룰 |
| `eslint-plugin-react-hooks` | React hooks 룰 |

> 추가 lib (Tailwind, Redux, axios, lodash 등) 없음.

---

## 2. Web Speech API (브라우저 내장)

| API | 사용처 |
|---|---|
| `window.speechSynthesis` | TTS 발화 |
| `SpeechSynthesisUtterance` | 발화 객체 |
| `localStorage` | 토큰 영속화 |
| `WebSocket` | wire 통신 |

> 브라우저 내장 — npm 의존 0.

---

## 3. 프로젝트 / 파일 레이아웃 (확정)

```
web/
├── package.json
├── tsconfig.json
├── vite.config.ts
├── index.html
├── .eslintrc.cjs
├── src/
│   ├── main.tsx              # ReactDOM.createRoot
│   ├── App.tsx               # Router + GameProvider
│   ├── routes.tsx            # /public, /play
│   ├── context/
│   │   ├── GameContext.tsx   # Context + useReducer + 함수
│   │   └── reducer.ts        # gameReducer + applyEvent
│   ├── hooks/
│   │   ├── useWebSocket.ts   # 자동 재연결 + 지수 백오프
│   │   └── useTTSQueue.ts    # ko-KR + urgent/queue
│   ├── views/
│   │   ├── PublicView/
│   │   │   ├── PublicView.tsx
│   │   │   ├── PhaseHeader.tsx
│   │   │   ├── TimerBar.tsx
│   │   │   ├── PlayersGrid.tsx
│   │   │   ├── SubtitleArea.tsx
│   │   │   ├── HostControls.tsx
│   │   │   ├── VoiceToggle.tsx
│   │   │   └── PublicView.module.css
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
│   │       └── PlayerView.module.css
│   ├── components/
│   │   ├── ConnectionBadge.tsx
│   │   ├── NicknameForm.tsx
│   │   ├── PlayerPicker.tsx
│   │   └── ToastList.tsx
│   ├── styles/
│   │   └── global.css        # 색상 변수 + reset
│   ├── types/
│   │   └── wire.ts           # wire 타입 단일 진실 소스
│   └── tests/
│       └── (vitest 단위 테스트)
└── (Vite 빌드 산출물 → ../cmd/mafia-game/web/dist)
```

---

## 4. Vite 설정 (Q-FD-U5-15=A)

```ts
// web/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../cmd/mafia-game/web/dist",   // U4 embed 위치와 일치
    emptyOutDir: true,
    sourcemap: false,
  },
  server: {
    proxy: {
      "/ws":  { target: "ws://localhost:8080", ws: true },
      "/api": { target: "http://localhost:8080" },
    },
  },
});
```

---

## 5. TypeScript 설정 (Q-NFR-U5-4=A)

```jsonc
// web/tsconfig.json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noFallthroughCasesInSwitch": true,
    "isolatedModules": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "types": ["vite/client", "vitest/globals"]
  },
  "include": ["src"]
}
```

---

## 6. ESLint 설정 요약

```js
// web/.eslintrc.cjs
module.exports = {
  parser: "@typescript-eslint/parser",
  plugins: ["@typescript-eslint", "react-hooks"],
  extends: [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:react-hooks/recommended",
  ],
  parserOptions: { ecmaVersion: 2022, sourceType: "module" },
  rules: {
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn",
  },
};
```

---

## 7. 빌드 / 실행 가정

| 항목 | 결정 |
|---|---|
| Node 버전 | 20+ (Vite 5 요구) |
| 패키지 매니저 | `npm` (yarn/pnpm 미사용 — 단순성) |
| 개발 모드 | `cd web && npm run dev` (Vite proxy로 `/ws`, `/api` 백엔드로 forward) |
| 빌드 | `cd web && npm run build` → `../cmd/mafia-game/web/dist/` 생성 |
| 통합 빌드 | `cd web && npm run build && cd .. && go build ./cmd/mafia-game` |
| 빌드 산출물 | `cmd/mafia-game/web/dist/` (Go embed 동봉) |

---

## 8. 의존 그래프

```
web/src/main.tsx
  ├── react / react-dom
  └── App.tsx
       ├── react-router-dom
       └── GameProvider (Context)
            ├── useWebSocket → WebSocket API (browser)
            ├── useTTSQueue → SpeechSynthesis (browser)
            └── reducer + types/wire.ts
```

> 외부 npm lib는 react-router-dom 단 1개의 추가. 나머지는 react/vite/ts/types 인프라.

---

## 9. 미결정 / 후속 결정 사항

| 항목 | 결정 시점 |
|---|---|
| 다크 모드 / 테마 토글 | NFR 비-요구 (단일 다크 테마) |
| Service Worker (오프라인) | NFR 비-요구 |
| Bundle splitting / lazy load | 빌드 산출물 < 500 KB이면 단일 chunk 유지 |
| 접근성 자동 검증 (axe) | NFR 비-요구 |

---

## 10. 검증 체크리스트

- [x] npm 직접 의존 11종(runtime 3 + tooling 5 + test 4 + lint 4)만 사용
- [x] 패키지 레이아웃 정의 — `web/src/{context,hooks,views,components,styles,types,tests}`
- [x] Vite outDir = `../cmd/mafia-game/web/dist` (U4 embed 위치)
- [x] TS strict + noUncheckedIndexedAccess
- [x] 의존 그래프 — react-router-dom 단일 추가
- [x] 후속 결정 사항 명시
