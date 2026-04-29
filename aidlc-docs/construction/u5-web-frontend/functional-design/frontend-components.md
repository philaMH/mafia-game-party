# Frontend Components — U5 Web Frontend

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `domain-entities.md`, `business-logic-model.md`, `business-rules.md`

본 문서는 React 컴포넌트 트리, 각 컴포넌트의 props/state, 인터랙션 흐름, form validation을 정의합니다.

---

## 1. 컴포넌트 트리 (전체)

```
<App>
└── <GameProvider>                  // Context + useReducer + useWebSocket + useTTSQueue
    └── <BrowserRouter>
        └── <Routes>
            ├── Route "/"        → <Navigate to="/play" />
            ├── Route "/public"  → <PublicView>
            │   ├── <ConnectionBadge>
            │   ├── <NicknameForm onSubmit={createSession} />     # 호스트 미생성 시
            │   ├── <PhaseHeader>
            │   ├── <TimerBar>
            │   ├── <PlayersGrid>
            │   ├── <SubtitleArea>
            │   ├── <HostControls>                                  # isHost일 때만
            │   ├── <VoiceToggle>
            │   └── <ToastList>
            └── Route "/play"    → <PlayerView>
                ├── <ConnectionBadge>
                ├── <NicknameForm onSubmit={joinPlayer} />          # 미식별 시
                ├── <YourInfoCard your={your} />
                └── <PhaseInputs>                                   # Phase 분기
                    ├── <LobbyView />
                    ├── <IntroView />
                    ├── <NightInputs>
                    │   ├── <MafiaPicker />
                    │   ├── <DoctorPicker />
                    │   ├── <PolicePicker />
                    │   └── <CitizenWaiting />
                    ├── <DiscussionView />
                    ├── <VoteForm />
                    └── <EndScreen />
```

---

## 2. 공통 컴포넌트

### 2.1 `<ConnectionBadge>`

```ts
interface Props {
  status: "connecting" | "connected" | "reconnecting" | "closed";
}
```

| status | 표시 |
|---|---|
| connecting | "🔄 연결 중…" (회색) |
| connected | "🟢 연결됨" (녹색, 3초 후 fade out) |
| reconnecting | "⚠️ 재연결 중…" (주황) |
| closed | "🔴 연결 끊김" (빨강) |

### 2.2 `<NicknameForm>`

```ts
interface Props {
  prompt: string;             // "호스트 닉네임" 또는 "닉네임"
  onSubmit: (name: string) => void;
}
interface State {
  name: string;
  error?: string;
}
```

**검증 규칙** (BR-U5-INPUT-1):
- 1~20자
- 한글/영문/숫자 + 일부 특수문자 (`-`, `_`, 공백)
- 위반 시 `error="닉네임은 1~20자 한글/영문/숫자입니다"`

```ts
function validateName(s: string): string | undefined {
  const t = s.trim();
  if (t.length === 0) return "닉네임을 입력하세요";
  if (t.length > 20) return "닉네임은 20자 이하입니다";
  if (!/^[가-힣a-zA-Z0-9 _-]+$/.test(t)) return "허용되지 않는 문자가 있습니다";
  return undefined;
}
```

### 2.3 `<ToastList>`

```ts
interface Props {
  errors: { code: string; message: string; expiresAt: number }[];
  onDismiss: (index: number) => void;
}
```

5초 후 자동 사라짐 (BR-U5-ERR-2).

### 2.4 `<PlayerPicker>`

```ts
interface Props {
  players: Player[];
  value?: PlayerID;
  disabled?: boolean;
  onChange: (id: PlayerID) => void;
}
```

라디오 그룹. 살아있는 플레이어만 표시. disabled면 회색 + 클릭 차단.

---

## 3. PublicView 컴포넌트

### 3.1 `<PublicView>`

State는 GameContext에서. 직접 useState 없음.

