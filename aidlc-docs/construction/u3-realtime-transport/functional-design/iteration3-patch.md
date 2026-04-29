# U3 Realtime Transport — Functional Design Iteration 3 Patch

**문서 버전**: 1.0
**작성일**: 2026-04-29
**기준 산출물**: `iteration2-patch.md`, `wire-protocol.md`
**상위 변경 명세**: `u2-session-persistence-announce/functional-design/iteration3-patch.md` (RoomSnapshot API)
**처리 방식**: 변경분만 명시. 신규 wire 메시지 0건 (기존 `room:opened` / `snapshot` / `room:host-occupied` 재사용).

---

## 1. 변경 요약

| ID | 종류 | 설명 |
|---|---|---|
| **W3-1** | Register 후처리 | `Register()`가 `welcome` enqueue 직후 `mgr.RoomSnapshot()` 조회 → 결과에 따라 0 ~ 2건 추가 메시지를 본 클라이언트에만 enqueue |
| **W3-2** | 단일 클라이언트 push 헬퍼 | `dispatch.go`에 `pushRoomState(c *Client, snap session.RoomSnapshot)` 신규 추가 — broadcast 헬퍼와 분리 |
| **W3-3** | snapshot 메시지 PUBLIC variant | 신규 PUBLIC 클라이언트(=PlayerID 없음)에 보낼 때 `your` 필드는 zero-value `yourInfo{}` 로 채움. `IsHost=false` (host 토큰 미보유). 기존 `snapshotMsg` 구조체/타입 변경 없음 |
| **불변** | 기존 wire | `room:opened`/`snapshot`/`room:host-occupied`/`welcome`/`event` 등 모두 보존, broadcast 경로 변경 없음 |
| **불변** | hub 인터페이스 | `Hub.Register/Unregister/UpgradeHandler/Run/Close` 시그니처 보존 |

---

## 2. 시퀀스

```
Client TCP connect
  │
  ▼
hub.UpgradeHandler ── upgrader.Upgrade ─► hub.Register(conn)
                                              │
                                              │ newClient + registry.add
                                              ▼
                                          enqueue(welcome)         ◄── 기존
                                              │
                                              │ snap := mgr.RoomSnapshot()  ◄── W3-1 신규
                                              ▼
                                          pushRoomState(c, snap)   ◄── W3-2 신규
                                              │     ├─ snap.RoomOpened   → enqueue(room:opened, options)
                                              │     ├─ snap.GameStarted  → enqueue(snapshot, state, your=zero, isHost=false)
                                              │     └─ snap.HostOccupied → enqueue(room:host-occupied)
                                              │
                                              ▼
                                          go readLoop / writeLoop
```

조건부 push 정책 (의사코드):
```go
func pushRoomState(c *Client, snap session.RoomSnapshot) {
    if snap.RoomOpened {
        enqueue(c, mustMarshal(roomOpenedMsg{Type: TypeRoomOpened, Options: snap.Options}))
    }
    if snap.GameStarted {
        enqueue(c, mustMarshal(snapshotMsg{
            Type:   TypeSnapshot,
            State:  snap.State,
            IsHost: false,
            Your:   yourInfo{},
        }))
    }
    if snap.HostOccupied {
        enqueue(c, mustMarshal(roomHostOccupiedMsg{Type: TypeRoomHostOccupied}))
    }
}
```

순서 근거:
1. `room:opened` 먼저 — 클라이언트 reducer에서 `roomOpened=true`가 되어야 후속 `snapshot`이 의미를 가짐(기존 `applySnapshot`은 둘을 독립 처리하지만 일관된 UX 진행).
2. `snapshot`은 진행 중 게임에만 — LOBBY/END 페이즈에서는 의미 없는 빈 state가 될 수 있어 skip.
3. `room:host-occupied`는 가장 늦게 — host 좌석 점유 상태는 `/public` 클라이언트의 폼 비활성에만 영향, 게이트 진행을 차단하지 않음.

`snap.RoomOpened == false && snap.HostOccupied == true` 조합도 발생 가능 (claim 후 open 전). 이 경우 `room:host-occupied` 1건만 발송.

---

## 3. 영향 분석

### 3.1 Hub 코드
- `hub.Register()`: welcome enqueue 직후 `pushRoomState(c, h.mgr.RoomSnapshot())` 호출.
  - `mgr.RoomSnapshot()`은 GM lock을 잡으므로 readLoop/writeLoop 시작 *전에* 호출해야 lock 해제 보장. (실제로는 메서드가 동기 반환 후 락 해제하므로 무관하나, 명시적으로 register 동기 경로에 둠.)
- `dispatch.go`: `pushRoomState` 헬퍼 추가. 기존 `broadcastRoomOpened`는 그대로.

### 3.2 데이터 race
- `Register`는 `h.registry.add(c)` 직후 → readLoop/writeLoop 고루틴 시작 *전*까지 메인 고루틴이 enqueue 가능. 기존 welcome도 동일 패턴이라 추가 race 없음.
- `mgr.RoomSnapshot()`은 GM lock 안에서 deep copy 반환. hub의 `h.registry`/`c` lock과 직교.

