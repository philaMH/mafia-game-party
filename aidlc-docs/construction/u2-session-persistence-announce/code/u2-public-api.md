# U2 — Public API Catalog

**작성일**: 2026-04-26
**대상 패키지**: `github.com/saltware/mafia-game/internal/session`, `internal/announce`, `internal/persistence`
**버전**: 1.0 (Code Generation 1차 산출물)

본 문서는 U2가 외부(U3 WSHub, U4 HTTP Bootstrap, 테스트)에 노출하는 **공개 API**의 빠른 참조용 카탈로그입니다. godoc 주석은 소스 파일에서 직접 확인 가능합니다.

---

## 1. `internal/session` — SessionManager facade

### 1.1 인터페이스

```go
type SessionManager interface {
    CreateSession(ctx context.Context, hostName string) (JoinResult, error)
    JoinPlayer(ctx context.Context, name string) (JoinResult, error)
    ResumePlayer(ctx context.Context, token string) (JoinResult, error)
    StartGame(ctx context.Context, hostID game.PlayerID, opts game.Options) ([]EventOut, error)
    SubmitAction(ctx context.Context, action game.Action) ([]EventOut, error)
    Tick(now time.Time)
    Subscribe(handler EventHandler) (unsubscribe func())
    Close(ctx context.Context) error
}
```

### 1.2 데이터 타입

```go
type Member struct {
    ID        game.PlayerID
    Name      string
    Token     string  // 비밀, 로그 금지
    Connected bool
    JoinedAt  time.Time
}

type JoinResult struct {
    PlayerID     game.PlayerID
    Token        string
    IsHost       bool
    CurrentState game.State
    YourRole     game.Role     // ResumePlayer일 때만 채워짐
    YourKeyword  string
    YourTeam     game.Team
    MafiaCohort  []game.PlayerID  // 마피아 viewer일 때만
}

type EventOut struct {
    Envelope     game.EventEnvelope
    Announcement *announce.Announcement  // 비공개 이벤트는 nil
}

type EventHandler func(EventOut)
```

### 1.3 옵션

```go
type SessionOpts struct {
    TickInterval time.Duration  // 기본 1s
    EventLog     bool           // events 테이블 기록 (기본 false)
    MaxLobbySize int            // 기본 12
    MinPlayers   int            // 기본 6
}
```

### 1.4 생성자

```go
func New(
    store persistence.PersistenceStore,
    catalog announce.AnnouncementCatalog,
    engine game.Engine,
    clock game.Clock,    // nil → wallClock 사용
    rng io.Reader,       // nil → crypto/rand.Reader 사용
    opts SessionOpts,
) (SessionManager, error)
```

> ⚠️ Subscribe 핸들러는 단일 GM 락 안에서 호출됨 — 빠르게 반환할 것. 무거운 작업은 핸들러 안에서 별도 고루틴으로 분리.

### 1.5 PrivateView 빌더

```go
type PrivateView struct {
    State       game.State
    YourRole    game.Role
    YourKeyword string
    YourTeam    game.Team
    MafiaCohort []game.PlayerID
    IsHost      bool
}

// pid == "" 이면 PublicView (모든 Role/Keyword 마스킹).
func BuildPrivateView(state game.State, pid game.PlayerID, hostID game.PlayerID) PrivateView
```

---

## 2. `internal/announce` — Korean catalog

### 2.1 인터페이스

```go
type AnnouncementCatalog interface {
    Render(env game.EventEnvelope, ctx CatalogContext) Announcement
    RenderError(err error, sender game.PlayerID, ctx CatalogContext) Announcement
}

func NewDefaultCatalog() AnnouncementCatalog
```

### 2.2 데이터 타입

```go
type Announcement struct {
    Subtitle      string
    Speech        string
    Severity      Severity
    ForPublicOnly bool
}

func (Announcement) IsEmpty() bool

type Severity string
const (
    SeverityInfo     Severity = "INFO"
    SeverityEmphasis Severity = "EMPHASIS"
    SeverityWarn     Severity = "WARN"
)

type CatalogContext struct {
    GetName               func(id game.PlayerID) string
    IntroSecondsPerPlayer int
}
```

