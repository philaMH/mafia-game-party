# Iteration 3 — Build and Test Results

**문서 버전**: 1.0
**작성일**: 2026-04-29
**상위 변경 명세**: `audit.md` Iteration 3 (Late-Joiner Sync 옵션 A)
**처리 방식**: 본 반복 변경분(U2 + U3)과 무영향 단위(U1/U4/U5)를 분리. 전 패키지 회귀 + 신규 11 테스트 + Chrome DevTools MCP 수동 회귀 검증.

---

## 1. 단위별 산출물 갱신 요약

| 단위 | Functional Design 산출물 | 코드 변경 | 단위 테스트 결과 | 커버리지 (이전 → 현재) |
|---|---|---|---|---|
| **U1 Game Core** | (변경 없음) | (변경 없음) | PASS | **90.6%** 유지 |
| **U2 Session/Persistence/Announce** | `iteration3-patch.md` (S3-1, S3-2) | `types.go` (`RoomSnapshot` 구조체 export), `session.go` (`SessionManager.RoomSnapshot()` 인터페이스 + `*session.RoomSnapshot()` 구현) | PASS — 신규 6 테스트 (S3-T1~T6) | 87.4% → **88.2%** (+0.8 pp) |
| **U3 Realtime Transport** | `iteration3-patch.md` (W3-1~W3-3) | `dispatch.go` (`pushRoomState(c, snap)` 헬퍼), `hub.go` (Register welcome 직후 single-client push) | PASS — 신규 5 테스트 (W3-T1~T5) | 87.0% → **87.2%** (+0.2 pp) |
| **U4 HTTP Bootstrap & Static** | (변경 없음) | (변경 없음 — 정적 자산은 U5 빌드 산출 그대로) | PASS | **89.8%** 유지 |
| **U5 Web Frontend** | (변경 없음) | (변경 없음 — 기존 reducer가 `room:opened`/`snapshot`/`room:host-occupied`를 idempotent 처리하므로 클라이언트 변경 불요) | PASS — 38 tests | reducer.ts **92.2%** 유지 |

---

## 2. 통합 회귀 결과

### 2.1 Go 전체 테스트 (`go test ./... -count=1`)

```
ok  	github.com/saltware/mafia-game/internal/announce	0.574s
ok  	github.com/saltware/mafia-game/internal/game	0.369s
ok  	github.com/saltware/mafia-game/internal/persistence	0.823s
ok  	github.com/saltware/mafia-game/internal/session	1.150s
ok  	github.com/saltware/mafia-game/internal/transport/http	1.205s
ok  	github.com/saltware/mafia-game/internal/transport/ws	2.955s
```

전체 PASS. Iteration 2 회귀 영향 0건 (인터페이스 추가만 발생, 기존 시그니처/시맨틱 보존).

### 2.2 패키지별 커버리지 (`go test ./... -coverprofile=...`)

```
internal/announce         93.3% (유지)
internal/game             90.6% (유지)
internal/persistence      80.2% (유지)
internal/session          88.2% (87.4 → 88.2, +0.8 pp)
internal/transport/http   89.8% (유지)
internal/transport/ws     87.2% (87.0 → 87.2, +0.2 pp)
```

### 2.3 Go 빌드 (`go build -o /tmp/mafia-game-iter3 ./cmd/mafia-game`)

- **성공** — 단일 바이너리 15 MB 생성. Iteration 2 대비 사실상 동일 크기.

### 2.4 Web 테스트 (`npm test`)

```
✓ src/hooks/useToken.test.ts (3 tests)
✓ src/context/reducer.test.ts (24 tests)
✓ src/hooks/useTTSQueue.test.ts (5 tests)
✓ src/components/NicknameForm.test.tsx (6 tests)

Test Files  4 passed (4)
Tests       38 passed (38)
Duration    805ms
```

본 반복은 U5 변경 0건 → 신규 테스트 0건, 회귀 0건.

### 2.5 Web 빌드 (`npm run build`)

```
✓ 63 modules transformed.
../cmd/mafia-game/web/dist/index.html                   0.44 kB │ gzip:  0.30 kB
../cmd/mafia-game/web/dist/assets/index-DtVIq_uM.css    0.77 kB │ gzip:  0.49 kB
../cmd/mafia-game/web/dist/assets/index-CkgQSzta.js   186.47 kB │ gzip: 60.84 kB
✓ built in 330ms
```

- gzip 합계: **60.84 + 0.49 + 0.30 = 61.63 KB** (Iteration 2와 동일, NFR < 70 KB 한도 내).

---

## 3. 신규 시나리오 검증

### 3.1 단위/통합 테스트 (Go)

