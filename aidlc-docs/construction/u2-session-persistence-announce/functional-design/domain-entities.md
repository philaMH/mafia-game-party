# Domain Entities — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1, `application-design/component-methods.md`, `u1-game-core/code/u1-public-api.md`, `plans/u2-session-functional-design-plan.md`

본 문서는 U2의 도메인 엔티티(타입)와 SQLite 스키마를 정의합니다. U1의 도메인 타입(`game.PlayerID`, `game.State`, `game.Action`, `game.EventEnvelope`)은 그대로 import하여 사용합니다.

---

## 1. Session (SessionManager 보유)

```
Session {
  Engine          game.Engine          // U1 인스턴스, 단일 GM 락 아래 직렬 호출
  GameID          string               // 활성 게임 식별자 (UUID, 게임 시작 시 발급)
  Members         map[PlayerID]Member  // 입장한 플레이어 (게임 시작 후엔 변경 불가)
  HostID          PlayerID
  Started         bool                 // false: LOBBY 모집 중, true: 게임 진행 중
}

Member {
  ID         PlayerID
  Name       string                  // 닉네임
  Token      string                  // 재연결 식별 토큰 (서버 발급, 사용자에게만 1회 노출)
  Connected  bool                    // 현재 WS 연결 상태 (정보용; 라우팅은 U3)
  JoinedAt   time.Time
}
```

### 불변식 (Invariants)
- `Members`는 `Started=true` 이후 추가/제거 금지 (Q-FD-U2-11=A)
- `Token`은 게임 1판 동안 불변, 게임 종료 후 다음 LOBBY로 진입 시 재발급
- `HostID ∈ Members`
- `Members[id].Token`은 32바이트 random hex (URL-safe), 같은 게임 내 중복 없음

### Lifecycle 상태도
```
empty → LOBBY (CreateSession)
LOBBY → LOBBY (JoinPlayer / Leave) ─[StartGame]→ ACTIVE
ACTIVE → ACTIVE (Apply / Tick) ─[GameEnded]→ DONE
DONE  ─[NewGame]→ LOBBY (선택, 같은 호스트가 새 게임 시작)
```

> 본 시스템은 **단일 활성 세션**만 보유. 동시 진행 게임 1개.

---

## 2. JoinResult (JoinPlayer 응답)

```
JoinResult {
  PlayerID  game.PlayerID
  Token     string             // 재연결 시 제시할 비밀 토큰
  IsHost    bool
  CurrentState  game.State     // 현재 게임 상태 스냅샷 (마스킹된 사본)
}
```

> 클라이언트는 Token을 localStorage에 저장. 재연결 시 `ResumeRequest{Token}`으로 다시 입장.

---

## 3. ResumeRequest

```
ResumeRequest {
  Token   string
}
```

성공 시 `JoinResult` 반환 (CurrentState만 갱신). 실패 시 `EngineError{CodeUnknownPlayer}` 또는 `CodeValidation`.

---

## 4. PrivateView (마스킹된 State)

플레이어 또는 공용 화면에 송신할 때, U2가 State를 가시성 정책에 따라 마스킹한 사본을 반환합니다.

```
PrivateView {
  State        game.State        // Players[*].Role / Keyword 마스킹됨
  YourRole     game.Role         // 본인의 역할 (있으면)
  YourKeyword  string            // 본인의 키워드 (있으면)
  YourTeam     game.Team         // 진영 (마피아만 동맹 식별 가능)
  MafiaCohort  []PlayerID        // 마피아일 때만 동맹 마피아 목록
  IsHost       bool
}
```

### 마스킹 규칙
| 화면 | Players[*].Role | Players[*].Keyword |
|---|---|---|
| PublicView | 빈 문자열 | 빈 문자열 |
| PlayerView (본인 외) | 빈 문자열 | 빈 문자열 |
| PlayerView (본인) | 자기 Role | 자기 Keyword |
| PlayerView (마피아 → 다른 마피아) | "MAFIA" 노출 | 빈 문자열 |
| GameEnded 후 모든 화면 | 그대로 노출 (Reveal) | 그대로 |

