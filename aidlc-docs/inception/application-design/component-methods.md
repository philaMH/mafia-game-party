# Component Methods — Mafia Game

**작성일**: 2026-04-25
**범위**: 컴포넌트별 메서드 시그니처 (Application Design 수준)
**비고**: 비즈니스 규칙 디테일(예: 동률 처리 알고리즘, 키워드 동률 무작위 시드)은 Functional Design 단계에서 확정.

표기 규약: Go 의사 코드 — 실제 구현 시 패키지 경로/타입은 조정 가능.

---

## 공용 타입 (도메인)

```go
type PlayerID string
type Role     string  // "MAFIA" | "CITIZEN" | "DOCTOR" | "POLICE"
type Phase    string  // "LOBBY" | "INTRO" | "NIGHT" | "DAY" | "VOTE" | "RECOUNT" | "END"
type Team     string  // "MAFIA" | "CITIZEN"

type Player struct {
    ID    PlayerID
    Name  string
    Alive bool
    Role  Role     // 비공개
    Keyword string // 비공개
}

type State struct {
    GameID    string
    Phase     Phase
    Day       int
    Players   []Player
    Deadline  time.Time      // 현재 단계 종료 예정 시각 (있는 경우)
    Pending   PendingActions // 미적용 야간 행동, 투표 등
    HostID    PlayerID       // Q-AD-6=B: 호스트도 플레이어
    Settings  Options
    StartedAt time.Time
}

type Options struct {
    IntroSecondsPerPlayer int  // Q-AD-4=B: 자기소개 1인당 초
    DiscussionSeconds     int  // Q-AD-5: 기본 토론 시간
    DoctorSelfHealAllowed bool // Q-AD-8=A: true
    AnnouncementVoiceOn   bool // FR-8.5 기본값
}

type Action interface{} // sealed via type switch
type StartGame struct{ HostID PlayerID; Options Options }
type AdvanceIntro struct{ HostID PlayerID } // 자동 진행이므로 거의 사용 안 함, 호스트 강제 진행만
type SubmitMafiaKill struct{ Mafia PlayerID; Target PlayerID }
type SubmitDoctorHeal struct{ Doctor PlayerID; Target PlayerID }
type SubmitPoliceCheck struct{ Police PlayerID; Target PlayerID }
type EndDiscussionEarly struct{ HostID PlayerID } // Q-AD-5=C
type SubmitVote struct{ Voter PlayerID; Target PlayerID }
type ToggleVoice struct{ HostID PlayerID; On bool }

type Event interface{} // sealed via type switch
type GameStarted struct{ State State }
type PhaseChanged struct{ Phase Phase; Day int; Deadline time.Time }
type RoleRevealedToPlayer struct{ PlayerID PlayerID; Role Role; Keyword string } // 비공개 채널
type IntroSpeakerChanged struct{ PlayerID PlayerID; SecondsLeft int }
type PoliceResult struct{ Police PlayerID; Target PlayerID; Team Team } // 비공개
type DeathAnnounced struct{ Victim PlayerID }
type PeacefulNight struct{}
type DiscussionTimerTick struct{ SecondsLeft int }
type VoteTallied struct{ Counts map[PlayerID]int; Eliminated *PlayerID; Recount bool }
type GameEnded struct{ Winner Team; Reveal []Player }
```

---

## C1. GameEngine

```go
package game

type Engine interface {
    // 새 게임 시작. 인원수 검증 후 RoleAssigner를 통해 역할/키워드를 배분하고
    // 초기 상태(Phase=INTRO, Day=1)와 GameStarted/PhaseChanged 이벤트를 반환.
    Start(players []PlayerID, opts Options) (State, []Event, error)

    // 외부 입력(Action)을 받아 상태 머신을 한 번 진전시키고 발생한 이벤트들을 반환.
    // 부적합 입력(잘못된 단계, 사망자 입력 등)은 error.
    Apply(action Action) (State, []Event, error)

    // 시간 기반 진전 (자기소개 자동 진행, 토론 타이머, 야간 마감 등).
    // 멱등(idempotent) — 동일 시각으로 여러 번 호출되어도 안전.
    Tick(now time.Time) (State, []Event, error)

    // 현재 상태 스냅샷.
    Snapshot() State

    // 영속화된 스냅샷에서 엔진 상태 복원 (서버 재시작 시).
    Restore(s State) error
}

func New(assigner RoleAssigner, clock Clock, rand io.Reader) Engine
```