| 시나리오 | 검증 위치 | 결과 |
|---|---|---|
| ① RoomSnapshot 방 미개설 시 zero 값 반환 | `session/iteration3_test.go:TestRoomSnapshot_BeforeOpenRoom` | PASS |
| ② RoomSnapshot ClaimHost 후 OpenRoom 전 — `HostOccupied=true`, `RoomOpened=false` | `session/iteration3_test.go:TestRoomSnapshot_AfterClaimBeforeOpen` | PASS |
| ③ RoomSnapshot OpenRoom 후 — `RoomOpened=true`, `Options` 반영, `GameStarted=false` | `session/iteration3_test.go:TestRoomSnapshot_AfterOpenRoom` | PASS |
| ④ RoomSnapshot HostStartGame 후 — `GameStarted=true`, `State.Phase=INTRO`, `Players=6` | `session/iteration3_test.go:TestRoomSnapshot_AfterHostStartGame` | PASS |
| ⑤ RoomSnapshot ReleaseHost 후 — `HostOccupied=false`, `RoomOpened` 유지 | `session/iteration3_test.go:TestRoomSnapshot_AfterReleaseHost` | PASS |
| ⑥ RoomSnapshot.State 가 deep copy 임을 확인 (mutation 시 누수 없음) | `session/iteration3_test.go:TestRoomSnapshot_StateIsDeepCopy` | PASS |
| ⑦ Register 시 방 미개설이면 welcome 외 추가 메시지 없음 | `ws/iteration3_test.go:TestIter3_Register_BeforeOpenRoom_NoExtraMessages` | PASS |
| ⑧ Register 시 ClaimHost만 된 상태에서는 `room:host-occupied` 만 push | `ws/iteration3_test.go:TestIter3_Register_AfterClaimBeforeOpen_PushesHostOccupied` | PASS |
| ⑨ Register 시 방 개설 후에는 `room:opened` + `room:host-occupied` push, options 일치 | `ws/iteration3_test.go:TestIter3_Register_AfterOpenRoom_PushesRoomOpened` | PASS |
| ⑩ Register 시 게임 시작 후에는 `room:opened` + `snapshot(your=zero, isHost=false)` + `room:host-occupied` push, state.Phase=INTRO | `ws/iteration3_test.go:TestIter3_Register_AfterHostStartGame_PushesSnapshot` | PASS |
| ⑪ Register-time push 송신 순서: welcome → room:opened → snapshot → room:host-occupied | `ws/iteration3_test.go:TestIter3_Register_PushOrder` | PASS |

### 3.2 Chrome DevTools MCP 수동 회귀 검증

| 시나리오 | 절차 | 결과 |
|---|---|---|
| Late-joiner 정상 동기화 | (1) `/public` 접속 → "방 개설" 클릭 → "참가자 모집 중" 전환 확인. (2) **새 탭에서 `/play` 접속** | **PASS** — 즉시 "닉네임을 입력하고 입장하세요" 폼 표시 (수정 전 결함: "방이 아직 없습니다" 정체) |
| 콘솔 에러 부재 | 두 탭 콘솔 메시지 점검 | 결함 없음 (기존 a11y 경고 1건만 잔존, 본 패치 무관) |
| 서버 로그 정상 | `ws client registered` + `GET /ws status=200` 정상 출력 | PASS |

---

## 4. 회귀 영향 분석

| 영향 영역 | 분석 | 결과 |
|---|---|---|
| `SessionManager.Snapshot()` 기존 호출자 (U3 `routeEvent`) | 시그니처/시맨틱 변경 없음 | 무영향 |
| `OpenRoom` broadcast `room:opened` | 기존 broadcast 경로 그대로 보존, register-time push는 신규 단일 클라이언트만 대상 | 무영향. 동시 발생 시 reducer idempotent로 무해 |
| `Resume` snapshot 재전송 (handlers.go:95-105) | 변경 없음. PLAYER 컨텍스트 → `Your` 정보 포함 / Register-time push → PUBLIC zero `Your` 로 분리 유지 | 무영향 |
| `subscribe:public` no-op 핸들러 | 변경 없음 | 무영향 |
| 외부 mock/스텁 | `internal/transport/{ws,http}/*_test.go` 모두 `session.New` 실 인스턴스 사용 → 인터페이스 추가에도 깨질 코드 0건 | 무영향 |
| 클라이언트(U5) reducer | `room:opened`/`snapshot`/`room:host-occupied` 핸들러 기존 그대로 사용. zero `Your` snapshot도 무해 | 무영향 |

---

## 5. 비기능적 영향 (NFR)

