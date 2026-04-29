# NFR Design Patterns — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-requirements.md`, `tech-stack-decisions.md`, `functional-design/*.md`

---

## 1. 패턴 개요

| 패턴 ID | 패턴 | 적용 영역 | 주요 NFR | 출처 |
|---|---|---|---|---|
| P-U2-1 | 단일 라이터 connection pool | PersistenceStore SQLite 핸들 | Reliability, Performance | Q-NFRD-U2-1=A |
| P-U2-2 | Prepared statement 재사용 | SaveSnapshot / LoadActiveSnapshot 등 | Performance(P1<50ms) | Q-NFRD-U2-2=A |
| P-U2-3 | 영속화 실패 격리 (log+notify, no-block) | SubmitAction → SaveSnapshot | Reliability + UX | Q-NFRD-U2-3=A |
| P-U2-4 | EventHandler panic 격리 (`defer recover`) | Subscribe 콜백 호출 루프 | Reliability | Q-NFRD-U2-4=A |
| P-U2-5 | JSON BLOB + 결정적 직렬화 | Snapshot/Members/Options 저장 | Storage 결정성 | Q-NFRD-U2-5=A |
| P-U2-6 | 단일 GM mutex (FD에서 결정) | SessionManager 모든 메서드 | Concurrency | Q-FD-U2-1=A |
| P-U2-7 | 백그라운드 ticker + stopCh (FD) | Tick loop | Reliability | Q-FD-U2-5=A |
| P-U2-8 | WAL + synchronous=NORMAL (FD) | SQLite 부팅 PRAGMA | Reliability + P | NFR-U2-R7 |
| P-U2-9 | 손상 스냅샷 archive(rename) | Restore 실패 fallback | Reliability | Q-NFR-U2-8=A |
| P-U2-10 | 토큰 격리 (`crypto/rand` + 충돌 차단) | JoinPlayer | Security | Q-NFR-U2-6=A |

---

## 2. 패턴 다이어그램

```mermaid
flowchart LR
    Caller["U3 WSHub<br/>OnAction callback"] -->|SubmitAction| SM

    subgraph SM["SessionManager (P-U2-6 단일 mutex)"]
        Apply["Engine.Apply"]
        Persist["persistAndDispatch<br/>(P-U2-3 실패 격리)"]
        Dispatch["handlers loop<br/>(P-U2-4 panic 격리)"]
    end

    SM -->|SaveSnapshot| PS

    subgraph PS["PersistenceStore"]
        Pool["sql.DB pool<br/>(P-U2-1 MaxOpenConns=1)"]
        Stmts["Prepared Stmts<br/>(P-U2-2 캐싱)"]
        WAL["WAL + synchronous=NORMAL<br/>(P-U2-8)"]
    end

    PS -->|JSON Marshal<br/>(P-U2-5)| SQLite[("data/mafia.db<br/>0600")]

    Tick["tickLoop goroutine<br/>(P-U2-7 stopCh)"] -->|Tick| SM

    Recovery["bootRestore<br/>(P-U2-9 archive on fail)"] -->|Restore| SM

    style SM fill:#FFE0B2,stroke:#E65100,color:#000
    style PS fill:#F8BBD0,stroke:#AD1457,color:#000
    style SQLite fill:#FFF59D,stroke:#F57F17,color:#000
    style Tick fill:#C8E6C9,stroke:#2E7D32,color:#000
    style Recovery fill:#BBDEFB,stroke:#1565C0,color:#000
```

### 텍스트 대안

```
WSHub.OnAction → SessionManager(단일 mutex)
  → Engine.Apply
  → persistAndDispatch (실패 격리)
    → PersistenceStore.SaveSnapshot
       → sql.DB(MaxOpenConns=1) → 캐시된 Prepared Stmt
       → JSON Marshal → SQLite WAL 0600
  → handlers loop (panic 격리)

tickLoop goroutine → 1초마다 SessionManager.Tick
bootRestore → LoadActiveSnapshot → (실패 시 archive rename) → Engine.Restore
```

---

## 3. 패턴 상세

### 3.1 P-U2-1 — 단일 라이터 connection pool (Q-NFRD-U2-1=A)

**의도**: Go `database/sql`의 표준 connection pool을 사용하되, `MaxOpenConns=1`로 SQLite 라이터 직렬화. WAL 모드에서 reader는 다중 허용되지만 본 단위에서는 reader/writer 모두 동일 락 안에서 호출되므로 단일 connection이 가장 단순.

