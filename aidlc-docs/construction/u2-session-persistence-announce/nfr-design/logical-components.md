# Logical Components — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-design-patterns.md`, `tech-stack-decisions.md`, `functional-design/*.md`

본 문서는 U2의 논리적 구성요소를 정의합니다. U2는 3개 패키지(`internal/session`, `internal/announce`, `internal/persistence`)에 걸쳐 있습니다.

---

## 1. 구성요소 카탈로그

| ID | 구성요소 | 패키지 | 종류 | 책임 | 적용 패턴 |
|---|---|---|---|---|---|
| LC-U2-1 | `SessionManager` | session | 인터페이스 + impl | 단일 GM 락, 세션 라이프사이클, 액션 dispatch | P-U2-3, P-U2-4, P-U2-6, P-U2-7 |
| LC-U2-2 | `Session` (내부 상태) | session | struct | Members, GameID, HostID, Started 보유 | — |
| LC-U2-3 | `Member` | session | 데이터 타입 | ID/Name/Token/Connected/JoinedAt | P-U2-10 |
| LC-U2-4 | `tickLoop` | session | goroutine | 1초 ticker → Engine.Tick | P-U2-7 |
| LC-U2-5 | `tokenIssuer` | session | 함수 | crypto/rand 32-byte hex 토큰 발급, 충돌 차단 | P-U2-10 |
| LC-U2-6 | `viewBuilder` | session | 함수 | PrivateView 마스킹 5종 | NFR-U2-S4 |
| LC-U2-7 | `AnnouncementCatalog` | announce | 인터페이스 | Render(env) → Announcement | FR-7.2 |
| LC-U2-8 | `defaultCatalog` | announce | impl | 25개 한국어 매핑 | Q-FD-U2-7=A |
| LC-U2-9 | `errorAnnouncer` | announce | 함수 | EngineError 9종 → 한국어 메시지 | Q-FD-U2-6=A |
| LC-U2-10 | `PersistenceStore` | persistence | 인터페이스 | SaveSnapshot/Load/SaveResult/List/AppendEvent/Close | — |
| LC-U2-11 | `sqliteStore` | persistence | impl | modernc.org/sqlite + database/sql | P-U2-1, P-U2-2, P-U2-5, P-U2-8 |
| LC-U2-12 | `schema` | persistence | 함수 | DDL 적용, PRAGMA 설정 | P-U2-8 |
| LC-U2-13 | `recovery` | persistence | 함수 | 손상 스냅샷 archive(rename) | P-U2-9 |

---

## 2. 패키지 / 파일 레이아웃 (확정)

### 2.1 `internal/session/`
```
internal/session/
├── doc.go                    # godoc
├── types.go                  # Session, Member, JoinResult, EventOut, SessionOpts
├── session.go                # SessionManager 인터페이스 + session struct + New
├── lifecycle.go              # CreateSession, JoinPlayer, ResumePlayer, StartGame, Close
├── action.go                 # SubmitAction + persistAndDispatch + dispatchHandlers
├── tick.go                   # tickLoop, Tick (멱등 호출 위임은 Engine)
├── view.go                   # PrivateView 빌더 (LC-U2-6)
├── token.go                  # newToken, issueUniqueToken (LC-U2-5)
└── *_test.go
```

### 2.2 `internal/announce/`
```
internal/announce/
├── doc.go
├── catalog.go                # AnnouncementCatalog 인터페이스 + Announcement, Severity
├── catalog_default.go        # defaultCatalog: Render switch (LC-U2-8)
├── catalog_data.go           # 한국어 메시지 상수
├── error.go                  # ErrorAnnounce(err) → Announcement (LC-U2-9)
└── *_test.go
```

### 2.3 `internal/persistence/`
```
internal/persistence/
├── doc.go
├── store.go                  # PersistenceStore 인터페이스 + Snapshot, GameResult 타입
├── sqlite_store.go           # sqliteStore 구현체 (P-U2-1, P-U2-2)
├── schema.go                 # CREATE TABLE DDL + applyPragmas (P-U2-8)
├── recovery.go               # ArchiveCorrupt (rename) (P-U2-9)
└── *_test.go
```

---

## 3. 구성요소별 상세

### 3.1 LC-U2-1 SessionManager

```go
type SessionManager interface {
    CreateSession(ctx context.Context, hostName string) (JoinResult, error)
    JoinPlayer(ctx context.Context, name string) (JoinResult, error)
    ResumePlayer(ctx context.Context, token string) (JoinResult, error)
    StartGame(ctx context.Context, hostID game.PlayerID, opts game.Options) error
    SubmitAction(ctx context.Context, action game.Action) ([]EventOut, error)
    Tick(now time.Time)
    Subscribe(handler EventHandler) (unsubscribe func())
    Close(ctx context.Context) error
}

type session struct {
    mu          sync.Mutex
    persistence PersistenceStore
    catalog     announce.AnnouncementCatalog
    engine      game.Engine
    clock       game.Clock
    rand        io.Reader

    sess        Session   // GameID, Members, HostID, Started
    handlers    []handlerEntry  // unsubscribe id → handler

    stopCh      chan struct{}
    stopped     bool
    tickerDone  chan struct{}
}

func New(persistence PersistenceStore, catalog announce.AnnouncementCatalog,
         engine game.Engine, clock game.Clock, rand io.Reader, opts SessionOpts) (SessionManager, error)
```

### 3.2 LC-U2-7 AnnouncementCatalog