**렌더링 분기**:
- `state === undefined` && `isHost === false` → "호스트 PC가 게임을 생성하기를 기다리는 중…"
- `state === undefined` && first PUBLIC client → `<NicknameForm onSubmit={createSession} prompt="호스트 닉네임" />`
- `state.phase === "LOBBY"` → 입장한 플레이어 목록 + HostControls
- 그 외 → 풀 화면 (PhaseHeader + TimerBar + PlayersGrid + SubtitleArea)

### 3.2 `<PhaseHeader>`

```ts
interface Props {
  phase: Phase;
  day: number;
}
```

| phase | 텍스트 |
|---|---|
| LOBBY | "참가자 모집 중" |
| INTRO | "{day}일째 — 자기소개" |
| NIGHT | "{day}일째 — 밤" |
| DAY | "{day}일째 — 낮" |
| VOTE | "{day}일째 — 투표" |
| RECOUNT | "{day}일째 — 재투표" |
| END | "게임 종료" |

폰트 ≥ 48px (BR-U5-PUBLIC-3).

### 3.3 `<TimerBar>`

```ts
interface Props {
  deadline?: string;          // ISO 8601
}
```

- `deadline` 없으면 비표시
- 1초마다 갱신 — `setInterval(1000)`
- 남은 초 표시 + 진행률 바
- `secondsLeft <= 10`이면 빨강 색상

### 3.4 `<PlayersGrid>`

```ts
interface Props {
  players: Player[];
  pendingMafiaTarget?: PlayerID;     // PublicView에는 보이지 않음 (마스킹)
}
```

플레이어 카드 그리드. 카드:
- 큰 닉네임 (≥ 32px)
- ALIVE: 컬러 보더 / DEAD: 회색 + X 표식 (BR-U5-PUBLIC-4)
- Phase=END에서는 reveal[*].role도 표시

### 3.5 `<SubtitleArea>`

```ts
interface Props {
  ann?: { subtitle: string; severity: string; receivedAt: number };
}
```

- 화면 중앙 큰 자막 (≥ 32px)
- severity별 색상: INFO 흰색 / EMPHASIS 주황 / WARN 빨강
- 5초 동안 표시 후 자동 fade

### 3.6 `<HostControls>`

```ts
interface Props {
  state: State;
  send: (msg: OutgoingMsg) => void;
  voiceOn: boolean;
}
```

Phase별 버튼 (BR-U5-HOST-1~5):
- LOBBY: "게임 시작" (≥ 6명 활성)
- INTRO: "다음 발언자"
- NIGHT: "야간 마감"
- DAY: "토론 조기 종료"
- 모든 단계: "강제 종료" (확인 다이얼로그) + 음성 토글

### 3.7 `<VoiceToggle>`

```ts
interface Props {
  on: boolean;
  onChange: (on: boolean) => void;
  available: boolean;
}
```

- `available === false`이면 disabled + 툴팁 ("이 브라우저는 음성 안내를 지원하지 않습니다")
- 토글 변경 시 `host:toggle-voice` 송신

---

## 4. PlayerView 컴포넌트

### 4.1 `<PlayerView>`

```
const ctx = useGameContext()
if (!ctx.playerId) return <NicknameForm prompt="닉네임" onSubmit={(n) => ctx.send({type:"join", name: n})} />
return (
  <>
    <ConnectionBadge status={ctx.status} />
    <YourInfoCard your={ctx.your} />
    <PhaseInputs />
    <ToastList errors={ctx.errors} />
  </>
)
```

### 4.2 `<YourInfoCard>`

```ts
interface Props {
  your?: YourInfo;
}
```

- `your.role` 표시 ("당신의 역할: 마피아")
- `your.keyword` 표시 ("키워드: 정의")
- `your.role === "MAFIA"`이면 `mafiaCohort` 닉네임 목록도 표시 ("동료 마피아: 영희, 철수")
- 플레이어가 사망했으면 카드에 "사망" 워터마크

### 4.3 `<PhaseInputs>` (Phase 분기 컨테이너)