---

## 5. AnnouncementService — 카탈로그 인터페이스 (FR-7.2 외부화 가능)

```
AnnouncementCatalog interface {
  // Render 도메인 이벤트를 한국어 안내 메시지로 변환.
  // 동일 메시지가 자막(subtitle)과 TTS(speech)로 사용됨.
  // PublicView에만 보내야 하는 메시지는 ForPublicOnly=true 반환.
  Render(env game.EventEnvelope) Announcement
}

Announcement {
  Subtitle      string  // 자막(공용 화면 큰 텍스트)
  Speech        string  // TTS 발화 텍스트 (보통 Subtitle과 동일)
  Severity      Severity  // INFO, EMPHASIS, WARN
  ForPublicOnly bool    // true → PublicView 전용 (PlayerView에 송신 안 함)
}

Severity = "INFO" | "EMPHASIS" | "WARN"
```

### 안내 카탈로그 (FR-8.4 풍부, Q-FD-U2-8=A 근엄 톤)

| 이벤트 | 한국어 안내 (Subtitle = Speech) | Severity | ForPublicOnly |
|---|---|---|---|
| `GameStarted` | "마피아 게임이 시작됩니다. 모든 시민은 침묵 속에서 운명을 받아들이시오." | EMPHASIS | true |
| `PhaseChanged{INTRO}` | "각자 차례대로 자기소개를 진행하시오. 한 사람당 {seconds}초가 주어집니다." | INFO | true |
| `IntroSpeakerChanged` | "{name}, 발언하시오." | INFO | true |
| `PhaseChanged{NIGHT}` | "이제 밤이 깊어졌습니다. 모두 눈을 감으시오." | EMPHASIS | true |
| `PhaseChanged{DAY}` | "{day}일째 아침이 밝았습니다. 마을은 어떤 운명을 맞이했는가." | EMPHASIS | true |
| `PhaseChanged{VOTE}` | "토론은 끝났습니다. 이제 의심스러운 자에게 표를 던지시오." | EMPHASIS | true |
| `PhaseChanged{RECOUNT}` | "결과가 같습니다. 마지막 한 번 더, 신중히 선택하시오." | WARN | true |
| `DeathAnnounced` | "{victim}이(가) 새벽에 발견되었습니다. 마을의 슬픔이 깊어집니다." | EMPHASIS | true |
| `PeacefulNight` | "어젯밤은 평온하였습니다. 누구도 사라지지 않았습니다." | INFO | true |
| `Eliminated` | "{name}이(가) 마을의 결정으로 처형되었습니다. 그의 정체는 {role_kr}이었습니다." | EMPHASIS | true |
| `DiscussionTimerTick{30}` | "토론 종료까지 30초 남았습니다." | INFO | true |
| `DiscussionTimerTick{10}` | "토론 종료까지 10초 남았습니다. 마음을 정하시오." | WARN | true |
| `DiscussionTimerTick{0}` | "토론이 종료되었습니다." | INFO | true |
| `VoteTallied{Recount=true}` | "득표가 동률입니다. 재투표를 진행합니다." | WARN | true |
| `VoteTallied{Eliminated=nil, Recount=false}` | "재투표 또한 동률이었습니다. 오늘은 처형이 없습니다." | INFO | true |
| `GameEnded{MAFIA_WIN}` | "마피아의 승리. 어둠이 마을을 삼켰습니다." | EMPHASIS | true |
| `GameEnded{CITIZEN_WIN}` | "시민의 승리. 정의가 어둠을 몰아냈습니다." | EMPHASIS | true |
| `GameEnded{HOST_FORCE_END}` | "진행자의 결정으로 게임이 종료되었습니다." | INFO | true |
| `RoleRevealedToPlayer` | (PlayerView 비공개 — 안내 없음, U5가 자체 표시) | — | — |
| `MafiaCohortRevealed` | (마피아 비공개 — 안내 없음) | — | — |
| `MafiaTargetSelected` | (마피아 비공개 — 안내 없음) | — | — |
| `PoliceResult` | (경찰 비공개 — 안내 없음) | — | — |
| `MafiaRepresentativeReassigned` | (마피아 비공개 — 안내 없음) | — | — |
| `VoiceToggled{On=true}` | "음성 안내가 활성화되었습니다." | INFO | true |
| `VoiceToggled{On=false}` | "음성 안내가 비활성화되었습니다." | INFO | true |

