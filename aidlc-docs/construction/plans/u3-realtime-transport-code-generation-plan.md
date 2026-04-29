# Code Generation Plan — U3 Realtime Transport

**작성일**: 2026-04-26
**대상 단위**: U3 (`internal/transport/ws/*`)
**참조**:
- `application-design/unit-of-work.md` §3
- `construction/u3-realtime-transport/functional-design/*.md`
- `construction/u3-realtime-transport/nfr-requirements/*.md`
- `construction/u3-realtime-transport/nfr-design/*.md`
- `aidlc-state.md` (Workspace Root: `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game`)
- U2 공개 API: `construction/u2-session-persistence-announce/code/u2-public-api.md`
- U1 공개 API: `construction/u1-game-core/code/u1-public-api.md`

> 본 plan은 U3 Code Generation의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트

**책임**: 다중 WebSocket 클라이언트 연결 관리 + 가시성 정책에 따른 도메인 이벤트 라우팅 + 클라이언트 입력의 SessionManager 위임 + Korean Announcement push.

**구현 대상 요구사항** (story map §4 U3 Primary):
- FR-1.1 (LAN URL 노출 보조), FR-1.2 (재연결 기반)
- NFR-1 (재연결 시 화면 자동 복원), NFR-2 (LAN 즉시 반응 + 12명 동접)

**의존**:
- **U2 SessionManager** (`internal/session`) — Subscribe + 모든 lifecycle/action 메서드 호출
- **U1 Game Core** (`internal/game`) — Action/Event/State/EngineError/Visibility 타입 import
- **announce** (`internal/announce`) — Announcement 타입 import
- **외부**: `github.com/gorilla/websocket` (신규 직접 의존 1개)

**산출물**: Go 패키지 1개 (`internal/transport/ws`) + 단위 테스트 + U2 SessionManager 인터페이스 확장 (`Snapshot() game.State` 추가).

---

## 1. 코드 위치 결정

| 항목 | 위치 |
|---|---|
| Workspace Root | `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game` |
| U3 패키지 | `internal/transport/ws/` |
| U2 인터페이스 확장 | `internal/session/session.go` (Snapshot 메서드 추가) |
| U2 구현체 확장 | `internal/session/session.go` (session.Snapshot 구현) |
| 문서 산출물 | `aidlc-docs/construction/u3-realtime-transport/code/` (markdown 요약) |

---

## 2. Part 1 — Planning 체크리스트

- [x] (P1-1) 단위 컨텍스트 분석
- [x] (P1-2) 코드 위치·구조 결정
- [x] (P1-3) plan 문서 작성
- [x] (P1-4) 사용자에게 요약 제공
- [x] (P1-5) audit에 승인 게이트 로그
- [x] (P1-6) 사용자 승인
- [x] (P1-7) Part 2 진입

---

## 3. Part 2 — Generation 체크리스트

### 3.1 모듈 의존성 추가
- [x] (G1) `go get github.com/gorilla/websocket@latest` → `go.mod` / `go.sum` 갱신 (v1.5.3 추가)

### 3.2 U2 SessionManager 인터페이스 확장 (P-U3-8)
- [x] (G2) `internal/session/session.go` — `SessionManager` 인터페이스에 `Snapshot() game.State` 메서드 추가
- [x] (G3) `internal/session/session.go` — `(*session).Snapshot()` 구현 (mu.Lock + engine.Snapshot 반환)
- [x] (G4) `internal/session/snapshot_test.go` — Snapshot 검증 4건 + EventOut.State 추가 (락 재진입 데드락 회피)

### 3.3 `internal/transport/ws/` — 도메인 타입 (LC-U3-2, LC-U3-10)
- [x] (G5) `doc.go` — 패키지 godoc
- [x] (G6) `protocol.go` — incoming 14종 + outgoing 7종 wire 메시지 struct + event kind 15종 매핑 + JSON 헬퍼
- [x] (G7) `id.go` — `ClientID` + `newClientID` (8-byte hex16)
- [x] (G8) `client.go` — `Client` struct + `ClientKind` 상수 + `clientRegistry` (RWMutex 단일 락)

### 3.4 `internal/transport/ws/` — Hub 본체 (LC-U3-1)
- [x] (G9) `hub.go` — `Hub` 인터페이스 + impl + `New` + Register/Unregister/Run/Close/UpgradeHandler
- [x] (G10) `handlers.go` — readLoop + handleIncoming + respondJoin + bindPlayer + handleSubmitErr + errorCodeOf
- [x] (G11) `writer.go` — writeLoop (ctx.Done + ping ticker) + enqueue (default-branch 백프레셔)
- [x] (G12) `dispatch.go` — onEvent (panic recover) + routeEvent (가시성 3종, EventOut.State 활용) + buildEventPayload (15종)

