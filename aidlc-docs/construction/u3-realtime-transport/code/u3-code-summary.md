# U3 Realtime Transport — Code Summary

**작성일**: 2026-04-26
**대상 단위**: U3 (`internal/transport/ws`)
**plan**: `aidlc-docs/construction/plans/u3-realtime-transport-code-generation-plan.md`

---

## 1. 빌드 / 검증 결과

| 게이트 | 결과 |
|---|---|
| `go build ./...` | ✅ 통과 |
| `go vet ./internal/...` | ✅ 0 issue |
| `gofmt -l ./internal/transport/ws/` | ✅ empty |
| `go test ./internal/transport/ws/...` | ✅ 모든 테스트 통과 |
| `go test -race ./...` | ✅ 통과 (NFR-U3-C2) |
| 라인 커버리지 (4 패키지 합산) | ✅ **87.4%** ≥ 85% (NFR-U3-M1 / NFR-U2-M1 모두) |
| · ws | 89.0% |
| · session | 88.2% |
| · announce | 93.3% |
| · persistence | 80.2% |
| 외부 직접 의존 | ✅ +1: `github.com/gorilla/websocket v1.5.3` (NFR-U3-M4) |

---

## 2. 산출 파일 인벤토리

### 2.1 `internal/transport/ws/` (8 코드 + 6 테스트)

| 파일 | 책임 | LC |
|---|---|---|
| `doc.go` | 패키지 godoc | — |
| `protocol.go` | 와이어 메시지 타입(incoming 14종 + outgoing 7종) + event kind 15종 매핑 + JSON 헬퍼 | LC-U3-10 |
| `id.go` | `ClientID` 타입 + `newClientID` (8-byte hex16) | LC-U3-11 |
| `client.go` | `Client` struct + `ClientKind` + `clientRegistry` (RWMutex 단일 락) | LC-U3-2/3 |
| `hub.go` | `Hub` 인터페이스 + `hub` impl + `New` + `Register`/`Unregister`/`Run`/`Close`/`UpgradeHandler` | LC-U3-1 |
| `handlers.go` | `readLoop` + `handleIncoming` (14 type 디스패치) + `respondJoin` + `bindPlayer` + `handleSubmitErr` + `errorCodeOf` | LC-U3-4/6 |
| `writer.go` | `writeLoop` (ctx.Done + ping ticker) + `enqueue` (default-branch 백프레셔) | LC-U3-5/9 |
| `dispatch.go` | `onEvent` (Subscribe 핸들러, panic recover) + `routeEvent` (가시성 3종) + `buildEventPayload` (event kind 15종) | LC-U3-7/8 |
| `protocol_test.go` | 봉투 디코딩 / visibility / event kind 직렬화 | — |
| `client_test.go` | clientRegistry add/remove/bindPlayer/snapshots + last-connect-wins | — |
| `writer_test.go` | enqueue 정상/cancelled/full | — |
| `dispatch_test.go` | routeEvent VisPublic/VisPlayer/VisRoleMafia/Unknown + timeToMs | — |
| `handlers_test.go` | 모든 submit/host 메시지 타입 + JSON 디코드 실패 + last-connect-wins eviction + Run/Close 라이프사이클 | — |
| `integration_test.go` | E2E httptest + WebSocket dialer (host start, 비공개 라우팅, snapshot, graceful shutdown < 2s, goroutine leak) | — |

### 2.2 U2 인터페이스 확장 (1 코드 추가 + 1 테스트)
| 파일 | 변경 |
|---|---|
| `internal/session/session.go` | `SessionManager` 인터페이스에 `Snapshot() game.State` 추가 + `(*session).Snapshot()` 구현 |
| `internal/session/types.go` | `EventOut`에 `State game.State` 필드 추가 (Subscribe 핸들러가 락 재진입 없이 상태 조회) |
| `internal/session/action.go` | `persistAndDispatch`가 EventOut에 state를 채움 |
| `internal/session/snapshot_test.go` | 신규 — Snapshot 검증 4건 (zero/post-start/race/clone) |

총 **8 Go 코드** + **6 ws 테스트** + **U2 확장 + 1 테스트** + **본 문서 2종**.