> `role_kr` 매핑: `MAFIA → 마피아`, `CITIZEN → 시민`, `DOCTOR → 의사`, `POLICE → 경찰`.
> `{name}` 보간은 `Session.Members[id].Name` 조회.

### ErrorAnnouncement (Q-FD-U2-6=A)

EngineError 9종에 대한 한국어 사용자 메시지:

| ErrorCode | 한국어 사용자 메시지 | 표시 위치 |
|---|---|---|
| `VALIDATION_ERROR` | "입력이 올바르지 않습니다: {field}" | PlayerView 폼 옆 |
| `WRONG_PHASE_ERROR` | "지금은 그 행동을 할 수 없습니다." | PlayerView 토스트 |
| `PERMISSION_DENIED_ERROR` | "권한이 없습니다." | PlayerView 토스트 |
| `ROLE_MISMATCH_ERROR` | "당신의 역할은 그 행동을 할 수 없습니다." | PlayerView 토스트 |
| `NOT_REPRESENTATIVE_ERROR` | "이번 게임의 마피아 대표자만 살해 대상을 입력할 수 있습니다." | PlayerView 토스트 |
| `DEAD_PLAYER_ERROR` | "사망한 플레이어는 행동할 수 없습니다." | PlayerView 토스트 |
| `ALREADY_DONE_ERROR` | "이번 단계에서는 이미 행동을 완료했습니다." | PlayerView 토스트 |
| `INVALID_TARGET_ERROR` | "선택할 수 없는 대상입니다." | PlayerView 토스트 |
| `UNKNOWN_PLAYER_ERROR` | "알 수 없는 플레이어입니다." | PlayerView 토스트 |

---

## 6. PersistenceStore — SQLite 스키마

### 6.1 DDL

```sql
-- Q-FD-U2-9=A — 3 테이블

CREATE TABLE IF NOT EXISTS active_snapshot (
    id              INTEGER PRIMARY KEY CHECK (id = 1),  -- 단일 row
    game_id         TEXT    NOT NULL,
    state_json      BLOB    NOT NULL,
    member_json     BLOB    NOT NULL,                    -- []Member 직렬화
    host_id         TEXT    NOT NULL,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_results (
    game_id         TEXT    PRIMARY KEY,
    started_at      DATETIME NOT NULL,
    ended_at        DATETIME NOT NULL,
    winner          TEXT,                  -- "MAFIA" | "CITIZEN" | NULL (force-end)
    end_reason      TEXT    NOT NULL,
    options_json    BLOB    NOT NULL,
    members_json    BLOB    NOT NULL,
    reveal_json     BLOB    NOT NULL       -- []Player (final reveal, roles included)
);
CREATE INDEX IF NOT EXISTS idx_game_results_ended_at ON game_results(ended_at DESC);

CREATE TABLE IF NOT EXISTS events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id         TEXT    NOT NULL,
    event_type      TEXT    NOT NULL,
    visibility      TEXT    NOT NULL,      -- "PUBLIC" | "PLAYER" | "ROLE_MAFIA"
    recipient_id    TEXT,                  -- VisPlayer일 때만
    payload_json    BLOB    NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_game_id ON events(game_id);
```

### 6.2 운영 규칙

