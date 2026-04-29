# U3 — Public API Catalog

**작성일**: 2026-04-26
**대상 패키지**: `github.com/saltware/mafia-game/internal/transport/ws`
**버전**: 1.0 (Code Generation 1차 산출물)

본 문서는 U3가 외부(U4 HTTP Bootstrap, 테스트)에 노출하는 **공개 API**의 빠른 참조용 카탈로그입니다. godoc 주석은 소스 파일에서 직접 확인 가능합니다.

---

## 1. Hub 인터페이스

```go
type Hub interface {
    Register(conn *websocket.Conn) (ClientID, error)
    Unregister(id ClientID)
    Run(ctx context.Context) error
    Close() error
    UpgradeHandler() http.HandlerFunc
}

func New(
    upgrader websocket.Upgrader,
    mgr session.SessionManager,
    log *slog.Logger,    // nil → slog.Default()
) Hub
```

**의미**:
- `Register`는 이미 업그레이드된 `*websocket.Conn`을 받아 신규 Client 등록 + read/write goroutine 구동.
- `UpgradeHandler()`는 `mux.HandleFunc("/ws", hub.UpgradeHandler())` 한 줄로 끝나는 캡슐화된 진입점 — 권장 사용.
- `Run(ctx)`은 ctx 취소 시까지 블록 후 Close 위임.
- `Close()`는 모든 클라이언트 close + Subscribe 해제 (idempotent, < 2초).

---

## 2. 데이터 타입

```go
type ClientID string

type ClientKind int
const (
    ClientPublic ClientKind = iota  // /public
    ClientPlayer                    // /play (token 인증 후)
)

func (k ClientKind) String() string  // "PUBLIC" | "PLAYER" | "UNKNOWN"
```

> Hub 외부에서 직접 다룰 수 있는 타입은 `ClientID`와 `ClientKind`뿐. 내부 `Client` struct는 비공개.

---

## 3. 와이어 프로토콜 — 메시지 타입 상수 (외부 호환용)

```go
// Incoming (client → server)
const (
    TypeHostCreateSession  = "host:create-session"
    TypeJoin               = "join"
    TypeResume             = "resume"
    TypeHostStart          = "host:start"
    TypeSubmitAdvanceIntro = "submit:advance-intro"
    TypeSubmitMafiaKill    = "submit:mafia-kill"
    TypeSubmitDoctorHeal   = "submit:doctor-heal"
    TypeSubmitPoliceCheck  = "submit:police-check"
    TypeSubmitEndNight     = "submit:end-night"
    TypeSubmitEndDiscuss   = "submit:end-discussion"
    TypeSubmitVote         = "submit:vote"
    TypeHostToggleVoice    = "host:toggle-voice"
    TypeHostForceEnd       = "host:force-end"
    TypeSubscribePublic    = "subscribe-public"
)

// Outgoing (server → client)
const (
    TypeWelcome  = "welcome"
    TypeJoined   = "joined"
    TypeSnapshot = "snapshot"
    TypeEvent    = "event"
    TypeAnnounce = "announce"
    TypeError    = "error"
)
```

> 클라이언트(U5)가 type discriminator를 그대로 매칭 가능. 외부에서 const 참조해 typo 방지.

---

## 4. 메시지 페이로드 (요약)

### 4.1 Incoming

```json
// host:create-session
{ "type": "host:create-session", "name": "host" }

// join
{ "type": "join", "name": "철수" }

// resume
{ "type": "resume", "token": "abc123..." }

// host:start
{ "type": "host:start", "options": { "mafiaCount": 1, "introSecondsPerPlayer": 20, "discussionSeconds": 180, "doctorSelfHealAllowed": true, "announcementVoiceOn": true } }

// submit:vote (또는 mafia-kill / doctor-heal / police-check 동일 형식)
{ "type": "submit:vote", "target": "p_xxx" }

// host:toggle-voice
{ "type": "host:toggle-voice", "on": true }

// 인자 없는 액션: submit:advance-intro / submit:end-night / submit:end-discussion / host:force-end / subscribe-public
{ "type": "submit:end-night" }
```

### 4.2 Outgoing

