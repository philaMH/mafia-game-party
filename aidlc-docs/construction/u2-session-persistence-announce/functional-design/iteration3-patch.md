# U2 Session/Persistence/Announce — Functional Design Iteration 3 Patch

**문서 버전**: 1.0
**작성일**: 2026-04-29
**기준 산출물**: `iteration2-patch.md`, `business-logic-model.md` v1
**상위 변경 명세**: `audit.md` Iteration 3 (사용자 옵션 A)
**처리 방식**: 변경분만 명시. 기존 v1/v2 인터페이스·자료구조 보존, 신규 메서드 1건 추가.

---

## 1. 변경 요약

| ID | 종류 | 설명 |
|---|---|---|
| **S3-1** | 신규 인터페이스 메서드 | `SessionManager.RoomSnapshot() RoomSnapshot` — 방·게임 메타 + 엔진 State를 atomic 단일 호출로 노출 |
| **S3-2** | 신규 자료구조 | `RoomSnapshot` 구조체 (export) |
| **불변** | 기존 | `Snapshot()` 포함 모든 v1/v2 메서드, `Session` 필드, `JoinPlayer`/`OpenRoom`/`HostStartGame` 게이트 로직 모두 보존 |

> **비고**: 본 패치는 기존 `Snapshot()` API를 변경하지 않는다. 호출자(현행 U3 `routeEvent`)는 기존 시그니처를 그대로 사용한다.

---

## 2. 자료구조

```go
// RoomSnapshot is a frozen view of the SessionManager's room and game
// state, captured atomically under the GM lock. Used by U3 to push the
// current state to a freshly registered WebSocket client (late-joiner
// resync). All fields are deep-copied; callers may mutate freely.
type RoomSnapshot struct {
    // RoomOpened mirrors Session.RoomOpened.
    RoomOpened bool

    // Options is the host-configured game options captured at OpenRoom
    // time. Zero value when RoomOpened == false.
    Options game.Options

    // GameStarted reports whether the engine has progressed past LOBBY.
    // True when State.Phase is one of INTRO/NIGHT/DAY/VOTE/RECOUNT/END.
    GameStarted bool

    // State is a deep copy of the engine state. Always present (engine
    // returns a zero State when no game is active).
    State game.State

    // HostOccupied reports whether the host seat is currently held by a
    // live token holder. Used by /public late-joiners to pre-disable the
    // claim form.
    HostOccupied bool
}
```

배치: `internal/session/types.go` (기존 `Session`/`JoinResult`/`EventOut` 인접).

---

## 3. 메서드 명세

### 3.1 `RoomSnapshot() RoomSnapshot`

**시그니처**:
```go
func (s *session) RoomSnapshot() RoomSnapshot
```

**동작 (의사코드)**:
```
acquire s.mu (Lock)
defer release s.mu

state := s.engine.Snapshot()                       // 이미 deep copy 반환
opts  := s.sess.PendingOptions
roomOpened := s.sess.RoomOpened
gameStarted := isActivePhase(state.Phase)          // 기존 helper 재사용
hostOccupied := s.hostAuth.IsClaimed()             // 신규 helper (3.2 참조)

return RoomSnapshot{
    RoomOpened:   roomOpened,
    Options:      opts,
    GameStarted:  gameStarted,
    State:        state,
    HostOccupied: hostOccupied,
}
```

**동시성**: 기존 `Snapshot()`과 동일한 GM mutex 안에서 실행. 다른 호출자(SubmitAction, Subscribe 콜백 등)가 진행 중이면 차단됨. 전체 임계 구역은 O(필드 복사) 수준이라 추가 지연 무시 가능.

**의도**:
- 단일 호출 = 단일 lock acquire = atomic 일관성. (`RoomOpened` 읽고 → 별도 호출로 `Snapshot()` 읽으면 그 사이 phase가 바뀔 수 있음)
- 호출자(U3 hub)는 lock-free로 결과를 사용 가능 (deep copy).

### 3.2 `hostAuthority.IsClaimed() bool` (private, U2 내부)

