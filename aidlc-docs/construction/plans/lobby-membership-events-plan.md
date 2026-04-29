# Plan — LOBBY Membership Events (옵션 A 정공법)

**작성일**: 2026-04-27
**유형**: Post-Construction Maintenance / Cross-Unit Change
**범위**: U1 Game Core, U2 Session, U3 Realtime Transport, U5 Web Frontend
**상태**: **코드 수정 완료 (2026-04-27 후속 세션)** — 모든 Stage A~E DoD 통과, 사용자 승인 대기

---

## 0. 동기 (Why)

Chrome DevTools MCP를 사용한 6+1명 LAN 시나리오 검증 (`/play` 6 탭 + `/public` 1 탭) 결과 다음 결함이 재현됨:

- LOBBY 단계에서 GM(`/public`) 화면이 영구히 "플레이어 입장을 기다리는 중…" 으로 머무름.
- `internal/transport/ws/dispatch.go:13` `onEvent`는 `game.Event`만 broadcast 하지만, `internal/session/lifecycle.go` 의 `CreateSession`/`JoinPlayer`는 어떤 도메인 이벤트도 발행하지 않음 → 다른 클라이언트는 합류를 통보받을 방법이 없음.
- 호스트의 "게임 시작" 버튼은 `ctx.state` 가 채워져야 렌더되지만 (`web/src/views/PublicView/PublicView.tsx:65`) LOBBY에서 state가 비어 있어 UI만으로는 게임 시작이 불가능.
- 임시 검증 시 `host:start` 메시지를 살아있는 다른 탭에서 임시 WebSocket으로 직접 전송해야 했음 (workaround, ship 불가).

**왜 옵션 A (도메인 이벤트 추가) 인가**:
옵션 B (transport-only snapshot broadcast) 는 변경 라인 수는 작지만 LOBBY 멤버십이라는 1급 도메인 사실을 transport 레벨에 숨겨 두므로, 추후 persistence/announce 카탈로그/리플레이가 합류 사건을 인지해야 할 때 부채가 됨. 옵션 A는 U1 도메인 모델에 사실을 명시함으로써 모든 단위가 같은 진실을 공유하게 하고, 기존 dispatch 파이프라인(`persistAndDispatch`)에 자연스럽게 얹힘. 외부 의존 0, 단위 경계 보존.

---

## 1. 영향 분석 (Affected Files)

| Unit | 파일 | 변경 종류 |
|---|---|---|
| U1 | `internal/game/event.go` | 새 sealed Event 타입 `PlayerJoined` (VisPublic) 추가 |
| U1 | `internal/game/types.go` | `Player` 또는 `State` 에 LOBBY 표현 보강 (아래 §2 결정 필요) |
| U2 | `internal/session/lifecycle.go` | `CreateSession`/`JoinPlayer` 가 envelope 생성 후 `persistAndDispatch` 호출, `emptyLobbyState` → `lobbyStateFromMembers` |
| U2 | `internal/session/types.go` | (필요 시) lobby snapshot helper |
| U3 | `internal/transport/ws/dispatch.go` | `buildEventPayload` 에 `PlayerJoined` 케이스 추가 |
| U3 | `internal/transport/ws/protocol.go` | `eventPayload` 에 `Name` (또는 lobby 멤버 정보) 필드 추가 |
| U5 | `web/src/types/wire.ts` | `EventPayload` union 에 `PlayerJoined` 추가, `Player` 타입의 LOBBY 호환 검토 |
| U5 | `web/src/context/reducer.ts` | `PlayerJoined` 핸들러 → `state.players` push |
| U5 | `web/src/views/PublicView/PlayersGrid.tsx` | LOBBY 단계 렌더 점검 (역할 미공개 표시) |
| U5 | `web/src/views/PlayerView/*.tsx` | LOBBY 진입 후 자기 이름 + 명단 표시 (선택) |

테스트:
- U1: `internal/game/event_test.go` 또는 새 `lobby_event_test.go` — 이벤트 직렬화/sealed 보장
- U2: `internal/session/lifecycle_test.go` 에 broadcast 검증 (Subscribe 후 host create + 1명 join 시 envelope 2개 도착)
- U3: `internal/transport/ws/dispatch_test.go` 에 PlayerJoined wire 변환 케이스
- U5: `web/src/context/reducer.test.ts` 에 PlayerJoined → players reducer 검증
- 통합: `internal/transport/ws/integration_test.go` 에 6명 합류 시 모든 PUBLIC+PLAYER 가 매번 PlayerJoined 수신 + GM 시작 가능 케이스

---