```
switch state.phase:
  case "LOBBY": return <LobbyView players={state.players} />
  case "INTRO": return <IntroView state={state} me={ctx.playerId} />
  case "NIGHT": return <NightInputs state={state} your={ctx.your} send={ctx.send} />
  case "DAY":
  case "RECOUNT": // RECOUNT는 사실 VOTE 형태, but presents differently
    return <DiscussionView state={state} />
  case "VOTE":
  case "RECOUNT":
    return <VoteForm state={state} me={ctx.playerId} send={ctx.send} />
  case "END":
    return <EndScreen state={state} />
```

### 4.4 `<LobbyView>`

```ts
interface Props {
  players: Player[];
  isHost: boolean;
}
```

- 입장한 플레이어 목록 (이름만)
- "호스트가 게임을 시작하길 기다리는 중…" 메시지
- isHost면 (`/public`에서 입장한 호스트의 `/play` 동시 사용 케이스) 추가 안내

### 4.5 `<IntroView>`

```ts
interface Props {
  state: State;
  me: PlayerID;
}
```

- 현재 발언자(`state.players[state.introSpeakerIdx]`) 강조
- 본인 차례면 "지금 자기소개를 시작하세요. 키워드: {your.keyword}" 큰 메시지
- 다른 플레이어 차례면 "{name}이(가) 자기소개 중입니다."

### 4.6 `<NightInputs>` (역할 분기)

```ts
interface Props {
  state: State;
  your: YourInfo;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}
```

```
const me = state.players.find(p => p.id === ctx.playerId)
if (!me?.alive) return <p>당신은 사망했습니다. 야간 진행을 관전하시오.</p>

switch your.role:
  case "MAFIA":     return <MafiaPicker state={state} your={your} send={send} />
  case "DOCTOR":    return <DoctorPicker state={state} me={me} send={send} />
  case "POLICE":    return <PolicePicker state={state} me={me} send={send} />
  case "CITIZEN":   return <CitizenWaiting />
```

### 4.7 `<MafiaPicker>` (Q-AD-7, BR-U5-PLAYER-5)

```ts
interface Props {
  state: State;
  your: YourInfo;
  send: (msg: OutgoingMsg) => void;
}
```

- 후보: `state.players.filter(p => p.alive && !your.mafiaCohort.includes(p.id))`
- 활성화 조건: `state.mafiaRepresentativeId === your.playerId`
- 비대표자는 비활성 + 현재 선택 표시 ("대표자가 선택한 대상: {pendingMafiaTarget}")
- 선택 시 `send({type:"submit:mafia-kill", target})`

### 4.8 `<DoctorPicker>`

```ts
interface Props {
  state: State;
  me: Player;
  send: (msg: OutgoingMsg) => void;
}
```

- 후보: 살아있는 모든 플레이어. `state.settings.doctorSelfHealAllowed === true`이면 자기 ID 포함, 아니면 제외
- 선택 시 `send({type:"submit:doctor-heal", target})`

### 4.9 `<PolicePicker>`

```ts
interface Props {
  state: State;
  me: Player;
  send: (msg: OutgoingMsg) => void;
}
```

- 후보: 살아있는 플레이어 - 자기 자신
- `state.policeCheckedThisNight === true`이면 disabled + "이번 밤 조사 완료"
- 마지막 결과는 `<YourInfoCard>` 추가 영역에 표시 (`lastPoliceResult`)

### 4.10 `<DiscussionView>`

```ts
interface Props {
  state: State;
}
```

- DAY 단계 토론 중 — 입력 없음
- 남은 시간 표시 (`<TimerBar>` 재사용)

### 4.11 `<VoteForm>`

```ts
interface Props {
  state: State;
  me: PlayerID;
  send: (msg: OutgoingMsg) => void;
}
```

- 후보: 살아있는 플레이어 - 자기 자신
- 선택 시 `send({type:"submit:vote", target})` 즉시
- 본인 표 변경 가능 (last-write-wins, BR-U5-INPUT-3)
- 표 집계는 백엔드 — 클라이언트에는 `VoteTallied` 이벤트 도착 시 표시

### 4.12 `<EndScreen>`

```ts
interface Props {
  state: State;
}
```

- "{winner}의 승리" 큰 헤더
- `state.players` 전부 — 닉네임 + Role 공개 (BR-U5-MASK-4)
- "다시 시작" 버튼 (호스트면 `host:create-session` 재호출)

