# U5 — Public API Catalog

**작성일**: 2026-04-26
**대상 패키지**: `web/src/*` (mafia-game-web)
**버전**: 1.0 (Code Generation 1차 산출물)

본 문서는 U5가 외부(브라우저, 사용자)에 노출하는 라우트, wire 프로토콜, CLI 빌드 명령, 통합 진입점을 요약합니다.

---

## 1. 라우트

| 경로 | 컴포넌트 | 사용 대상 |
|---|---|---|
| `/` | redirect → `/play` | 진입 |
| `/public` | `<PublicView>` | 호스트 PC, 관전자 모니터 (TTS + 호스트 컨트롤) |
| `/play` | `<PlayerView>` | 플레이어 디바이스 (모바일/태블릿) |
| `/*` | redirect → `/play` | unknown path |

---

## 2. 진입점 (브라우저)

```
http://<LAN-IP>:8080/public      # 호스트 PC
http://<LAN-IP>:8080/play        # 플레이어
```

호스트가 처음 접속 시 닉네임 입력 → `host:create-session` → 호스트 권한 자동 획득.

---

## 3. wire 프로토콜 (브라우저 ↔ 서버)

`src/types/wire.ts`가 단일 진실 소스. U3의 `protocol.go`와 lockstep.

### 3.1 Incoming (server → client)

```ts
type IncomingMsg =
  | WelcomeMsg
  | JoinedMsg
  | SnapshotMsg
  | EventMsg          // 15 event kinds: GameStarted, PhaseChanged, ...
  | AnnounceMsg       // 한국어 자막 + speech + severity
  | ErrorMsg;
```

### 3.2 Outgoing (client → server)

```ts
type OutgoingMsg =
  | { type: "host:create-session"; name: string }
  | { type: "join"; name: string }
  | { type: "resume"; token: string }
  | { type: "host:start"; options: Options }
  | { type: "submit:advance-intro" }
  | { type: "submit:mafia-kill"; target: PlayerID }
  | { type: "submit:doctor-heal"; target: PlayerID }
  | { type: "submit:police-check"; target: PlayerID }
  | { type: "submit:end-night" }
  | { type: "submit:end-discussion" }
  | { type: "submit:vote"; target: PlayerID }
  | { type: "host:toggle-voice"; on: boolean }
  | { type: "host:force-end" }
  | { type: "subscribe-public" };
```

---

## 4. localStorage 키

| 키 | 용도 |
|---|---|
| `mafia.token` | PlayerView 재연결 토큰 (32바이트 hex) |

> 다른 키 미사용. `useToken` 훅이 단일 진실 소스.

---

## 5. CLI / 빌드 명령

| 명령 | 책임 |
|---|---|
| `npm install` | 의존 11종 설치 (~ 100~200 MB) |
| `npm run dev` | Vite 개발 서버 (port 5173 기본, `/ws`/`/api` proxy 백엔드 8080으로) |
| `npm run typecheck` | tsc --noEmit (TS strict 검증) |
| `npm run lint` | eslint (TS + react-hooks 룰) |
| `npm test` | vitest run |
| `npm run test:coverage` | vitest run --coverage (핵심 모듈 ≥ 70%) |
| `npm run build` | tsc --noEmit + vite build → `../cmd/mafia-game/web/dist/` |

---

## 6. 통합 빌드

```bash
cd web && npm install && npm run build
cd .. && go build ./cmd/mafia-game
./mafia-game
```

→ 단일 바이너리 (Mach-O 64-bit / ELF, ~16 MB)

---

## 7. 보장 / 계약

| 항목 | 보장 |
|---|---|
| 토큰 미노출 (NFR-U5-S1) | `members[].token`은 wire 응답에 없고, `localStorage.mafia.token`만 존재. 화면/로그에 미표시 |
| TTS 부재 시 자막 폴백 (FR-8.7) | `window.speechSynthesis === undefined` 시 토스트 + 자막만 |
| 자동 재연결 (FR-1.2) | WebSocket onclose 후 1/2/4/8/16s 백오프, 16s 후 무한 재시도 |
| 마피아 대표자만 입력 활성 (FR-4.3) | MafiaPicker가 `state.mafiaRepresentativeId !== me`이면 disabled |
| 의사 자가 보호 (FR-4.4) | DoctorPicker가 `state.settings.doctorSelfHealAllowed`로 자기 ID 포함 여부 결정 |
| 단계 전환 즉시 발화 (FR-8.6) | URGENT_KINDS = {PhaseChanged, Eliminated, DeathAnnounced, GameEnded} → enqueueUrgent |
| 모바일 320px 반응형 | PlayerView max-width 32rem, PlayerPicker flex-wrap |
| TS strict | `noUncheckedIndexedAccess` + `noImplicitOverride` |

---

## 8. 변경 영향도

- 새 wire 메시지 타입 추가는 안전 (백엔드와 동시 변경 필요).
- `members[].token` 같은 비공개 필드 추가는 NFR-U5-S1 위반 — 절대 노출 금지.
- 라우트 추가는 안전 (기존 fallback이 unknown path를 `/play`로 보냄).
- React 18 → 19 등 메이저 업그레이드 시 GameProvider effect 의존 배열 검토 필요.

---

## 9. 후속 작업

- U5는 마지막 단위 — 다음은 Build & Test (모든 단위 통합 검증).
- 운영 단계에서 `cd web && npm run build`를 CI에 내장 권장.