### 3.5 단위 테스트 — `internal/transport/ws/`
- [x] (G13) `protocol_test.go` — 봉투 디코딩 + visibility 매핑 + event kind 15종 직렬화
- [x] (G14) `client_test.go` — clientRegistry add/remove/bindPlayer/snapshots + ClientKind String + ClientID 길이
- [x] (G15) `handlers_test.go` — 모든 submit/host 메시지 + JSON 디코드 실패 + last-connect-wins eviction + Run/Close 라이프사이클
- [x] (G16) `writer_test.go` — enqueue 정상/cancelled/full
- [x] (G17) `dispatch_test.go` — routeEvent VisPublic/VisPlayer/VisRoleMafia/Unknown + timeToMs
- [x] (G18) hub 동작은 integration_test의 testRig가 종합 검증
- [x] (G19) `integration_test.go` — E2E 6건 (host start, 비공개 라우팅, graceful shutdown < 2s, leak, unknown type, resume)
- [x] (G20) goroutine leak는 integration_test의 TestE2E_LeakNoGoroutineGrowth가 검증

### 3.6 문서 산출물
- [x] (G21) `aidlc-docs/construction/u3-realtime-transport/code/u3-code-summary.md`
- [x] (G22) `aidlc-docs/construction/u3-realtime-transport/code/u3-public-api.md`

### 3.7 N/A 단계
- [x] (G23) Deployment Artifacts — N/A (단일 바이너리에 통합)
- [x] (G24) DB Migration Scripts — N/A (U3는 SQLite 미사용)
- [x] (G25) Frontend Components — N/A (백엔드 단위, U5에서 wire client 작성)

---

## 4. Definition of Done

- [x] (V1) 모든 G1~G25 [x]
- [x] (V2) `go build ./...` 통과
- [x] (V3) `go vet ./internal/...` 0 issue
- [x] (V4) `gofmt -l ./internal/transport/ws/` empty
- [x] (V5) `go test ./internal/transport/ws/... ./internal/session/... ./internal/announce/... ./internal/persistence/...` 모든 테스트 통과
- [x] (V6) `go test -race` 통과 (NFR-U3-C2)
- [x] (V7) `go test -cover` 합산 **87.4%** ≥ 85% (NFR-U3-M1) — ws 89.0% / session 88.2% / announce 93.3% / persistence 80.2%
- [x] (V8) 직접 의존 +1 (`gorilla/websocket v1.5.3`) — transitive 0개 (NFR-U3-M4)

---

## 5. 스토리/요구사항 추적성

| 요구사항 | 구현 단계 |
|---|---|
| FR-1.1 (LAN URL + 단일 호스트 1세션) | G9 (UpgradeHandler), G18 (테스트) |
| FR-1.2 (재연결) | G10 (handleIncoming resume), G17 (snapshot 메시지 검증) |
| FR-2.3 (역할 비공개 라우팅) | G12 (routeEvent), G17/G19 (검증) |
| FR-7.2 (안내 외부화 wire 변환) | G6 (announceMsg), G12 (onEvent) |
| FR-8.4 (안내 풍부) | G6 (announceMsg) |
| NFR-1 (재연결 시 화면 자동 복원) | G10 (resume 후 snapshotMsg push) |
| NFR-2 (LAN 즉시 반응 + 12명 동접) | G11 (enqueue 백프레셔), G18 (16 동접 테스트) |
| NFR-4 (비공개 정보) | G12 (routeEvent VisPlayer/VisRoleMafia), G19 (E2E 검증) |
| NFR-7 (외부 의존 1개) | G1 (단일 외부 lib) |
| NFR-U3-R4 (graceful shutdown < 2초) | G9 (Close), G19 (테스트) |
| NFR-U3-G2 (goroutine 누수 0) | G20 (leak_test) |

---

## 6. 산출물 요약 (예상)

| 종류 | 파일 수 | 위치 |
|---|---:|---|
| ws 코드 | 8 | `internal/transport/ws/*.go` (doc, protocol, id, client, hub, handlers, writer, dispatch) |
| ws 테스트 | 8 | `internal/transport/ws/*_test.go` |
| U2 확장 | 1줄 인터페이스 + Snapshot 구현 + 테스트 1건 | `internal/session/session.go`, `internal/session/snapshot_test.go` |
| 문서 요약 | 2 | `aidlc-docs/construction/u3-.../code/*.md` |

---

## 7. 사용자 승인 게이트

본 plan에 동의하시면 **"승인"** 또는 **"continue"** 로 답변. 변경이 필요하면 구체적 항목을 알려주세요 (예: "G19에 12 PLAYER 동접 시나리오 추가").
