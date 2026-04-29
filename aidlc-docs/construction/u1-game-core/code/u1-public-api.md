# U1 Game Core — Public API Catalog

**작성일**: 2026-04-26
**대상 패키지**: `github.com/saltware/mafia-game/internal/game`
**버전**: 1.0 (Code Generation 1차 산출물)

본 문서는 U1이 다른 단위(U2 SessionManager 등)에 노출하는 **공개 API**의 빠른 참조용 카탈로그입니다. godoc 주석은 소스 파일에서 직접 확인 가능합니다.

---

## 1. 도메인 식별자 / 열거형

```go
type PlayerID string
type Role     string  // "MAFIA" | "CITIZEN" | "DOCTOR" | "POLICE"
type Team     string  // "MAFIA" | "CITIZEN"
type Phase    string  // "LOBBY" | "INTRO" | "NIGHT" | "DAY" | "VOTE" | "RECOUNT" | "END"
type EndReason string // "MAFIA_WIN" | "CITIZEN_WIN" | "HOST_FORCE_END"

func TeamOf(r Role) Team
```

---

## 2. 도메인 데이터 타입

```go
type Player struct {
    ID      PlayerID
    Name    string
    Alive   bool
    Role    Role    // 비공개
    Keyword string  // 비공개
}

type Options struct {
    MafiaCount            int
    IntroSecondsPerPlayer int
    DiscussionSeconds     int
    DoctorSelfHealAllowed bool
    AnnouncementVoiceOn   bool
}

func DefaultOptions(playerCount int) Options

type State struct {
    GameID, Phase, Day, Players, HostID, Settings, StartedAt, Deadline,
    IntroSpeakerIdx, IntroSpeakerStartedAt,
    MafiaRepresentativeID, PendingMafiaTarget, PendingDoctorTarget,
    PendingPoliceTarget, PoliceCheckedThisNight,
    Votes, VoteRound, VoteCandidates,
    Winner, EndReason, LastTickAt
}

func (State) Clone() State
func (State) Pending() PendingActions
func (*State) FindPlayer(PlayerID) (*Player, bool)
func (*State) LiveCount() int
func (*State) LiveMafiaCount() int
func (*State) LiveCitizenSideCount() int
func (*State) HasLivingDoctor() bool
func (*State) HasLivingPolice() bool
func (*State) LivingMafiaIDs() []PlayerID
```

---

## 3. Action 타입 (sealed)

```go
type Action interface { /* sealed */ }

type StartGame          struct { HostID PlayerID; Options Options }
type AdvanceIntro       struct { HostID PlayerID }
type SubmitMafiaKill    struct { Mafia, Target PlayerID }
type SubmitDoctorHeal   struct { Doctor, Target PlayerID }
type SubmitPoliceCheck  struct { Police, Target PlayerID }
type EndNightEarly      struct { HostID PlayerID }
type EndDiscussionEarly struct { HostID PlayerID }
type SubmitVote         struct { Voter, Target PlayerID }
type ToggleVoice        struct { HostID PlayerID; On bool }
type ForceEndGame       struct { HostID PlayerID }
```

---

## 4. Event 타입 + Visibility (sealed)

```go
type Event interface { /* sealed */ }
type Visibility int
const ( VisPublic Visibility = iota; VisPlayer; VisRoleMafia )

type EventEnvelope struct {
    Event      Event
    Visibility Visibility
    PlayerID   PlayerID  // VisPlayer일 때만
}

// 15종 이벤트 — 자세한 내용은 event.go 참조
type GameStarted struct { State State }
type PhaseChanged struct { Phase Phase; Day int; Deadline time.Time }
type RoleRevealedToPlayer struct { PlayerID PlayerID; Role Role; Keyword string }
type MafiaCohortRevealed struct { MafiaIDs []PlayerID; RepresentativeID PlayerID }
type IntroSpeakerChanged struct { PlayerID PlayerID; SecondsLeft int }
type MafiaTargetSelected struct { RepresentativeID, Target PlayerID }
type PoliceResult struct { Police, Target PlayerID; Team Team }
type DeathAnnounced struct { Victim PlayerID }
type PeacefulNight struct{}
type DiscussionTimerTick struct { SecondsLeft int }
type VoteTallied struct { Counts map[PlayerID]int; Eliminated *PlayerID; Recount bool }
type Eliminated struct { PlayerID PlayerID; Role Role }
type MafiaRepresentativeReassigned struct { OldID, NewID PlayerID }
type GameEnded struct { Winner *Team; EndReason EndReason; Reveal []Player }
type VoiceToggled struct { On bool }
```

---

## 5. 에러 표현