```go
type AnnouncementCatalog interface {
    Render(env game.EventEnvelope, ctx CatalogContext) Announcement
    RenderError(err error, sender game.PlayerID, ctx CatalogContext) Announcement
}

type CatalogContext struct {
    GetName func(id game.PlayerID) string
}

type Announcement struct {
    Subtitle      string
    Speech        string
    Severity      Severity
    ForPublicOnly bool
}
```

### 3.3 LC-U2-10 PersistenceStore

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

type Snapshot struct {
    GameID   string
    State    game.State
    Members  []Member
    HostID   game.PlayerID
}

type GameResult struct {
    GameID    string
    StartedAt time.Time
    EndedAt   time.Time
    Winner    *game.Team
    EndReason game.EndReason
    Options   game.Options
    Members   []Member
    Reveal    []game.Player
}
```

> 주의: `Member`는 session 패키지가 정의했지만 persistence가 그대로 사용 — 순환 import 회피를 위해 본 단계에서 위치를 결정해야 함.
>
> **결정**: `Member`는 `internal/session` 패키지가 정의처. `internal/persistence`는 session을 import (역방향 X). 만약 순환 발생 시, 공용 타입 `internal/session/types.go` 분리가 가장 자연스러움.

> Code Generation 단계 시 import 그래프를 검증: session → persistence (호출), session → announce (호출). persistence → session(타입) 의존이 발생하면 import cycle.
>
> **해결책**: `Member` 타입을 별도 minor 패키지 `internal/session/types`로 분리하거나, persistence가 익명 struct/interface로 받음. 단순성 우선 → persistence가 자체 정의한 동등 struct(`PersistedMember`)를 사용하고 session이 변환. (코드 단계 미세 결정)

### 3.4 LC-U2-11 sqliteStore

```go
type sqliteStore struct {
    db   *sql.DB
    path string

    saveSnapshotStmt   *sql.Stmt
    loadSnapshotStmt   *sql.Stmt
    deleteSnapshotStmt *sql.Stmt
    saveResultStmt     *sql.Stmt
    listResultsStmt    *sql.Stmt
    appendEventStmt    *sql.Stmt
}

func NewSqliteStore(ctx context.Context, path string) (*sqliteStore, error) {
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { ... }
    db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
    ...
    db.SetMaxOpenConns(1)
    s := &sqliteStore{db: db, path: path}
    if err := s.applyPragmas(ctx); err != nil { ... }
    if err := s.applySchema(ctx); err != nil { ... }
    if err := s.prepareStmts(ctx); err != nil { ... }
    if err := os.Chmod(path, 0600); err != nil { ... }   // NFR-U2-S3
    return s, nil
}
```

### 3.5 LC-U2-13 ArchiveCorrupt

```go
func (s *sqliteStore) ArchiveCorrupt(ctx context.Context) error {
    if err := s.Close(); err != nil { /* log only */ }
    ts := time.Now().UTC().Format("20060102-150405")
    archived := fmt.Sprintf("%s.corrupt-%s", s.path, ts)
    if err := os.Rename(s.path, archived); err != nil {
        return err
    }
    // 새 DB 생성은 호출자가 NewSqliteStore 다시 호출
    return nil
}
```

---

## 4. 책임 매트릭스 (NFR ↔ LC)

| NFR Req | 책임 LC |
|---|---|
| NFR-U2-R1 (영속화 트리거) | LC-U2-1 (persistAndDispatch) + LC-U2-11 (실제 저장) |
| NFR-U2-R2 (자동 복원) | LC-U2-1 (boot) + LC-U2-11 + LC-U2-13 |
| NFR-U2-R3 (손상 처리) | LC-U2-13 |
| NFR-U2-R4 (트랜잭션) | LC-U2-11 (SaveResultAndClearActive) |
| NFR-U2-R5 (토큰) | LC-U2-5 |
| NFR-U2-R6 (graceful shutdown) | LC-U2-1 (Close), LC-U2-4 (stopCh) |
| NFR-U2-R7 (WAL/synchronous) | LC-U2-12 |
| NFR-U2-P1~P3 (성능) | LC-U2-11 (prepared stmts), LC-U2-1 (mutex contention) |
| NFR-U2-M1~M6 | 모든 LC가 godoc + 테스트 가능한 인터페이스로 노출 |
| NFR-U2-S1~S3 | LC-U2-5 (토큰), LC-U2-11 (0600 chmod) |
| NFR-U2-S4 (마스킹) | LC-U2-6 (viewBuilder) |
| NFR-U2-C1~C3 | LC-U2-1 (mutex), LC-U2-4 (tickLoop), panic recover |

---

## 5. 외부 인프라 / 의존

| 외부 | 사용처 | 비고 |
|---|---|---|
| SQLite (`./data/mafia.db`) | LC-U2-11 | 단일 파일, 0600, WAL |
| `modernc.org/sqlite` | LC-U2-11 | 순수 Go 드라이버 |
| `crypto/rand` | LC-U2-5 (토큰) | 표준 lib |
| `encoding/json` | LC-U2-11 (직렬화) | 표준 lib |
| `log/slog` | 모든 LC | 표준 lib (Go 1.21+) |

---

## 6. 검증 체크리스트

- [x] 모든 LC가 정확히 하나의 패키지에 위치
- [x] import cycle 가능성 식별 + 해결책 제시 (Member 위치)
- [x] NFR Req 항목이 모두 책임 LC에 매핑됨 (§4)
- [x] 외부 의존성 1개(modernc.org/sqlite)만 추가 (NFR-U2-M6)
- [x] 모든 패턴(P-U2-1~10)이 LC에 적용됨 (§1 표)
- [x] FR-7.2 외부화 인터페이스(AnnouncementCatalog, PersistenceStore)가 분리됨