---

## 3. 스토리/요구사항 ↔ 구현 매핑

| 요구사항 | 구현 위치 |
|---|---|
| FR-1.1 (LAN URL + 단일 호스트) | `hub.UpgradeHandler` (U4가 호출) |
| FR-1.2 (재연결) | `handlers.respondJoin` resume 분기 + snapshot 메시지 push |
| FR-2.3 (역할 비공개) | `dispatch.routeEvent` VisPlayer/VisRoleMafia |
| FR-7.2 (안내 외부화 wire 변환) | `dispatch.onEvent` Announcement → wire announceMsg |
| FR-8.4 (안내 풍부) | onEvent + announceMsg 송신 |
| NFR-1 (재연결 시 화면 자동 복원) | resume 직후 snapshotMsg push |
| NFR-2 (LAN 즉시 반응 + 12명 동접) | `enqueue` 백프레셔 + integration test 16 동접 검증 |
| NFR-4 (비공개 정보) | `routeEvent` + `protocol` (DEBUG 로그 type만) |
| NFR-7 (외부 서비스 0) | `gorilla/websocket` 단일 외부 의존 |
| NFR-U3-R4 (graceful shutdown < 2초) | `Hub.Close` + `TestE2E_GracefulShutdownUnder2Seconds` |
| NFR-U3-G2 (goroutine 누수 0) | `TestE2E_LeakNoGoroutineGrowth` 50회 connect/disconnect |

---

## 4. 핵심 설계 결정 (재확인)

| 결정 | 위치 |
|---|---|
| 단일 RWMutex (P-U3-1) | `clientRegistry.mu` |
| 짧은 RLock onEvent (P-U3-2) | `dispatch.onEvent` + `clientRegistry.snapshot*` |
| SubmitAction 직접 호출 (P-U3-3) | `handleIncoming` → `mgr.SubmitAction` |
| ctx.Done writeLoop 종료 (P-U3-4) | `writer.writeLoop` (close(c.Out) 미사용) |
| 단일 GM 락 위임 (NFR-U3-C1) | Hub 자체 SubmitAction 락 0개 |
| **EventOut.State 추가** | `SessionManager.Snapshot()` 재진입 데드락 회피 — VisRoleMafia 라우팅용 |
| last-connect-wins (P-U3-7) | `bindPlayer` + `handlers_test.go` E2E 검증 |
| net.Pipe in-memory 통합 테스트 (P-U3-10) | `integration_test.go` httptest.NewServer |

> **EventOut.State 추가 결정 배경**: NFR Design 단계에서 `SessionManager.Snapshot()`을 추가하기로 했으나, Subscribe 핸들러가 GM 락 안에서 호출되므로 onEvent에서 다시 `Snapshot()`을 호출하면 동일 mutex 재진입 데드락 발생. 해결: U2가 dispatch 시점의 state를 EventOut에 동봉, Hub는 락 재진입 없이 사용. 외부 호출자용 `Snapshot()` 메서드도 그대로 유지(추가 메서드 1개).

---

## 5. 알려진 제한 / 후속 작업

| 항목 | 상태 |
|---|---|
| `Hub.Run`의 ctx 처리 | 단순 select — 코드 단계 OK |
| `host:create-session` vs `join` 첫 호출 분기 정책 | 호스트가 자동으로 첫 호출자가 되도록 별도 인증 없음 (PoC) |
| WebSocket compression (permessage-deflate) | NFR-U3 비-요구사항 — 적용 안 함 |
| protocolVersion validation | 정보용만 — Q-FD-U3-13=B |
| Composition Root (Hub + SessionManager + HTTPServer 와이어링) | U4 단계에서 `cmd/mafia-game/main.go` 작성 예정 |

---

## 6. 변경된 모듈 메타데이터

`go.mod`:
- 신규 직접 의존: `github.com/gorilla/websocket v1.5.3`
- transitive 의존 추가 없음 (gorilla/websocket는 표준 lib만 사용)

> 직접 의존 누계: `modernc.org/sqlite` (U2) + `github.com/gorilla/websocket` (U3) = 2개. 모두 NFR-7 단위별 1개 정책 만족.