## 2. 설계 결정 (다음 세션 시작 전에 사용자 확정 필요)

### Q1. LOBBY 멤버를 어떻게 `game.State` 에 표현할 것인가?

옵션 1 — **기존 `Player` 재사용**:
- `Role` 을 빈 문자열, `Alive=true` 로 둔 `Player` 를 LOBBY 단계 동안 `State.Players` 에 채움.
- 장점: web/PlayersGrid 가 그대로 동작.
- 단점: `Role` 빈 값이 invariant 검사를 깰 수 있음 (`internal/game/validation.go` 점검 필요).

옵션 2 — **새 필드 `State.LobbyMembers []LobbyMember`**:
- LOBBY 전용 슬라이스 분리, `LobbyMember{ID, Name, JoinedAt}`.
- 장점: 게임 시작 후 `Players` 와 명확히 분리.
- 단점: web/PlayersGrid 분기 추가 필요.

**추천**: 옵션 1 — 변경 면적이 적고 web 단의 분기를 줄임. validation.go 의 phase==LOBBY 분기에서 role-empty 허용으로 한 줄 추가.

### Q2. `PlayerJoined` 발행 위치 — game vs session?

`game.Engine` 은 `StartGame()` 부터 가동되므로 LOBBY는 엄밀히 도메인 외부 상태. 그러나 `Event` 와 `EventEnvelope` 타입은 U1 소속이므로 새 이벤트 타입은 U1 에 추가. **발행자**는 U2 의 `session.CreateSession`/`JoinPlayer` 에서 직접 envelope 생성 후 `persistAndDispatch`. game.Engine 인스턴스는 호출하지 않음.

→ U1 변경은 타입 정의만, 로직은 U2.

### Q3. `PlayerLeft` 도 함께 도입?

현재 LOBBY 에서 멤버 탈퇴 흐름은 정의돼 있지 않음 (FR 검토 필요). **이번 변경에서는 제외**. 추후 별도 plan.

---

## 3. 단계별 작업 항목 (다음 세션에서 체크박스 채움)

### Stage A — U1 Game Core
- [x] `event.go` 에 `PlayerJoined struct { sealedEvent; PlayerID PlayerID; Name string }` 추가
- [x] `validation.go` LOBBY 단계 role-empty 허용 분기 (Q1 옵션 1 채택 시) — **N/A**: U1 의 `validateOptions` 는 옵션만 검사하고 LOBBY 단계 State 는 U2 가 구성하므로 분기 불필요. U1 markers/event 테스트로 sealed 보장만 추가.
- [x] `event_test.go` sealed 보장 + 직렬화 평등 테스트 (`TestPlayerJoinedFields`, `TestPlayerJoinedEnvelopePublic`, `markers_test.go::TestEventInterfaceImplementations`)
- [x] `go test ./internal/game/...` 통과 + 커버리지 ≥ 90 % 유지 — **실측 90.4 %**

### Stage B — U2 Session
- [x] `lifecycle.go::lobbyStateFromMembers(gameID, hostID, members) game.State` 헬퍼
- [x] `CreateSession` 마지막에 `EventEnvelope{Event: PlayerJoined{...}, Visibility: VisPublic}` 발행 + `persistAndDispatch` 호출
- [x] `JoinPlayer` 마지막에 동일 패턴
- [x] `JoinResult.CurrentState` 가 호출자에게 즉시 LOBBY 명단을 돌려주도록 갱신 (`lobby` 변수)
- [x] `lifecycle_test.go::TestLobbyMembership_BroadcastsPlayerJoined` — Subscribe 핸들러로 envelope 도착 카운트 검증 (호스트 1 + 5명 join → 6 envelope)
- [x] `lifecycle_test.go::TestLobbyMembership_JoinResultLobbyMembers` — JoinResult.CurrentState 의 누적 명단 검증
- [x] `go test ./internal/session/...` 통과 + 커버리지 ≥ 86 % 유지 — **실측 88.5 %** (커버리지 +2.0 pp 증가)

### Stage C — U3 Realtime Transport
- [x] `protocol.go::eventPayload` 에 `Name string \`json:"name,omitempty"\`` 추가
- [x] `dispatch.go::buildEventPayload` 에 `case game.PlayerJoined: ...` 케이스
- [x] `protocol_test.go::TestBuildEventPayload_AllKinds` 에 PlayerJoined 케이스 추가 + `TestBuildEventPayload_PlayerJoinedCarriesName` 신규 (Name 필드 wire 직렬화 보장)
- [x] `integration_test.go::TestE2E_LobbyMembershipBroadcast` — PUBLIC viewer + 호스트 + 5명 시나리오: 모든 connection 이 PlayerJoined 6건 수신 → `host:start` 후 PhaseChanged 가 모두에게 도달
- [x] `go test ./internal/transport/ws/...` 통과 + 커버리지 ≥ 85 % 유지 — **실측 89.3 %**

