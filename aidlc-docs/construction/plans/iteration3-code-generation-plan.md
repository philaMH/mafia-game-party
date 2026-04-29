# Iteration 3 Code Generation Plan — Late-Joiner Sync

**문서 버전**: 1.0
**작성일**: 2026-04-29
**범위**: U2 + U3 단위 패치 (옵션 A)
**기준 산출물**:
- `aidlc-docs/construction/u2-session-persistence-announce/functional-design/iteration3-patch.md` (S3-1, S3-2)
- `aidlc-docs/construction/u3-realtime-transport/functional-design/iteration3-patch.md` (W3-1 ~ W3-3)

본 plan은 Construction Code Generation Part 1 (Planning) 산출물이다. 사용자 승인 후 Part 2 (Generation) 단계에서 체크박스를 차례로 [x] 처리하며 코드를 생성한다.

---

## 0. 사전 조건 (DoR)

- [x] FD 패치 사용자 승인 완료 (2026-04-29T08:55Z)
- [x] 기존 코드 구조 확인: `hostAuthority.IsClaimed()` 이미 존재 (`internal/session/host_authority.go:64-68`) → 신설 불요
- [x] 테스트 mock 영향 없음 — `internal/transport/ws/*_test.go`, `internal/transport/http/*_test.go` 모두 `session.New` 실 인스턴스 사용
- [x] 클라이언트(U5) 변경 0건 — reducer가 기존 메시지 핸들링으로 자연 커버

---

## 1. 변경 파일 목록

### U2
| # | 파일 | 변경 종류 | 비고 |
|---|---|---|---|
| 1 | `internal/session/types.go` | 추가 | `RoomSnapshot` 구조체 export |
| 2 | `internal/session/session.go` | 변경 | `SessionManager` 인터페이스에 `RoomSnapshot()` 추가, `*session` 구현 추가 |
| 3 | `internal/session/iteration3_test.go` | 신규 | 6 테스트 (S3-T1~T6) |

### U3
| # | 파일 | 변경 종류 | 비고 |
|---|---|---|---|
| 4 | `internal/transport/ws/dispatch.go` | 변경 | `pushRoomState(c, snap)` 헬퍼 추가 |
| 5 | `internal/transport/ws/hub.go` | 변경 | `Register()`에서 welcome 직후 push 호출 |
| 6 | `internal/transport/ws/iteration3_test.go` | 신규 | 5 테스트 (W3-T1~T5) |

### 공통
| # | 파일 | 변경 종류 | 비고 |
|---|---|---|---|
| 7 | `aidlc-docs/aidlc-state.md` | 갱신 | Iteration 3 체크박스 진행 |
| 8 | `aidlc-docs/audit.md` | 갱신 | Code Generation 시작/완료 로그 |

---

## 2. 단계별 체크리스트 (실행 시 [ ] → [x])

### 단계 A — U2 자료구조 (`types.go`)
- [x] `RoomSnapshot` 구조체 정의 추가 (`Session` 정의 직후 위치)
- [x] doc comment: 패치 §2 주석 그대로 사용
- [x] `go vet ./internal/session/...` 통과

### 단계 B — U2 인터페이스 + 구현 (`session.go`)
- [x] `SessionManager` 인터페이스에 `RoomSnapshot() RoomSnapshot` 추가 (line 45 직후)
- [x] `*session` 메서드 `RoomSnapshot()` 구현 (`Snapshot()` 직후 line 232 부근)
  - GM lock acquire → engine.Snapshot + Session 필드 + hostAuth.IsClaimed → return
- [x] doc comment: 패치 §3.1 의사코드 반영
- [x] `go vet ./internal/session/...` 통과

### 단계 C — U2 테스트 (`iteration3_test.go`)
- [x] 신규 파일 헤더 (package session_test, imports)
- [x] S3-T1: `TestRoomSnapshot_BeforeOpenRoom`
- [x] S3-T2: `TestRoomSnapshot_AfterClaimBeforeOpen`
- [x] S3-T3: `TestRoomSnapshot_AfterOpenRoom`
- [x] S3-T4: `TestRoomSnapshot_AfterHostStartGame`
- [x] S3-T5: `TestRoomSnapshot_AfterReleaseHost`
- [x] S3-T6: `TestRoomSnapshot_StateIsDeepCopy`
- [x] `go test ./internal/session/...` PASS