```go
type ErrorCode string
const (
    CodeValidation, CodeWrongPhase, CodePermissionDenied, CodeRoleMismatch,
    CodeNotRepresentative, CodeDeadPlayer, CodeAlreadyDone, CodeInvalidTarget,
    CodeUnknownPlayer ErrorCode
)

type EngineError struct {
    Code    ErrorCode
    Message string
    Field   string
}
func (*EngineError) Error() string
func (*EngineError) Is(target error) bool

// Sentinel — errors.Is(err, ErrXxx) 매칭용
var (
    ErrValidation, ErrWrongPhase, ErrPermissionDenied, ErrRoleMismatch,
    ErrNotRepresentative, ErrDeadPlayer, ErrAlreadyDone, ErrInvalidTarget,
    ErrUnknownPlayer *EngineError
)

// 누적 검증 에러
type ValidationErrors []FieldError
type FieldError struct { Field string; Code ErrorCode; Message string }
```

### 호출자 사용 예

```go
state, evs, err := engine.Apply(action)
if err != nil {
    switch {
    case errors.Is(err, game.ErrPermissionDenied):
        // 호스트 권한 부족
    case errors.Is(err, game.ErrValidation):
        var ve game.ValidationErrors
        if errors.As(err, &ve) {
            for _, fe := range ve { /* 필드별 표시 */ }
        }
    case errors.Is(err, game.ErrWrongPhase):
        // 단계 위반
    }
}
```

---

## 6. Engine 인터페이스

```go
type Engine interface {
    Start(gameID string, host PlayerID, players []Player, opts Options) (State, []EventEnvelope, error)
    Apply(action Action) (State, []EventEnvelope, error)
    Tick(now time.Time) (State, []EventEnvelope, error)
    Snapshot() State
    Restore(s State) error
}

func New(assigner RoleAssigner, clock Clock, rng io.Reader) Engine
func NewDefault(pool KeywordPool) Engine  // 운영 헬퍼: realClock + crypto/rand
```

> ⚠️ Engine은 **동시 호출 비안전**. 호출자가 직렬화 필요 (U2가 단일 mutex로 보장).

---

## 7. 인프라 인터페이스 (확장 지점)

```go
// 키워드 풀 (FR-7.1 외부화 가능)
type KeywordPool interface {
    Pick(role Role, rng *math.Rand) (string, error)
}
func NewDefaultKeywordPool() KeywordPool
func LoadKeywordPool(r io.Reader) (KeywordPool, error)  // JSON

// 역할 배정자
type RoleAssigner interface {
    Assign(playerIDs []PlayerID, opts Options, rng *math.Rand) (Assignments, error)
}
func NewAssigner(pool KeywordPool) RoleAssigner

type Assignments struct {
    PlayerRoles      map[PlayerID]Role
    PlayerKeywords   map[PlayerID]string
    MafiaIDs         []PlayerID
    RepresentativeID PlayerID
}

// 시간
type Clock interface { Now() time.Time }
type FakeClock struct { T time.Time }
func (*FakeClock) Now() time.Time
func (*FakeClock) Advance(d time.Duration)
```

---

## 8. 가시성 정책 — U3 라우팅 가이드

| Visibility | 라우팅 대상 |
|---|---|
| `VisPublic` | 모든 PublicView + 살아있는 모든 PlayerView (Player 화면) |
| `VisPlayer` | `EventEnvelope.PlayerID` 1인의 PlayerView |
| `VisRoleMafia` | 살아있는 모든 마피아 PlayerView (envelope의 PlayerID는 무시) |

> 비공개 정보가 들어 있는 이벤트(`RoleRevealedToPlayer`, `MafiaCohortRevealed`, `MafiaTargetSelected`, `PoliceResult`, `MafiaRepresentativeReassigned`)를 PublicView로 보내면 게임 무결성이 깨집니다. 라우팅 테스트 필수.

---

## 9. 무작위성 / 결정성

| 환경 | RNG | 결정성 |
|---|---|---|
| 운영 | `crypto/rand.Reader`를 `New(...)`에 주입 | 비결정 |
| 단위 테스트 | `bytes.NewReader([]byte{...})` 또는 SHA-256 시드 reader | 결정 |

게임 1판당 inner PRNG가 1회 시드되며, 그 안에서 Shuffle/Pick/대표자 선정이 결정적으로 진행됨.

---

## 10. 변경 영향도

본 API는 U2~U5 모두 의존하므로 **호환성 유지가 중요**:
- Action/Event 타입 추가는 안전 (sealed interface 임베드)
- 기존 Action/Event 필드 제거는 깨짐 — 추가만 권장
- ErrorCode 추가는 안전, 제거는 깨짐
- State 필드 추가는 JSON 직렬화 호환성을 위해 신규 필드만, 기본값 0/nil 안전성 유지