**적용**:
```go
db, err := sql.Open("sqlite", dsn)  // modernc.org/sqlite
if err != nil { return err }
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)  // 영구 (단일 프로세스)
```

**근거**:
- WAL 모드에서도 SQLite는 동시 writer 1개만 안전 → 명시적으로 1로 제한.
- ListResults 같은 read 작업도 단일 GM 락 안에서 호출되므로 추가 reader connection 불필요.
- `*sql.DB`를 사용하면 향후 reader pool 확장이 단순.

### 3.2 P-U2-2 — Prepared statement 재사용 (Q-NFRD-U2-2=A)

**의도**: 자주 쓰는 SQL을 `*sql.Stmt`로 한 번 prepare 후 매 호출에서 재사용. p99 지연 안정화.

**적용**:
```go
type sqliteStore struct {
    db *sql.DB

    saveSnapshot      *sql.Stmt
    loadSnapshot      *sql.Stmt
    deleteSnapshot    *sql.Stmt
    saveResult        *sql.Stmt
    listResults       *sql.Stmt
    appendEvent       *sql.Stmt
}

func (s *sqliteStore) prepareStmts(ctx context.Context) error {
    var err error
    s.saveSnapshot, err = s.db.PrepareContext(ctx, `INSERT OR REPLACE INTO active_snapshot (id, game_id, state_json, member_json, host_id, updated_at) VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP)`)
    if err != nil { return err }
    // ... 동일하게 5개 더
    return nil
}

func (s *sqliteStore) Close() error {
    s.saveSnapshot.Close()
    // ...
    return s.db.Close()
}
```

**근거**:
- prepare 1회 + execute N회 패턴이 매 호출 prepare보다 빠름.
- Close에서 `*sql.Stmt`를 모두 정리 → 자원 누수 방지.

### 3.3 P-U2-3 — 영속화 실패 격리 (Q-NFRD-U2-3=A)

**의도**: SaveSnapshot 실패가 게임 진행을 차단하지 않도록 격리. 다음 PhaseChanged에서 자동 재시도(매번 최신 state 저장하므로 누락된 중간 상태 손실 미발생).

**적용**:
```go
func (s *session) persistAndDispatch(state game.State, envs []game.EventEnvelope) []EventOut {
    outs := s.renderEnvs(envs)

    if shouldPersist(envs) {
        snap := buildSnapshot(s, state)
        if err := s.persistence.SaveSnapshot(s.ctx, snap); err != nil {
            slog.Error("save snapshot failed", "err", err, "game_id", s.session.GameID)
            // 호스트에게 시스템 안내 1건 추가
            outs = append(outs, EventOut{
                Announcement: &Announcement{
                    Subtitle: "게임 상태를 저장하지 못했습니다. 곧 다시 시도합니다.",
                    Severity: SeverityWarn,
                    ForPublicOnly: true,
                },
            })
            // game continues; next PhaseChanged will retry
        }
        if g, ok := findGameEnded(envs); ok {
            s.handleGameEnd(g, state)  // 별도 트랜잭션
        }
    }

    s.dispatchHandlers(outs)
    return outs
}
```

**근거**:
- NFR-1 안정성: 게임 흐름 끊김 회피.
- Q-NFR-U2-9=A: 재시도 없음 — 다음 트리거에서 자동.

### 3.4 P-U2-4 — EventHandler panic 격리 (Q-NFRD-U2-4=A)

**의도**: 하나의 Subscribe 핸들러(예: WSHub)가 panic해도 다른 핸들러와 SessionManager의 다음 호출은 정상.

**적용**:
```go
func (s *session) dispatchHandlers(outs []EventOut) {
    for _, h := range s.handlers {
        for _, out := range outs {
            s.callHandler(h, out)
        }
    }
}

func (s *session) callHandler(h EventHandler, out EventOut) {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("event handler panicked", "panic", r)
        }
    }()
    h(out)
}
```

**근거**:
- Subscribe는 외부 코드(WSHub) 호출 — 격리 필수.
- 단일 GM 락 안에서 호출되므로 panic 전파 시 락이 강제 해제되어 deadlock 가능 → recover로 정상 unwind.

### 3.5 P-U2-5 — JSON BLOB + 결정적 직렬화 (Q-NFRD-U2-5=A)

**의도**: `encoding/json`이 map 키를 정렬 직렬화하므로 동일 state → 동일 바이트 출력. 디버깅 / 비교 용이.

