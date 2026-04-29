# Logical Components — U3 Realtime Transport

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-design-patterns.md`, `tech-stack-decisions.md`, `functional-design/*.md`

본 문서는 U3의 논리적 구성요소를 정의합니다. U3는 단일 패키지 `internal/transport/ws`에 위치합니다.

---

## 1. 구성요소 카탈로그

| ID | 구성요소 | 종류 | 책임 | 적용 패턴 |
|---|---|---|---|---|
| LC-U3-1 | `Hub` | 인터페이스 + impl | Register / Unregister / Run / Close, UpgradeHandler 노출 | P-U3-1, P-U3-7, P-U3-9 |
| LC-U3-2 | `Client` | struct | conn + Out chan + ctx + cancel + 메타데이터 | P-U3-4 |
| LC-U3-3 | `clientRegistry` | struct (Hub 내부) | byID/byPlayerID/publics 동기화 | P-U3-1 |
| LC-U3-4 | `readLoop` | goroutine | 클라이언트 메시지 수신 + handleIncoming 디스패치 | (FD §4) |
| LC-U3-5 | `writeLoop` | goroutine | Out chan → conn write + ping ticker | P-U3-4, single-writer |
| LC-U3-6 | `handleIncoming` | 함수 | 14종 type → SessionManager 메서드/Action 매핑 + 에러 응답 | P-U3-3 |
| LC-U3-7 | `onEvent` | 함수 (SessionManager Subscribe 핸들러) | EventOut 라우팅 + announce 송신 | P-U3-2, P-U3-5 |
| LC-U3-8 | `routeEvent` | 함수 | Visibility → 대상 클라이언트 slice | P-U3-8 |
| LC-U3-9 | `enqueue` | 함수 | ctx 확인 + select default → disconnect | P-U3-6 |
| LC-U3-10 | `protocol` | 패키지 내 wire 타입 모음 | incoming/outgoing 메시지 struct + 직렬화 헬퍼 | NFR-U3-M5 |
| LC-U3-11 | `idGen` | 함수 | 8-byte hex16 ClientID 생성 | Q-NFRD-U3-5=A |

---

## 2. 패키지 / 파일 레이아웃 (확정)

```
internal/transport/ws/
├── doc.go              # 패키지 godoc
├── hub.go              # Hub 인터페이스 + hub struct + New + Run/Close + UpgradeHandler (LC-U3-1)
├── client.go           # Client struct + clientRegistry (LC-U3-2, LC-U3-3)
├── handlers.go         # readLoop + handleIncoming (LC-U3-4, LC-U3-6)
├── writer.go           # writeLoop + enqueue (LC-U3-5, LC-U3-9)
├── dispatch.go         # onEvent + routeEvent + buildEventMsg (LC-U3-7, LC-U3-8)
├── protocol.go         # incoming/outgoing wire 타입 + JSON 직렬화 헬퍼 (LC-U3-10)
├── id.go               # newClientID (LC-U3-11)
└── *_test.go           # 단위 테스트 (in-memory + net.Pipe)
```

---

## 3. 구성요소별 상세

### 3.1 LC-U3-1 Hub

```go
type Hub interface {
    Register(conn *websocket.Conn) (ClientID, error)
    Unregister(id ClientID)
    Run(ctx context.Context) error
    Close() error
    UpgradeHandler() http.HandlerFunc
}

type hub struct {
    mgr         session.SessionManager
    upgrader    websocket.Upgrader
    log         *slog.Logger
    registry    *clientRegistry
    unsubscribe func()
    rootCtx     context.Context
    rootCancel  context.CancelFunc
    closed      atomic.Bool
}

func New(upgrader websocket.Upgrader, mgr session.SessionManager, log *slog.Logger) Hub
```

### 3.2 LC-U3-2 Client

```go
type ClientKind int

const (
    ClientPublic ClientKind = iota
    ClientPlayer
)

type Client struct {
    ID         ClientID
    Kind       ClientKind
    PlayerID   game.PlayerID  // PUBLIC이면 ""

    Conn       *websocket.Conn
    Out        chan []byte    // 버퍼 16

    ctx        context.Context
    cancel     context.CancelFunc

    JoinedAt   time.Time
}
```

### 3.3 LC-U3-3 clientRegistry

```go
type clientRegistry struct {
    mu         sync.RWMutex
    byID       map[ClientID]*Client
    byPlayerID map[game.PlayerID]*Client
    publics    map[ClientID]*Client
}

func newClientRegistry() *clientRegistry
func (r *clientRegistry) add(c *Client)
func (r *clientRegistry) remove(id ClientID) *Client
func (r *clientRegistry) bindPlayer(c *Client, pid game.PlayerID) (oldID ClientID, hadOld bool)
func (r *clientRegistry) byPlayerSafe(pid game.PlayerID) *Client
func (r *clientRegistry) snapshotPublic() []*Client
func (r *clientRegistry) snapshotPlayers() []*Client
func (r *clientRegistry) all() []*Client
```

> 모든 snapshot* 메서드는 RLock으로 보호된 slice clone 반환 — 호출자가 락 없이 iterate 안전.

### 3.4 LC-U3-7 onEvent

```go
func (h *hub) onEvent(out session.EventOut) {
    defer func() {
        if r := recover(); r != nil {
            h.log.Error("onEvent panicked", "panic", r)
        }
    }()

    if out.Envelope.Event != nil {
        msg, err := buildEventMsg(out.Envelope)
        if err != nil { /* log */; return }
        for _, c := range h.routeEvent(out.Envelope) {
            h.enqueue(c, msg)
        }
    }

    if out.Announcement != nil && !out.Announcement.IsEmpty() && out.Announcement.ForPublicOnly {
        msg := mustMarshal(announceMsg{...})
        for _, c := range h.registry.snapshotPublic() {
            h.enqueue(c, msg)
        }
    }
}
```

### 3.5 LC-U3-8 routeEvent

```go
func (h *hub) routeEvent(env game.EventEnvelope) []*Client {
    switch env.Visibility {
    case game.VisPublic:
        out := h.registry.snapshotPublic()
        out = append(out, h.registry.snapshotPlayers()...)
        return out
    case game.VisPlayer:
        if c := h.registry.byPlayerSafe(env.PlayerID); c != nil {
            return []*Client{c}
        }
        return nil
    case game.VisRoleMafia:
        state := h.mgr.Snapshot()  // U2 인터페이스 확장 필요 (P-U3-8)
        out := []*Client{}
        for _, p := range state.Players {
            if p.Alive && p.Role == game.RoleMafia {
                if c := h.registry.byPlayerSafe(p.ID); c != nil {
                    out = append(out, c)
                }
            }
        }
        return out
    }
    return nil
}
```

### 3.6 LC-U3-10 protocol.go (wire 타입)

```go
// incoming
type incomingEnvelope struct {
    Type string          `json:"type"`
    Raw  json.RawMessage `json:"-"`  // ReadJSON 후 부모 raw에서 분리
}