### Stage D — U5 Web Frontend
- [x] `wire.ts::EventPayload` union 에 `{ kind: "PlayerJoined"; playerId: PlayerID; name: string }` 추가
- [x] `reducer.ts::applyPlayerJoined` 케이스 추가 — 기존 `state.state` 가 있으면 append, 없으면 stub LOBBY State 초기화 (PUBLIC viewer/joiner 의 첫 이벤트 처리)
- [x] `reducer.test.ts` 에 3 케이스 추가: append / fresh-init / 중복 idempotent
- [x] `PlayersGrid.tsx` LOBBY 단계 렌더 (역할 칩 LOBBY 에서도 숨김 유지 + "대기 중" 상태 라인 추가)
- [x] `npm run typecheck && npm run test` 통과 + 핵심 모듈 커버리지 ≥ 78 % 유지 — **실측 79.95 %** (reducer.ts 92.2 %, NicknameForm 100 %)
- [x] `npm run build` — gzip JS **60.23 KB** ≪ 500 KB target, `cmd/mafia-game/web/dist/` 자동 동기화 (vite outDir 설정)

### Stage E — 통합 검증
- [x] `go build -o /tmp/mafia-game ./cmd/mafia-game` 성공 — Mach-O 64-bit arm64, 15.6 MB (Vite dist 동봉)
- [x] `go test ./...` 통과 — game/session/announce/persistence/transport.http/transport.ws 전 패키지 PASS
- [x] Chrome DevTools MCP 로 격리 컨텍스트 7개 (host + p1..p6) 동시 합류 — **2026-04-27 검증 통과**: GM 화면 7명 실시간 누적 → "게임 시작" 활성 → INTRO 진입, p1/p6 player 역할/키워드 정상. 부수 발견(catalog GetName fallback 으로 마피아 cohort 일부 PID raw hex 노출)은 본 plan 외 항목으로 분리
- [x] aidlc-state.md / audit.md 갱신, 본 plan 의 모든 체크박스 [x]

---

## 4. 리스크 / 주의

- **persistence**: `active_snapshot` 은 게임 시작(StartGame) 후에만 기록됨 (`SELECT * FROM active_snapshot;` 빈 결과로 확인됨). LOBBY 멤버 broadcast가 persistence를 건드릴 필요는 **없음**. `persistAndDispatch` 의 persist 분기가 LOBBY를 skip 하는지 점검 — 안 그러면 매 join 마다 빈 game state 가 저장될 수 있음.
- **state-clone**: `state_clone.go` 가 LOBBY phase 의 `Players` 를 깊은 복사하는지 확인.
- **NFR 영향**: 외부 의존 추가 없음 (NFR-7 유지). 커버리지 목표 단위별 동일 유지.
- **하위 호환**: 와이어 프로토콜 v1 — 새 event kind 추가는 backward-compatible (구 클라이언트는 `Unknown` 으로 안전 무시).
- **dispatch 순서**: `persistAndDispatch` 내부에서 announcement 렌더 후 envelope 분배. PlayerJoined 의 announce 카탈로그 항목(예: "{이름}님이 입장했습니다") 추가 여부 결정 필요 — 본 변경에서는 announce 비움(빈 Subtitle/Speech), 카탈로그 확장은 별도 plan.

---

## 5. 다음 세션 시작 시 체크리스트

1. 본 plan 의 §2 Q1/Q2 결정사항 확인 (이미 추천 채택)
2. 메모리 `project_overview` 의 "다음 작업" 라인 확인
3. `aidlc-state.md` 의 Post-Construction Maintenance 섹션 확인
4. Stage A → B → C → D → E 순서로 진행, 단위마다 사용자 승인 게이트
5. 통합 검증 후 본 plan 모든 체크박스 [x] 처리하고 aidlc-state 갱신

---

## 6. 참조

- 결함 재현 로그: 2026-04-27 chrome-devtools 세션 (server.log: `host:create-session` → `joined` 만 응답, broadcast 부재)
- 기존 dispatch 패턴: `internal/session/action.go:38-90` `persistAndDispatch`
- 기존 event 정의: `internal/game/event.go:39-170`
- 기존 wire 변환: `internal/transport/ws/dispatch.go:80-170` `buildEventPayload`
- 기존 reducer: `web/src/context/reducer.ts`