### 2.3 시스템 토스트

```go
func SystemRestore() Announcement         // "이전 게임이 복원되었습니다…"
func SystemPersistFailure() Announcement  // "게임 상태를 저장하지 못했습니다…"
```

### 2.4 카탈로그 적용 범위 (BR-U2-CAT-1)

`Render`가 빈 `Announcement`(`IsEmpty() == true`)를 반환하는 이벤트:
- `RoleRevealedToPlayer`, `MafiaCohortRevealed`, `MafiaTargetSelected`, `PoliceResult`, `MafiaRepresentativeReassigned` (모두 비공개)
- `VoteTallied{Eliminated≠nil, Recount=false}` — 직후의 `Eliminated`가 안내 발행
- 미지원 PhaseChanged (LOBBY/END)

---

## 3. `internal/persistence` — SQLite store

### 3.1 인터페이스

```go
type PersistenceStore interface {
    SaveSnapshot(ctx context.Context, snap Snapshot) error
    LoadActiveSnapshot(ctx context.Context) (Snapshot, bool, error)
    DeleteActiveSnapshot(ctx context.Context) error
    SaveResultAndClearActive(ctx context.Context, r GameResult) error  // 원자 트랜잭션
    ListResults(ctx context.Context, limit int) ([]GameResult, error)
    AppendEvent(ctx context.Context, gameID string, env game.EventEnvelope) error
    ArchiveCorrupt(ctx context.Context) error
    Close() error
}

func OpenSqlite(ctx context.Context, path string) (PersistenceStore, error)
```

### 3.2 데이터 타입

```go
type PersistedMember struct {
    ID        game.PlayerID
    Name      string
    Token     string
    Connected bool
    JoinedAt  time.Time
}

type Snapshot struct {
    GameID  string
    State   game.State
    Members []PersistedMember
    HostID  game.PlayerID
}

type GameResult struct {
    GameID    string
    StartedAt time.Time
    EndedAt   time.Time
    Winner    *game.Team
    EndReason game.EndReason
    Options   game.Options
    Members   []PersistedMember
    Reveal    []game.Player
}
```

### 3.3 운영 보장

- 부팅 시 자동 PRAGMA 적용: `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=ON`
- DB 파일 권한 자동 chmod **0600** (소유자 RW만)
- `MaxOpenConns=1` (단일 라이터 직렬화)
- 모든 SQL은 prepared statement 캐시에서 재사용
- `ArchiveCorrupt`: 본 파일 + `-wal`, `-shm` 사이드카까지 함께 rename

---

## 4. 와이어링 예시 (Composition Root, U4에서 작성 예정)

```go
ctx := context.Background()
store, err := persistence.OpenSqlite(ctx, "./data/mafia.db")
if err != nil { log.Fatal(err) }

cat := announce.NewDefaultCatalog()
engine := game.NewDefault(game.NewDefaultKeywordPool())

mgr, err := session.New(store, cat, engine, nil, nil, session.SessionOpts{})
if err != nil { log.Fatal(err) }
defer mgr.Close(ctx)

unsub := mgr.Subscribe(func(out session.EventOut) {
    // U3 WSHub: out.Envelope을 가시성에 따라 라우팅,
    //          out.Announcement?.Subtitle을 자막으로 push
})
defer unsub()
```

---

## 5. 변경 영향도 / 호환성

- 본 API는 U3~U5 모두 의존 — Method 추가는 안전, 제거/시그니처 변경은 깨짐
- `Announcement.Subtitle`/`Speech` 분리 변경(향후 미세조정)은 가능하나 호환성 취약 → 새 필드 추가만 권장
- `PersistedMember`는 SQLite 스키마와 결합. 필드 추가 시 `state_json`/`member_json` migration 검토 필요