type joinPayload     struct{ Name string `json:"name"` }
type resumePayload   struct{ Token string `json:"token"` }
type targetPayload   struct{ Target game.PlayerID `json:"target"` }
type hostStartPayload struct{ Options game.Options `json:"options"` }
type voiceTogglePayload struct{ On bool `json:"on"` }

// outgoing
type welcomeMsg struct {
    Type            string `json:"type"`
    ClientID        ClientID `json:"clientId"`
    Kind            string `json:"kind"`
    ProtocolVersion string `json:"protocolVersion"`
}

type joinedMsg struct {
    Type     string `json:"type"`
    PlayerID game.PlayerID `json:"playerId"`
    Token    string `json:"token"`
    IsHost   bool `json:"isHost"`
}

type snapshotMsg struct {
    Type   string `json:"type"`
    State  game.State `json:"state"`
    Your   yourInfo `json:"your"`
    IsHost bool `json:"isHost"`
}

type yourInfo struct {
    Role        game.Role `json:"role,omitempty"`
    Keyword     string `json:"keyword,omitempty"`
    Team        game.Team `json:"team,omitempty"`
    MafiaCohort []game.PlayerID `json:"mafiaCohort,omitempty"`
}

type eventMsg struct {
    Type       string `json:"type"`
    Visibility string `json:"visibility"`
    Event      eventPayload `json:"event"`
}

type eventPayload struct {
    Kind string `json:"kind"`
    // 합성 — Kind에 따라 다른 필드들. 송신 시 type-specific struct로 marshal
    // (eventPayload 자체는 디코딩 시에만 partial 사용)
    Phase           game.Phase `json:"phase,omitempty"`
    Day             int `json:"day,omitempty"`
    DeadlineMs      int64 `json:"deadlineMs,omitempty"`
    Role            game.Role `json:"role,omitempty"`
    Keyword         string `json:"keyword,omitempty"`
    PlayerID        game.PlayerID `json:"playerId,omitempty"`
    SecondsLeft     int `json:"secondsLeft,omitempty"`
    Victim          game.PlayerID `json:"victim,omitempty"`
    Counts          map[game.PlayerID]int `json:"counts,omitempty"`
    Eliminated      *game.PlayerID `json:"eliminated,omitempty"`
    Recount         bool `json:"recount,omitempty"`
    MafiaIDs        []game.PlayerID `json:"mafiaIds,omitempty"`
    RepresentativeID game.PlayerID `json:"representativeId,omitempty"`
    Target          game.PlayerID `json:"target,omitempty"`
    Police          game.PlayerID `json:"police,omitempty"`
    Team            game.Team `json:"team,omitempty"`
    OldID           game.PlayerID `json:"oldId,omitempty"`
    NewID           game.PlayerID `json:"newId,omitempty"`
    Winner          *game.Team `json:"winner,omitempty"`
    EndReason       game.EndReason `json:"endReason,omitempty"`
    Reveal          []game.Player `json:"reveal,omitempty"`
    On              *bool `json:"on,omitempty"`
}