**시그니처**:
```go
func (h *hostAuthority) IsClaimed() bool
```

**동작**:
- `hostAuthority` 내부 mutex로 현재 토큰 유효 여부 확인 후 반환.
- `Claim`/`Release`/`Validate`와 동일한 락 전략 재사용.

**근거**: `RoomSnapshot`에서 atomic 일관성을 보장하기 위해 별도 lock 없이 caller(s.session)의 GM mutex 보호 하에 호출한다.

> **대안 검토**: `hostAuthority`에 새 메서드 추가 대신 `s.hostAuth.activeToken != ""` 비교 가능. 하지만 hostAuthority 캡슐화 깨므로 메서드 신설을 채택. mutex 중첩(GM lock + hostAuth lock) 위험은 없음 — hostAuthority의 어떤 메서드도 GM lock을 다시 잡지 않는다.

---

## 4. 영향 분석

### 4.1 인터페이스 변경
- `SessionManager` 인터페이스에 `RoomSnapshot()` 1개 추가. mock/스텁 사용 테스트 식별·갱신 필요:
  - `internal/transport/ws/*` 테스트의 `fakeSessionManager`/`recordingMgr` 등 — 메서드 추가 stub 필요.
  - `internal/transport/http/*` 테스트의 mock — 동일.
  - 외부 의존자 없음 (단일 모듈).

### 4.2 데이터 마이그레이션
- 없음. 기존 `Session.PendingOptions`/`RoomOpened` 필드 그대로 재사용.

### 4.3 영속성/Restore
- 변경 없음. `RoomSnapshot`은 read-only view이며 영속화 대상 아님.

---

## 5. 단위 테스트 추가 계획

위치: `internal/session/iteration3_test.go` (신규).

| ID | 테스트명 | 검증 |
|---|---|---|
| S3-T1 | `TestRoomSnapshot_BeforeOpenRoom` | `RoomOpened=false`, `GameStarted=false`, `Options=zero`, `HostOccupied=false` |
| S3-T2 | `TestRoomSnapshot_AfterClaimBeforeOpen` | `RoomOpened=false`, `HostOccupied=true` |
| S3-T3 | `TestRoomSnapshot_AfterOpenRoom` | `RoomOpened=true`, `Options.MafiaCount==2`, `HostOccupied=true`, `GameStarted=false` |
| S3-T4 | `TestRoomSnapshot_AfterHostStartGame` | `GameStarted=true`, `State.Phase==PhaseIntro`, `RoomOpened=true` |
| S3-T5 | `TestRoomSnapshot_AfterReleaseHost` | `HostOccupied=false`, `RoomOpened` 직전 상태 유지 (`OpenRoom` 후 `ReleaseHost` 시) |
| S3-T6 | `TestRoomSnapshot_StateIsDeepCopy` | 반환된 `State.Players`를 변경해도 다음 호출의 `State.Players`에 영향 없음 |

기존 `TestSubscribe_*`, `TestOpenRoom_*` 등 회귀 영향 없음 (인터페이스만 add, 동작 미변경).

---

## 6. 커버리지 목표

- 본 단위 (`internal/session`) 현행 87.4% 동등 이상.
- 신규 메서드 100% (위 6개 테스트로 커버).

---

## 7. Out of Scope

- `Snapshot()` 시그니처/시맨틱 변경 — 기존 호출자 영향 회피.
- `RoomSnapshot` 영속화 — 메모리 read-only view에 한정.
- `RoomSnapshot` 변경 통지(이벤트) — Iteration 3 범위는 polling-on-register 1회만.
- 호스트 토큰 노출 — `RoomSnapshot`은 토큰 자체를 포함하지 않는다 (`HostOccupied bool`만).

---

## 8. 추적성

| 패치 ID | Iteration 3 사용자 입력 | U3 패치 연결 |
|---|---|---|
| S3-1, S3-2 | "방을 연 뒤 새 클라이언트가 붙는 시나리오도 고려" + "A" | `u3-realtime-transport/functional-design/iteration3-patch.md` W3-1 ~ W3-3 |
