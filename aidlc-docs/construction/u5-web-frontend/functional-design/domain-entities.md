# Domain Entities — U5 Web Frontend

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: U3 wire protocol (`u3-public-api.md`), U2 announce (`u2-public-api.md`), U1 Engine (`u1-public-api.md`), `plans/u5-web-frontend-functional-design-plan.md`

본 문서는 U5의 도메인(타입 정의), 라우팅, UI 상태 모델, TTSQueue 인터페이스를 정의합니다.

---

## 1. 라우트 (Q-FD-U5-1=A, react-router-dom v6)

| 경로 | 컴포넌트 | 사용 대상 | 비고 |
|---|---|---|---|
| `/` | `<App>` | 진입 — `/play`로 redirect (기본) | 사용자가 host PC면 `/public` 수동 진입 |
| `/public` | `<PublicView>` | 호스트 PC 자체 + 관전자 모니터 | TTS + 자막 + 호스트 컨트롤 |
| `/play` | `<PlayerView>` | 플레이어 디바이스 (모바일/태블릿) | 닉네임 입장 + 입력 폼 |

> `/play/*` 하위 경로 없음 — 단일 PlayerView가 Phase 분기 (Q-FD-U5-9=A).

---

## 2. wire 타입 (TS 매핑) — `web/src/types/wire.ts`

U3의 Go wire 타입을 TypeScript로 수동 동기화 (Q-FD-U5-13=A).

```ts
// 기본 타입
export type PlayerID = string;
export type Role = "MAFIA" | "CITIZEN" | "DOCTOR" | "POLICE";
export type Team = "MAFIA" | "CITIZEN";
export type Phase = "LOBBY" | "INTRO" | "NIGHT" | "DAY" | "VOTE" | "RECOUNT" | "END";
export type EndReason = "MAFIA_WIN" | "CITIZEN_WIN" | "HOST_FORCE_END";

export interface Player {
  id: PlayerID;
  name: string;
  alive: boolean;
  role?: Role;       // 자기 자신만 채워짐 (마스킹)
  keyword?: string;
}

export interface Options {
  mafiaCount: number;
  introSecondsPerPlayer: number;
  discussionSeconds: number;
  doctorSelfHealAllowed: boolean;
  announcementVoiceOn: boolean;
}

export interface State {
  gameId: string;
  phase: Phase;
  day: number;
  players: Player[];
  hostId: PlayerID;
  settings: Options;
  startedAt?: string;
  deadline?: string;          // ISO 8601
  introSpeakerIdx?: number;
  pendingMafiaTarget?: PlayerID;
  pendingDoctorTarget?: PlayerID;
  pendingPoliceTarget?: PlayerID;
  votes?: Record<PlayerID, PlayerID>;
  voteRound?: number;
  voteCandidates?: PlayerID[];
  winner?: Team;
  endReason?: EndReason;
}
```

### 2.1 Incoming wire 메시지 (server → client)

```ts
export type IncomingMsg =
  | { type: "welcome"; clientId: string; kind: "PUBLIC" | "PLAYER"; protocolVersion: string }
  | { type: "joined"; playerId: PlayerID; token: string; isHost: boolean }
  | { type: "snapshot"; state: State; your: YourInfo; isHost: boolean }
  | { type: "event"; visibility: "PUBLIC" | "PLAYER" | "ROLE_MAFIA"; event: EventPayload }
  | { type: "announce"; subtitle: string; speech: string; severity: "INFO" | "EMPHASIS" | "WARN" }
  | { type: "error"; code: string; message: string };

export interface YourInfo {
  role?: Role;
  keyword?: string;
  team?: Team;
  mafiaCohort?: PlayerID[];
}
```

### 2.2 EventPayload (15 kind)

```ts
export type EventPayload =
  | { kind: "GameStarted" }
  | { kind: "PhaseChanged"; phase: Phase; day: number; deadlineMs: number }
  | { kind: "RoleRevealedToPlayer"; playerId: PlayerID; role: Role; keyword: string }
  | { kind: "MafiaCohortRevealed"; mafiaIds: PlayerID[]; representativeId: PlayerID }
  | { kind: "IntroSpeakerChanged"; playerId: PlayerID; secondsLeft: number }
  | { kind: "MafiaTargetSelected"; representativeId: PlayerID; target: PlayerID }
  | { kind: "PoliceResult"; police: PlayerID; target: PlayerID; team: Team }
  | { kind: "DeathAnnounced"; victim: PlayerID }
  | { kind: "PeacefulNight" }
  | { kind: "DiscussionTimerTick"; secondsLeft: number }
  | { kind: "VoteTallied"; counts: Record<PlayerID, number>; eliminated?: PlayerID; recount: boolean }
  | { kind: "Eliminated"; playerId: PlayerID; role: Role }
  | { kind: "MafiaRepresentativeReassigned"; oldId: PlayerID; newId: PlayerID }
  | { kind: "GameEnded"; winner?: Team; endReason: EndReason; reveal: Player[] }
  | { kind: "VoiceToggled"; on: boolean };
```

### 2.3 Outgoing wire 메시지 (client → server)

```ts
export type OutgoingMsg =
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

## 3. UI 상태 모델 (Q-FD-U5-2=C — Context + useReducer)

### 3.1 GameContext (전역 상태)

```ts
export interface GameContextValue {
  // WebSocket connection
  status: "connecting" | "connected" | "reconnecting" | "closed";
  clientId?: string;