type announceMsg struct {
    Type     string `json:"type"`
    Subtitle string `json:"subtitle"`
    Speech   string `json:"speech"`
    Severity string `json:"severity"`
}

type errorMsg struct {
    Type    string `json:"type"`
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### 3.7 LC-U3-11 idGen

```go
func newClientID() ClientID {
    var b [8]byte
    _, _ = rand.Read(b[:])
    return ClientID(hex.EncodeToString(b[:]))
}
```

---

## 4. 책임 매트릭스 (NFR ↔ LC)

| NFR Req | 책임 LC |
|---|---|
| NFR-U3-R1 (끊김 감지) | LC-U3-4 (read deadline) + LC-U3-5 (ping ticker) |
| NFR-U3-R2 (last-wins) | LC-U3-1 (bindPlayer) + LC-U3-3 (registry) |
| NFR-U3-R3 (snapshot push) | LC-U3-6 (handleIncoming → resume → snapshotMsg) |
| NFR-U3-R4 (graceful shutdown) | LC-U3-1 (Hub.Close → cancel 모든 client.ctx) |
| NFR-U3-R5 (panic 격리) | LC-U3-7 (onEvent recover) |
| NFR-U3-P1~P5 | LC-U3-7 (onEvent), LC-U3-8 (routeEvent), LC-U3-9 (enqueue 즉시 반환) |
| NFR-U3-C1 (직렬화 위임) | LC-U3-6 (SubmitAction 직접 호출) — 자체 락 없음 |
| NFR-U3-C2 (race-free) | LC-U3-3 (RWMutex) + LC-U3-2 (ctx 종료 패턴) |
| NFR-U3-C3 (single-writer) | LC-U3-5 (writeLoop만 conn.WriteMessage) |
| NFR-U3-C4 (onEvent I/O 금지) | LC-U3-7 (enqueue만, conn write 미수행) |
| NFR-U3-M1~M6 | 모든 LC가 godoc + 단위 테스트 가능 |
| NFR-U3-S1 (비공개 라우팅) | LC-U3-8 (routeEvent) — Snapshot 단일 진실 소스 |
| NFR-U3-S2 (로그 비공개) | LC-U3-4 (readLoop debug log type만) |
| NFR-U3-S3 (read limit) | LC-U3-4 (Conn.SetReadLimit) |
| NFR-U3-G1 (채널 16) | LC-U3-2 (Out 버퍼 16) |
| NFR-U3-G2 (goroutine 누수 0) | LC-U3-2 (ctx cancel) + LC-U3-1 (Close에서 모두 취소) |

---

## 5. Import Cycle 분석

```
internal/transport/ws ──→ internal/session   (Hub.New 인자, Subscribe 호출, SubmitAction 호출, Snapshot 호출)
                      ──→ internal/announce  (Announcement 타입 사용)
                      ──→ internal/game      (Action/Event/State/EngineError/Visibility 타입 사용)
                      ──→ github.com/gorilla/websocket
```

- session → ws: ✗ (session은 Subscribe 콜백을 받을 뿐, ws 패키지 import 안 함)
- announce → ws: ✗
- game → ws: ✗

**검증**: import cycle 없음. ws는 외부 도메인을 모두 의존하는 가장 외곽 패키지.

> ⚠️ U2 SessionManager 인터페이스에 `Snapshot() game.State` 메서드 추가 필요. 기존 U2 코드에 1줄 추가 (engine.Snapshot()를 락 안에서 노출). 하위 호환 유지.

---

## 6. 외부 인프라 / 의존

| 외부 | 사용처 | 비고 |
|---|---|---|
| `github.com/gorilla/websocket` | LC-U3-1, LC-U3-2, LC-U3-4, LC-U3-5 | RFC 6455 표준 라이브러리 |
| `crypto/rand` | LC-U3-11 (ClientID) | 표준 lib |
| `encoding/json` | LC-U3-10 (wire 직렬화) | 표준 lib |
| `log/slog` | 모든 LC | 표준 lib |
| `net/http` | LC-U3-1 (UpgradeHandler) | 표준 lib |
| `context` | LC-U3-1, LC-U3-2 (per-client ctx) | 표준 lib |
| `sync` | LC-U3-3 (RWMutex) | 표준 lib |

---

## 7. 검증 체크리스트

- [x] 모든 LC가 정확히 하나의 패키지(`internal/transport/ws`)에 위치
- [x] Hub 인터페이스 5개 메서드 (Register/Unregister/Run/Close/UpgradeHandler)
- [x] import cycle 없음 (ws는 외곽, session/announce/game 모두 의존만)
- [x] U2 SessionManager 확장(Snapshot) 식별 + 하위 호환 명시
- [x] NFR Req 항목이 모두 책임 LC에 매핑됨 (§4)
- [x] 외부 의존성 1개(gorilla/websocket)만 추가 (NFR-U3-M4)
- [x] 모든 패턴(P-U3-1~10)이 LC에 적용됨 (§1 표)
- [x] wire 타입은 protocol.go 1개 파일에 집중 (NFR-U3-M5)