**적용**:
```go
func encodeJSON(v any) ([]byte, error) {
    return json.Marshal(v)  // map keys sorted by encoding/json since Go 1.12
}

// SaveSnapshot 흐름
stateJSON, err := encodeJSON(snap.State)
memberJSON, err := encodeJSON(snap.Members)
_, err = s.saveSnapshot.ExecContext(ctx, snap.GameID, stateJSON, memberJSON, snap.HostID)
```

**근거**:
- 외부 의존 0 (표준 라이브러리만).
- U1의 NFR-U1-S1 (State JSON 직렬화 가능)과 호환.
- 결정적 직렬화로 동일 스냅샷 두 번 저장해도 같은 바이트.

### 3.6 P-U2-9 — 손상 스냅샷 archive on fail (Q-NFR-U2-8=A)

**의도**: Restore 실패 시 데이터 보존하면서 새 LOBBY 진행 보장.

**적용**:
```go
func bootRestore(persistence PersistenceStore, engine game.Engine) error {
    snap, found, err := persistence.LoadActiveSnapshot(ctx)
    if err != nil {
        slog.Error("load snapshot failed; archiving", "err", err)
        return persistence.ArchiveCorrupt(ctx)
    }
    if !found {
        return nil
    }
    if err := engine.Restore(snap.State); err != nil {
        slog.Error("engine.Restore failed; archiving", "err", err)
        return persistence.ArchiveCorrupt(ctx)
    }
    return nil
}

// PersistenceStore.ArchiveCorrupt
//   1. db.Close()
//   2. os.Rename("data/mafia.db", "data/mafia.db.corrupt-{ts}")
//   3. db = sql.Open(...) 새 파일
//   4. schema 재적용
```

**근거**:
- 손상 데이터 보존(분석 가능) + 게임 진행 보장 (새 LOBBY).
- 자동 처리로 수동 개입 불필요.

### 3.7 P-U2-10 — 토큰 격리 (Q-NFR-U2-6=A)

**적용**:
```go
func newToken() (string, error) {
    var b [32]byte
    if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
        return "", err
    }
    return hex.EncodeToString(b[:]), nil  // 64 hex chars
}

func (s *session) issueUniqueToken() (string, error) {
    for i := 0; i < 5; i++ {  // 256-bit space → 충돌 확률 무시 가능, 안전 최대 5회
        t, err := newToken()
        if err != nil { return "", err }
        if !s.tokenInUse(t) { return t, nil }
    }
    return "", errors.New("token collision after 5 attempts")
}
```

---

## 4. NFR Req ↔ 패턴 매핑

| NFR Req | 만족시키는 패턴 |
|---|---|
| NFR-U2-R1 (영속화 트리거) | persistAndDispatch (FD) + P-U2-3 |
| NFR-U2-R2 (자동 복원) | P-U2-9 + bootRestore (FD) |
| NFR-U2-R3 (손상 처리) | P-U2-9 |
| NFR-U2-R4 (SaveResult 원자성) | 트랜잭션 BEGIN/COMMIT (FD §12) |
| NFR-U2-R5 (토큰 정확성) | P-U2-10 |
| NFR-U2-R6 (graceful shutdown) | P-U2-7 stopCh + Close (FD) |
| NFR-U2-R7 (WAL + synchronous) | P-U2-8 |
| NFR-U2-P1 (SaveSnapshot p99<50ms) | P-U2-1 + P-U2-2 + P-U2-8 |
| NFR-U2-P2 (SubmitAction p99<100ms) | P-U2-1 + P-U2-2 + P-U2-3 (실패 격리) |
| NFR-U2-M1~M6 | P-U2-4 (안정), P-U2-5 (표준 lib) |
| NFR-U2-S1~S3 | P-U2-10 (토큰), 0600 권한 (코드 단계) |
| NFR-U2-S4 (마스킹) | PrivateView 빌더 (FD §11) |
| NFR-U2-C1~C3 | P-U2-6 + P-U2-4 (panic recover) |

---

## 5. 안티패턴 (의식적 회피)

- ❌ 단일 GM 락 분할 (예: persistence 따로) — 일관성 위험
- ❌ 비동기 영속화 큐 — Q-FD-U2-2 결정과 충돌, 손실 위험
- ❌ Engine 호출을 락 외부에서 수행 — Engine 단일 스레드 가정 위반
- ❌ 핸들러 panic 전파 — SessionManager 종료 위험
- ❌ ad-hoc SQL 매 호출 prepare — p99 안정성 저하
- ❌ DB 파일을 사용자 홈 디렉터리 외부에 두기 — 권한 모델 복잡
- ❌ 토큰을 닉네임 기반으로 발급 — 추측 가능, NFR-S1 위반