---

## C2. RoleAssigner

```go
package game

type Assignments struct {
    PlayerRoles    map[PlayerID]Role
    PlayerKeywords map[PlayerID]string
}

type RoleAssigner interface {
    // 인원수 6~12명 검증 후 역할 분배표(FR-2.2)와 무작위 키워드를 적용.
    // 동일 역할의 플레이어는 동일 키워드를 받음 (FR-3.1 잠정 가정).
    Assign(playerIDs []PlayerID, seed int64) (Assignments, error)
}

func NewAssigner(keywordPool KeywordPool) RoleAssigner

type KeywordPool interface {
    // 역할별 키워드 풀에서 1개 무작위 추출. 외부 파일 분리 가능 (점진적 확장 대비, FR-7).
    Pick(role Role, rng *rand.Rand) string
}
```

---

## C3. SessionManager

```go
package session

type Manager interface {
    // 호스트가 첫 접속 시 호출. 기존 활성 세션이 있으면 거부.
    CreateSession(host Player) error

    // 닉네임으로 입장 (또는 재입장). 닉네임 중복은 error.
    JoinPlayer(name string) (PlayerID, error)

    // 호스트만 호출 가능. GameEngine.Start 위임.
    StartGame(host PlayerID, opts Options) error

    // 클라이언트 입력 → GameEngine.Apply 위임 → 이벤트를 announce로 변환 후 hub로 디스패치.
    SubmitAction(action Action) error

    // 호스트 컨트롤 (조기 종료, 일시정지 등).
    HostControl(host PlayerID, cmd HostCommand) error

    // 백그라운드 틱커가 주기적으로 호출.
    Tick(now time.Time) error

    // 현재 상태 (디버그/관리용).
    State() State
}

type HostCommand string
const (
    HostStart      HostCommand = "START"
    HostEarlyVote  HostCommand = "EARLY_VOTE"   // Q-AD-5=C
    HostPause      HostCommand = "PAUSE"
    HostAbort      HostCommand = "ABORT"
)

func New(engine game.Engine, store persistence.Store, hub ws.Hub, ann announce.Service) Manager
```

---

## C4. AnnouncementService

```go
package announce

type Announcement struct {
    Text    string  // 자막 + TTS 텍스트
    Public  bool    // true=공용 화면, false=특정 플레이어 비공개
    Target  game.PlayerID // Public=false일 때만 사용
    Voice   VoicePayload // pitch/rate/언어
    Urgent  bool // 인터럽트 (이전 발화 중단)
}

type VoicePayload struct {
    Lang  string  // "ko-KR"
    Pitch float64 // 0.8 (근엄)
    Rate  float64 // 0.9
}

type Service interface {
    // 이벤트들을 받아 안내(자막+TTS 텍스트) 목록으로 변환.
    Render(events []game.Event) []Announcement
}

func New() Service
```

---

## C5. PersistenceStore

```go
package persistence

type Store interface {
    SaveSnapshot(ctx context.Context, s game.State) error
    LoadActiveSnapshot(ctx context.Context) (state game.State, found bool, err error)
    ClearActive(ctx context.Context) error
    SaveResult(ctx context.Context, r GameResult) error
    ListResults(ctx context.Context, limit int) ([]GameResult, error)
    Close() error
}

type GameResult struct {
    GameID    string
    StartedAt time.Time
    EndedAt   time.Time
    Winner    game.Team
    Players   []ResultPlayer // 닉네임 + 배정 역할 + 최종 생존
}

func NewSQLite(path string) (Store, error) // modernc.org/sqlite
```

---

## C6. WSHub