### 단계 D — U3 dispatch 헬퍼 (`dispatch.go`)
- [x] `pushRoomState(c *Client, snap session.RoomSnapshot)` 추가 (`broadcastRoomOpened` 직후)
- [x] doc comment: 패치 §2 의사코드 반영, 송신 순서(opened → snapshot → host-occupied) 명시
- [x] 분기별 enqueue:
  - [x] `RoomOpened` → `roomOpenedMsg{Type: TypeRoomOpened, Options: snap.Options}`
  - [x] `GameStarted` → `snapshotMsg{Type: TypeSnapshot, State: snap.State, IsHost: false, Your: yourInfo{}}`
  - [x] `HostOccupied` → `roomHostOccupiedMsg{Type: TypeRoomHostOccupied}`
- [x] `go vet ./internal/transport/ws/...` 통과

### 단계 E — U3 Register 통합 (`hub.go`)
- [x] `Register()` 본문 welcome enqueue 직후 (line 98 직후) `snap := h.mgr.RoomSnapshot()` 호출
- [x] `h.pushRoomState(c, snap)` 호출 (registry.add 이후, readLoop 시작 이전)
- [x] doc comment 갱신: late-joiner sync 한 줄 추가
- [x] `go vet ./internal/transport/ws/...` 통과

### 단계 F — U3 테스트 (`iteration3_test.go`)
- [x] 신규 파일 헤더 (package ws, imports)
- [x] 헬퍼: 새 클라이언트 dial + 메시지 read until N함수
- [x] W3-T1: `TestIter3_Register_BeforeOpenRoom_NoExtraMessages` (welcome only)
- [x] W3-T2: `TestIter3_Register_AfterClaimBeforeOpen_PushesHostOccupied`
- [x] W3-T3: `TestIter3_Register_AfterOpenRoom_PushesRoomOpened`
- [x] W3-T4: `TestIter3_Register_AfterHostStartGame_PushesSnapshot`
- [x] W3-T5: `TestIter3_Register_PushOrder` (welcome → opened → snapshot → host-occupied)
- [x] `go test ./internal/transport/ws/...` PASS

### 단계 G — 통합 빌드/회귀
- [x] `go test ./...` 전체 PASS
- [x] `go build -o /tmp/mafia-game ./cmd/mafia-game` 성공
- [x] (선택) Chrome DevTools MCP late-joiner 시나리오 수동 검증

### 단계 H — 문서 동기화
- [x] `aidlc-state.md` Iteration 3 Code Generation 체크박스 [x]
- [x] `audit.md` "Iteration 3 — Code Generation 완료" 항목 append

---

## 3. 위험 분석 및 완화

| 위험 | 가능성 | 영향 | 완화 |
|---|---|---|---|
| 인터페이스 추가로 외부 mock 깨짐 | 낮음 | 빌드 실패 | 사전 grep 결과 mock 0건. 빌드 시 `go vet`이 즉시 검출 |
| Register 동기 경로에서 GM lock 보유 시간 증가 | 낮음 | 신규 connection 처리 지연 | RoomSnapshot은 deep copy 한 번 + atomic 필드 읽기. 경합은 SubmitAction 활동 중에만 발생, ms 단위 |
| broadcast `room:opened`와 register-time push 동시 발생 시 메시지 2회 | 낮음 | reducer idempotent 처리, 무해 | 패치 §3.3에 명시. 별도 sequence id OOS |
| 테스트 timing flaky (WS read deadline) | 중간 | 테스트 간헐 실패 | 기존 `iteration2_test.go` `readType` helper 패턴 그대로 차용, deadline 1s |

---

## 4. DoD (Definition of Done)

- [x] 단계 A~F 모든 체크박스 [x]
- [x] `go test ./...` PASS, 신규 11개 테스트 모두 PASS
- [x] `go build` 성공
- [x] U2/U3 커버리지 직전 Iteration 대비 동등 이상 (`go test -coverprofile=...` 결과 기록)
- [x] `aidlc-state.md`/`audit.md` 갱신 완료

---

## 5. Out of Scope (재확인)

- U1, U4, U5 코드 변경
- 신규 wire 메시지 타입
- 클라이언트 reducer/UI 변경
- 영속성 layer 확장
- WebSocket reconnect/backoff 자동화

---

## 6. 실행 순서 약속

플랜 승인 시 Part 2 실행은 단계 A → B → C → D → E → F → G → H 의 순으로 진행하며, 각 단계 완료 시 체크박스를 [x]로 갱신한다. 단계 C, F, G에서 테스트 실패 시 즉시 중단 + 사용자에게 보고 후 결정 대기.