---

## 5. 인터랙션 시퀀스

### 5.1 PUBLIC 호스트 입장 + 게임 시작

```
1. /public 진입 → WebSocket connect → welcome
2. NicknameForm 입력 → host:create-session{name}
3. joined{isHost: true} → HostControls 노출
4. 다른 PLAYER 5명 입장 (별도 디바이스)
5. "게임 시작" 클릭 → host:start{options}
6. event(GameStarted, PhaseChanged INTRO, IntroSpeakerChanged)
7. announce("마피아 게임이 시작됩니다…", EMPHASIS) → TTSQueue.enqueueUrgent
8. SubtitleArea 표시 + TTS 발화
```

### 5.2 PLAYER 닉네임 입력 + 야간 입력

```
1. /play 진입 → WebSocket connect → welcome
2. NicknameForm → join{name}
3. joined{playerId, token, isHost: false} → localStorage 저장
4. snapshot 미수신 (LOBBY 단계는 snapshot 없음 — joined 응답만)
5. 호스트가 게임 시작 → snapshot? No, join은 LOBBY 단계만 가능
   실제로는: PLAYER가 join하면 LOBBY 화면. 호스트가 host:start 하면
   백엔드가 모든 클라이언트에게 event 송출 → reducer가 자동 갱신
6. event(RoleRevealedToPlayer{role: "DOCTOR", keyword: "신뢰"})
7. YourInfoCard 갱신: 의사 / 키워드 신뢰
8. event(PhaseChanged{NIGHT}) → PhaseInputs가 NightInputs로 전환
9. DoctorPicker 표시 → 보호 대상 선택 → submit:doctor-heal
10. 호스트가 야간 마감 → resolve → event(DeathAnnounced 또는 PeacefulNight)
```

### 5.3 PLAYER 재연결

```
1. 페이지 리로드 → WebSocket connect → welcome
2. localStorage.token 발견 → resume{token} 자동 송신
3. joined{playerId, isHost: false}
4. snapshot{state, your: {role, keyword, team, mafiaCohort}}
5. PhaseInputs가 즉시 현재 phase에 맞는 입력 폼 렌더 (FR-1.2)
```

---

## 6. State 흐름 요약 (UI 데이터 흐름)

```
[Backend WS message] → [useWebSocket onMessage]
  → [dispatch({type: "ws_message", msg})]
  → [reducer: applyEvent or 간단 갱신]
  → [GameContextValue 업데이트]
  → [모든 useGameContext() 컴포넌트 자동 리렌더]
  → [PublicView이면 lastAnnounce 변경 effect → TTSQueue.enqueue]
```

---

## 7. CSS 모듈 구성 (Q-FD-U5-11=A)

```
web/src/styles/
├── global.css                # 색상 변수 + 폰트 + reset
├── PublicView.module.css
├── PlayerView.module.css
├── components/
│   ├── HostControls.module.css
│   ├── PlayerPicker.module.css
│   └── ...
```

**색상 변수**:
```css
:root {
  --bg: #0e0e10;
  --fg: #e5e5e5;
  --info: #cccccc;
  --emphasis: #f59e0b;
  --warn: #ef4444;
  --alive: #22c55e;
  --dead: #6b7280;
}
```

---

## 8. 검증 체크리스트

- [x] 컴포넌트 트리 PublicView/PlayerView 분리
- [x] 모든 컴포넌트 props/state 인터페이스 정의
- [x] NicknameForm 검증 로직 (1~20자, 허용 문자)
- [x] PlayerView Phase 분기 (LOBBY/INTRO/NIGHT/DAY/VOTE/RECOUNT/END)
- [x] NightInputs 역할 분기 (MAFIA/DOCTOR/POLICE/CITIZEN)
- [x] 마피아 대표자 권한 분기 (MafiaPicker)
- [x] HostControls 단계별 버튼 매트릭스
- [x] 인터랙션 시퀀스 3종 (호스트 게임 시작, PLAYER 야간 입력, 재연결)
- [x] CSS Modules 구성