- **`active_snapshot`**: 항상 0 또는 1 row. UPSERT로 갱신 (`INSERT OR REPLACE`).
- **`game_results`**: 게임 종료 시 INSERT. 같은 game_id 중복 차단 (`PRIMARY KEY`).
- **`events`**: 디버깅용. 비활성화 가능 (옵션). 진행 중 게임이라도 보관 → 사후 분석. 단일 게임 종료 시 (옵션) Truncate.
- 모든 작업은 **트랜잭션 1회**로 묶음 (`BEGIN; ...; COMMIT;`)
- 파일 위치: `./data/mafia.db` (Q-FD-U2-10=A). 디렉터리는 `os.MkdirAll(0755)`로 자동 생성. 환경변수 `MAFIA_DB_PATH`로 오버라이드 가능.
- WAL 모드 활성화: `PRAGMA journal_mode=WAL;` (재시작 안전성 강화)
- `PRAGMA synchronous=NORMAL;` (NFR-1 안정성과 성능 균형)

### 6.3 PersistenceStore 인터페이스 (요약)

```go
type PersistenceStore interface {
    SaveSnapshot(ctx context.Context, snap Snapshot) error
    LoadActiveSnapshot(ctx context.Context) (Snapshot, bool, error)
    DeleteActiveSnapshot(ctx context.Context) error
    SaveResult(ctx context.Context, r GameResult) error
    ListResults(ctx context.Context, limit int) ([]GameResult, error)
    AppendEvent(ctx context.Context, gameID string, env game.EventEnvelope) error
    Close() error
}

type Snapshot struct {
    GameID  string
    State   game.State
    Members []Member
    HostID  game.PlayerID
}

type GameResult struct {
    GameID     string
    StartedAt  time.Time
    EndedAt    time.Time
    Winner     *game.Team
    EndReason  game.EndReason
    Options    game.Options
    Members    []Member
    Reveal     []game.Player        // 모든 플레이어의 최종 역할 공개
}
```

---

## 7. SessionManager 인터페이스 (요약, 코드 단계 확정)

```go
type SessionManager interface {
    // 호스트가 신규 세션을 만들 때 호출. 토큰을 발급하고 LOBBY 진입.
    CreateSession(ctx context.Context, hostName string) (JoinResult, error)

    // 신규 플레이어가 LOBBY에 입장. 토큰 발급.
    JoinPlayer(ctx context.Context, name string) (JoinResult, error)

    // 토큰을 가진 플레이어가 재연결. 단계 무관 허용.
    ResumePlayer(ctx context.Context, token string) (JoinResult, error)

    // 호스트가 게임 시작.
    StartGame(ctx context.Context, hostID game.PlayerID, opts game.Options) error

    // 액션을 큐에 넣고 실행. 발생한 안내 이벤트도 반환.
    SubmitAction(ctx context.Context, action game.Action) ([]EventOut, error)

    // 1초 간격으로 호출되어 시간 기반 단계 진전.
    Tick(now time.Time)

    // 외부(WSHub)가 이벤트 구독. 콜백은 SessionManager 락 내에서 호출됨 → 무거운 작업 금지.
    Subscribe(handler EventHandler) (unsubscribe func())

    // 종료 hook (백그라운드 ticker 정리, 마지막 스냅샷 저장).
    Close(ctx context.Context) error
}

type EventOut struct {
    Envelope     game.EventEnvelope
    Announcement *Announcement      // catalog가 매핑한 안내 (없을 수 있음)
}

type EventHandler func(out EventOut)
```

---

## 8. 검증 체크리스트

- [x] Session/Member 불변식 명확
- [x] JoinResult / ResumeRequest 형태 정의
- [x] PrivateView 마스킹 규칙 5종 명시
- [x] AnnouncementCatalog 인터페이스 + 카탈로그 25개 항목 (FR-8.4 풍부)
- [x] ErrorAnnouncement 9종 한국어 매핑
- [x] SQLite 스키마 3 테이블 + 인덱스 + WAL/synchronous PRAGMA
- [x] Snapshot/GameResult 레코드 정의
- [x] SessionManager 공개 인터페이스 정의
- [x] FR-7.2 외부화 가능한 인터페이스로 카탈로그 추상화