| 항목 | 측정/추정 | 결과 |
|---|---|---|
| **지연 (NFR-U3-P1)** | `Register()` 동기 경로에 GM lock 1회 acquire + deep copy 1회 추가. SubmitAction 활동 중 경합 시에만 ms 단위 발생, 그 외 nanosecond 수준 | 영향 무시 가능 |
| **메시지 순서 (NFR-U3-P2)** | welcome → room:opened → snapshot → room:host-occupied 결정적 순서 (`TestIter3_Register_PushOrder` 검증) | 보장 |
| **idempotency** | `room:opened` 중복 수신 가능 시나리오 분석 + reducer 단순 set 동작 확인 | 무해 |
| **goroutine 누수** | `Register()` 내 동기 호출만 추가, 신규 고루틴 0개. 기존 `TestE2E_LeakNoGoroutineGrowth` 통과 | 무누수 |
| **번들 크기** | U5 변경 0건 → gzip 61.63 KB 동일 (한도 70 KB 이내) | 유지 |
| **바이너리 크기** | 15 MB 동일 | 유지 |

---

## 6. Iteration 3 DoD 체크리스트

- [x] U2 Functional Design Patch 작성 + 사용자 승인 (2026-04-29T08:55Z)
- [x] U3 Functional Design Patch 작성 + 사용자 승인 (2026-04-29T08:55Z)
- [x] Code Generation Plan (Part 1) 작성 + 사용자 승인 (2026-04-29T09:05Z)
- [x] Code Generation (Part 2) 단계 A~H 모두 [x]
- [x] U2 코드 변경 + 6 신규 테스트 PASS, 커버리지 88.2% (≥ 87.4%)
- [x] U3 코드 변경 + 5 신규 테스트 PASS, 커버리지 87.2% (≥ 87.0%)
- [x] `go test ./...` 6 패키지 PASS (Iteration 2 회귀 영향 0건)
- [x] `go build -o /tmp/mafia-game-iter3 ./cmd/mafia-game` 성공
- [x] `npm test` 38 PASS (U5 변경 0건이라 회귀만)
- [x] `npm run build` gzip 61.63 KB 한도 내
- [x] Chrome DevTools MCP late-joiner 회귀 검증 PASS
- [x] `aidlc-state.md` / `audit.md` / `iteration3-code-generation-plan.md` 체크박스 동기화
- [x] 본 결과 보고서 작성

---

## 7. 후속 권장 사항

1. **메시지 중복 dedup** — broadcast `room:opened`와 register-time push가 동시 발생할 때 동일 메시지 2회 수신 가능. 현재는 reducer idempotent라 문제 없음. 향후 wire 메시지에 `seq`/`ts` 부여 시 자연스럽게 해결됨 — Iteration 3 OOS, 별도 plan 후보.
2. **이벤트 누락 복구** — Iteration 3은 register-time *one-shot* 동기화만 다룸. 게임 진행 도중 잠시 끊겼다 재접속한 PUBLIC 클라이언트는 그동안 broadcast된 `event`들을 잃는다. PLAYER는 `Resume` 경로로 보호됨. PUBLIC 재접속 보호는 이벤트 로그 + 시퀀스 기반 재전송이 필요 — Iteration 3 OOS.
3. **`HostOccupied` 클라이언트 활용** — 기존 reducer는 `room:host-occupied` 수신 시 `hostOccupied=true` 설정. `/public` 폼이 사전 차단되도록 UI 처리 가능 (현재는 사용자 클릭 후 차단). UX 개선 후보, 본 반복 OOS.
4. **a11y 경고 해소 (별건)** — 기존 NicknameForm/숫자 입력 필드의 `id`/`name` 속성 누락 (Chrome DevTools 콘솔 issue). 본 반복 무관.

---

## 8. 추적성

| 산출물 | 경로 |
|---|---|
| FD U2 | `aidlc-docs/construction/u2-session-persistence-announce/functional-design/iteration3-patch.md` |
| FD U3 | `aidlc-docs/construction/u3-realtime-transport/functional-design/iteration3-patch.md` |
| Code Gen Plan | `aidlc-docs/construction/plans/iteration3-code-generation-plan.md` |
| 코드 변경 | `internal/session/{types,session,iteration3_test}.go`, `internal/transport/ws/{dispatch,hub,iteration3_test}.go` |
| 본 보고서 | `aidlc-docs/construction/build-and-test/iteration3-test-results.md` |
| Audit | `aidlc-docs/audit.md` (Iteration 3 항목 6건) |
| State | `aidlc-docs/aidlc-state.md` (Iteration 3 Stage Progress 섹션) |