  // 식별
  playerId?: PlayerID;
  token?: string;
  isHost: boolean;

  // 게임 상태
  state?: State;
  your?: YourInfo;

  // 안내
  lastAnnounce?: { subtitle: string; severity: string; receivedAt: number };
  errors: { code: string; message: string }[];

  // TTS
  voiceOn: boolean;
  ttsAvailable: boolean;

  // Actions (dispatch wrappers)
  send: (msg: OutgoingMsg) => void;
  toggleVoice: (on: boolean) => void;
}
```

### 3.2 Reducer Action

```ts
type GameAction =
  | { type: "ws_open"; clientId?: string }
  | { type: "ws_message"; msg: IncomingMsg }
  | { type: "ws_reconnecting" }
  | { type: "ws_closed" }
  | { type: "set_voice"; on: boolean }
  | { type: "tts_unavailable" }
  | { type: "ack_error"; index: number };
```

### 3.3 reducer 정책 (요약)

| Action | State 변경 |
|---|---|
| `ws_open` | status="connected", clientId 설정 |
| `ws_message: welcome` | clientId 갱신 |
| `ws_message: joined` | playerId/token/isHost 설정. localStorage에 token 저장 |
| `ws_message: snapshot` | state/your 갱신 |
| `ws_message: event` | event.kind에 따라 state 부분 갱신 (PhaseChanged, Eliminated 등) |
| `ws_message: announce` | lastAnnounce 갱신 + TTSQueue.enqueue |
| `ws_message: error` | errors[] 추가 |
| `ws_reconnecting` | status="reconnecting" |
| `ws_closed` | status="closed" |
| `set_voice` | voiceOn 토글 + 백엔드에 host:toggle-voice 송신 |
| `tts_unavailable` | ttsAvailable=false |

---

## 4. TTSQueue 인터페이스 (Q-FD-U5-5=A)

```ts
export interface TTSQueue {
  // 일반 안내 — 큐 뒤에 적층
  enqueue(text: string, opts?: TTSOpts): void;

  // 단계 전환·사망 등 긴급 — 큐 클리어 후 즉시 발화
  enqueueUrgent(text: string, opts?: TTSOpts): void;

  // 사용자 토글
  setEnabled(on: boolean): void;

  // 외부 종료 (페이지 unload, 음성 OFF 시 즉시 중단)
  cancelAll(): void;
}

export interface TTSOpts {
  lang?: string;     // 기본 "ko-KR"
  pitch?: number;    // 기본 0.9 (근엄 톤)
  rate?: number;     // 기본 0.95
  volume?: number;   // 기본 1.0
}
```

### 4.1 어떤 announce를 urgent로 처리?

| 트리거 이벤트 | 분류 | 근거 |
|---|---|---|
| PhaseChanged{NIGHT/DAY/VOTE/RECOUNT} | **urgent** | 단계 전환은 즉시 알려야 함 |
| Eliminated | **urgent** | 처형 결과 즉시 통지 |
| DeathAnnounced | **urgent** | 새벽 사망 |
| GameEnded | **urgent** | 게임 결과 |
| IntroSpeakerChanged | normal (queue) | 자기소개 차례 안내 — 끊지 않음 |
| DiscussionTimerTick (30/10/0) | normal | 시간 알림은 긴급 아님 |
| PeacefulNight | normal | INFO |
| VoteTallied recount | normal (WARN) | 짧고 다음 단계 유발 안 함 |

> 분류는 `business-logic-model.md`의 `dispatchAnnounce` 함수가 결정.

---

## 5. 컴포넌트 트리 개관

```
<App>
├── <Router>
│   ├── Route "/"        → <Navigate to="/play" />
│   ├── Route "/public"  → <PublicView />
│   └── Route "/play"    → <PlayerView />
└── <GameProvider>      // Context + useReducer + WS hook
```

### 5.1 PublicView 하위

```
<PublicView>
├── <ConnectionBadge>           // status indicator
├── <PhaseHeader phase day />
├── <TimerBar deadline />        // DAY 토론 + INTRO 자기소개
├── <Players players />          // 큰 텍스트 그리드
├── <SubtitleArea />             // 마지막 announce.subtitle, severity 색상
├── <HostControls />             // isHost일 때만 표시
└── <VoiceToggle voiceOn />      // FR-8.5
```

### 5.2 PlayerView 하위

```
<PlayerView>
├── <ConnectionBadge>
├── <NicknameForm />              // 미입장 시
├── <YourInfo role keyword team />// 자기 정보 카드 (마피아면 cohort 표시)
└── <PhaseInputs phase your players /> // Phase에 따라:
    ├── <LobbyView />             // 입장 대기
    ├── <IntroView />             // 자기 차례 강조
    ├── <NightInputs />           // 마피아/의사/경찰 폼
    ├── <DiscussionView />        // 토론 중 (입력 없음)
    ├── <VoteForm />              // VOTE/RECOUNT
    └── <EndScreen />             // GameEnded 결과
```

---

## 6. 검증 체크리스트

- [x] 라우트 3종 + redirect 정책 명시
- [x] wire 타입 IncomingMsg/OutgoingMsg/EventPayload TS 매핑 완료
- [x] State 마스킹 정책 wire 타입에 반영 (Player.role optional)
- [x] GameContextValue + GameAction reducer 명세
- [x] TTSQueue 인터페이스 + urgent 분류 표
- [x] 컴포넌트 트리 PublicView/PlayerView 분리