```json
// welcome (Register 직후)
{ "type": "welcome", "clientId": "abc...", "kind": "PUBLIC", "protocolVersion": "v1" }

// joined (host:create-session / join / resume 성공)
{ "type": "joined", "playerId": "p_xxx", "token": "tok...", "isHost": true }

// snapshot (resume 직후)
{ "type": "snapshot", "state": { ... game.State JSON ... }, "your": { "role": "MAFIA", "keyword": "...", "team": "MAFIA", "mafiaCohort": [...] }, "isHost": false }

// event (도메인 이벤트 push)
{ "type": "event", "visibility": "PUBLIC", "event": { "kind": "PhaseChanged", "phase": "DAY", "day": 2, "deadlineMs": 1714000000000 } }

// announce (한국어 자막 + TTS)
{ "type": "announce", "subtitle": "이제 밤이 깊어졌습니다…", "speech": "이제 밤이 깊어졌습니다…", "severity": "EMPHASIS" }

// error (송신자 한정)
{ "type": "error", "code": "WRONG_PHASE_ERROR", "message": "..." }
```

### 4.3 event kind 카탈로그 (Engine event 15종 매핑)

| kind | 주요 필드 |
|---|---|
| `GameStarted` | (없음 — state는 별도 snapshot으로 전송) |
| `PhaseChanged` | phase, day, deadlineMs |
| `RoleRevealedToPlayer` | playerId, role, keyword |
| `MafiaCohortRevealed` | mafiaIds[], representativeId |
| `IntroSpeakerChanged` | playerId, secondsLeft |
| `MafiaTargetSelected` | representativeId, target |
| `PoliceResult` | police, target, team |
| `DeathAnnounced` | victim |
| `PeacefulNight` | (없음) |
| `DiscussionTimerTick` | secondsLeft |
| `VoteTallied` | counts, eliminated?, recount |
| `Eliminated` | playerId, role |
| `MafiaRepresentativeReassigned` | oldId, newId |
| `GameEnded` | winner?, endReason, reveal[] |
| `VoiceToggled` | on |

---

## 5. 보장사항 (계약)

| 항목 | 보장 |
|---|---|
| 클라이언트당 read goroutine 1 + write goroutine 1 | gorilla single-writer 요구 만족 |
| Subscribe 핸들러 panic 격리 | onEvent의 defer recover (P-U3-5) |
| 비공개 이벤트 라우팅 (NFR-U3-S1) | VisPlayer는 단일 PID, VisRoleMafia는 살아있는 마피아 PID만 |
| 송신 큐 가득 → 단일 클라이언트 disconnect | 다른 클라이언트 영향 0 (P-U3-6) |
| last-connect-wins | 새 resume 시 기존 동일 PID 클라이언트 강제 close (P-U3-7) |
| ping 25s / read deadline 30s | 끊긴 연결 ≤ 31초 내 정리 |
| graceful shutdown < 2초 | NFR-U3-R4 |
| 메시지 크기 한도 64 KiB | `Conn.SetReadLimit` (NFR-U3-S3) |

---

## 6. 와이어링 예시 (Composition Root, U4에서 작성 예정)

```go
ctx := context.Background()

store, err := persistence.OpenSqlite(ctx, "./data/mafia.db")
if err != nil { log.Fatal(err) }

cat := announce.NewDefaultCatalog()
engine := game.NewDefault(game.NewDefaultKeywordPool())

mgr, err := session.New(store, cat, engine, nil, nil, session.SessionOpts{})
if err != nil { log.Fatal(err) }
defer mgr.Close(ctx)

hub := ws.New(websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true }, // LAN 한정
}, mgr, slog.Default())
defer hub.Close()

mux := http.NewServeMux()
mux.HandleFunc("/ws", hub.UpgradeHandler())
// ... (U4: SPA, /api/results 등)
http.ListenAndServe(":8080", mux)
```

---

## 7. 변경 영향도

- 본 API는 U4(HTTP Bootstrap)와 U5(Web Frontend)가 의존 — Method/메시지 type 추가는 안전, 제거/시그니처 변경은 깨짐.
- event kind 추가는 안전 (Switch default → "Unknown"으로 fallback).
- `EventOut.State` 추가는 SessionManager Subscribe 호환성 영향 — 기존 핸들러 코드는 새 필드 무시 가능 (backward-compat).