```go
package ws

type Hub interface {
    // gorilla/websocket Upgrader.Upgrade 후 클라이언트 등록. 닉네임/역할(공용/플레이어) 식별.
    Register(conn *websocket.Conn, kind ClientKind, playerID game.PlayerID) (ClientID, error)

    // 클라이언트 단절 등록 해제 (재연결을 위한 grace period 포함 가능).
    Unregister(id ClientID)

    // 도메인 안내(공용 또는 비공개)를 대상 클라이언트에 송신.
    Dispatch(ann announce.Announcement) error

    // 추가로 도메인 이벤트(역할 공개 등)를 비공개 채널로 송신.
    DispatchEvent(ev game.Event) error

    // 클라이언트로부터 입력 수신 (콜백 등록).
    OnAction(handler func(action game.Action) error)
}

type ClientKind int
const (
    ClientPublic ClientKind = iota
    ClientPlayer
)

func New(upgrader websocket.Upgrader, log *slog.Logger) Hub
```

---

## C7. HTTPServer

```go
package httpx // 'http' 충돌 회피

func NewRouter(hub ws.Hub, mgr session.Manager, assets fs.FS, store persistence.Store) http.Handler
// 라우트:
//   GET  /                  -> SPA index.html (assets)
//   GET  /assets/*          -> 정적 자산
//   GET  /public            -> SPA fallback
//   GET  /play              -> SPA fallback
//   GET  /api/results       -> JSON, store.ListResults
//   GET  /ws                -> hub.Register (Upgrade)
//   GET  /healthz           -> "ok"

func PrintLANAddresses(port int) // 시작 시 호스트 LAN IP들 콘솔 출력
```

---

## C8. PublicView (React)

```ts
// web/src/views/PublicView.tsx
type PublicState = {
  phase: Phase;
  day: number;
  deadline?: number;        // epoch ms
  players: PublicPlayer[];  // 닉네임/생존 여부만 (역할 비공개)
  voteResult?: VoteResult;
  ended?: { winner: Team; reveal: RevealedPlayer[] };
  voiceOn: boolean;
  ttsAvailable: boolean;
};

interface TTSQueue {
  enqueue(text: string, opts: { urgent?: boolean; pitch?: number; rate?: number; lang?: string }): void;
  toggle(on: boolean): void;
  cancelAll(): void;
}

// 컴포넌트: <PublicView /> 내부에서 useWebSocket() 훅으로 메시지 수신,
// useTTSQueue() 훅으로 안내 발화. 호스트 컨트롤 패널은 호스트 클라이언트에서만 표시.
```

---

## C9. PlayerView (React)

```ts
// web/src/views/PlayerView.tsx
type PlayerState = {
  me: { id: string; name: string; role: Role; keyword: string; alive: boolean };
  phase: Phase;
  day: number;
  candidates: { id: string; name: string }[]; // 입력 가능한 대상
  myAction?: Action;                           // 미적용 입력 (수정 가능)
  policeResult?: { target: string; team: Team };
  hint?: string;                               // 단계별 가이드 텍스트
};

// 컴포넌트: <PlayerView /> — 닉네임 입력 → 입장 → 상태별 입력 UI.
//   마피아 차례: 살해 대상 선택 (Q-AD-7: 마피아 중 누구든 입력 가능, 마지막 입력이 채택; 다른 마피아는 현재 선택을 표시)
//   의사 차례: 보호 대상 선택 (자가 보호 허용)
//   경찰 차례: 조사 대상 선택 → 결과는 비공개 이벤트로 본인 화면에만 표시
//   투표: 후보 1명 선택, 잠금 후 재변경 불가(잠정)
```

---

## 메서드 매트릭스 요약

| 컴포넌트 | 핵심 메서드 |
|---|---|
| GameEngine | `Start`, `Apply`, `Tick`, `Snapshot`, `Restore` |
| RoleAssigner | `Assign` |
| SessionManager | `CreateSession`, `JoinPlayer`, `StartGame`, `SubmitAction`, `HostControl`, `Tick`, `State` |
| AnnouncementService | `Render` |
| PersistenceStore | `SaveSnapshot`, `LoadActiveSnapshot`, `ClearActive`, `SaveResult`, `ListResults`, `Close` |
| WSHub | `Register`, `Unregister`, `Dispatch`, `DispatchEvent`, `OnAction` |
| HTTPServer | `NewRouter`, `PrintLANAddresses` |
| PublicView (FE) | `<PublicView/>` + `useTTSQueue`, `useWebSocket` |
| PlayerView (FE) | `<PlayerView/>` + 단계별 입력 컴포넌트 |