### 3.3 메시지 중복/순서
- 기존 `broadcastRoomOpened`은 *모든* 등록된 클라이언트(PUBLIC + PLAYER)에 송신. Register-time push는 **방금 등록된 클라이언트** 1명에게만 송신 → 중복 가능 시점은 `OpenRoom` 진행 중 신규 등록이 끼어드는 마이크로초 단위뿐. broadcast 측은 `registry.snapshotPublic/snapshotPlayers`로 그 시점 등록된 클라이언트 목록을 잡으므로:
  - 신규 클라이언트가 broadcast 직전에 등록 → broadcast가 그 클라이언트도 포함 → register-time push도 발생 → 동일 메시지 2회 수신 가능.
  - 신규 클라이언트가 broadcast 직후 등록 → broadcast 미포함, register-time push만 발생.
- 클라이언트 reducer는 `room:opened`를 idempotent하게 처리 (`roomOpened=true` 단순 설정). 따라서 2회 수신 시에도 부작용 없음. 이 분석을 단위 테스트로 직접 입증할 필요는 낮음(별도 race 테스트 OOS) — 본 패치 단위 테스트는 결과 idempotency만 확인.

### 3.4 기존 핸들러
- `subscribe:public` no-op 유지 (변경 없음).
- `Resume` 핸들러의 snapshot push 로직 (handlers.go:95-105) 변경 없음 — Resume은 PLAYER 컨텍스트에서 `Your` 정보 포함 snapshot을 보내고, 본 패치는 PUBLIC 컨텍스트에서 zero `Your` snapshot을 보냄. 두 경로 분리 유지.

---

## 4. 단위 테스트 추가 계획

위치: `internal/transport/ws/iteration3_test.go` (신규).

| ID | 테스트명 | 검증 |
|---|---|---|
| W3-T1 | `TestIter3_Register_BeforeOpenRoom_NoExtraMessages` | 방 미개설 상태에서 새 클라이언트 등록 시 `welcome`만 수신 (room:opened/snapshot/host-occupied 없음) |
| W3-T2 | `TestIter3_Register_AfterClaimBeforeOpen_PushesHostOccupied` | host:claim 후 새 클라이언트 등록 시 `welcome` + `room:host-occupied` 수신 |
| W3-T3 | `TestIter3_Register_AfterOpenRoom_PushesRoomOpened` | OpenRoom 후 새 클라이언트 등록 시 `welcome` + `room:opened` (옵션 일치) + `room:host-occupied` 수신 |
| W3-T4 | `TestIter3_Register_AfterHostStartGame_PushesSnapshot` | HostStartGame 후 새 클라이언트 등록 시 `welcome` + `room:opened` + `snapshot` (state.Phase==INTRO, your=zero, isHost=false) + `room:host-occupied` 수신 |
| W3-T5 | `TestIter3_Register_PushOrder` | 위 시나리오 4에서 메시지 수신 순서가 welcome → room:opened → snapshot → room:host-occupied 임을 확인 |

기존 `TestIter2_HostOpenRoom_BroadcastsRoomOpened` 등 회귀 영향 없음 (broadcast 경로 그대로).

---

## 5. 클라이언트(U5) 영향

- 기존 reducer (`web/src/context/reducer.ts`)는 이미 `room:opened`/`snapshot`/`room:host-occupied`를 처리. 추가 변경 없음.
- 단, `snapshot.your`가 zero-value(`role: ""`, `keyword: ""`, `team: ""`, `mafiaCohort: null`)로 도착할 때 `applySnapshot`이 깨지지 않는지 회귀 검증 필요.
  - 현행 reducer는 `your`를 단순 대입하므로 zero 값도 무해. 별도 단위 테스트 1건 추가 권장: `applies zero-your snapshot to PUBLIC client without crash` (선택적).

본 패치 자체는 **U5 코드 변경 0건**.

---

## 6. 커버리지 목표

- 본 단위 (`internal/transport/ws`) 현행 87.0% 동등 이상.
- 신규 4개 테스트 (T1~T4) + 순서 1개 (T5) 추가 → 등록 경로의 새 분기 100% 커버.

---

## 7. Out of Scope

- `event` 재전송/replay — Iteration 3은 register-time *one-shot* 동기화에 한정. 게임 진행 도중 누락된 개별 event는 `snapshot`이 충분히 커버.
- 클라이언트 측 reconnect 시 자동 동기화 — 기존 `Resume` 경로가 담당.
- WebSocket 메시지 압축/배치 — 기존 그대로.
- PUBLIC ↔ PLAYER 변환 시 RoomSnapshot 재전송 — 변환은 join/resume 응답 경로에서 별도 처리.

---

## 8. 추적성

| 패치 ID | 의존 |
|---|---|
| W3-1 ~ W3-3 | U2 S3-1, S3-2 (`SessionManager.RoomSnapshot`/`session.RoomSnapshot`) |
| 검증 | Chrome DevTools MCP `방 개설 → 새 /play 탭 접속 → 입장 가능` 시나리오 회귀 (Build & Test 단계) |
