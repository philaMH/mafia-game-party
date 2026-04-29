# U1 Game Core · Code Generation Plan — Iteration 9 (Fix · 최종 결과 발표 → 승리 화면 전환)

**Status**: v1.0 — 사용자 승인 대기
**Source**: `iteration9-fix-final-result-requirements.md` v1.0 + `iteration9-execution-plan.md` v1.0 + `u1-game-core/functional-design/iteration9-patch.md` v1.0
**Workspace Root**: `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game/.claude/worktrees/fix+final-result/`
**Code Location**: `internal/game/` (워크스페이스 루트, 브라운필드 — 기존 파일 수정 우선)
**Single Source of Truth**: 본 plan 의 step 체크리스트가 Code Generation 의 기준이다.

---

## 0. 사전 조건

- Functional Design Patch v1.0 사용자 승인 완료.
- 영향 단위는 U1 만. U2/U3/U4/U5 변경 없음.
- 사용자 답변 (Q1~Q8 = 모두 A) 보존.

---

## 1. 변경 대상 파일 표

| # | 파일 | 변경 종류 | Step |
|---|---|---|---|
| 1 | `internal/game/types.go` | PendingGameEnd struct + State 필드 + 상수 | A |
| 2 | `internal/game/state_clone.go` | PendingGameEnd 깊은 복사 | B |
| 3 | `internal/game/end.go` | evaluateEnd / scheduleGameEnd / firePendingEnd 헬퍼 | C |
| 4 | `internal/game/tally.go` | applyElimination 의 checkEnd 분기 변경 | D |
| 5 | `internal/game/resolve_night.go` | resolveNight 의 checkEnd 분기 변경 | D |
| 6 | `internal/game/tick.go` | Tick 진입부 PendingGameEnd 분기 + LastTickAt 위치 조정 | E |
| 7 | `internal/game/handlers_lifecycle.go` | canPause 정책 + handleResumeGame shift + handleForceEnd 클리어 | F |
| 8 | `internal/game/fixtures_test.go` | runPendingEndTick 헬퍼 (1 함수 추가) | G |
| 9 | `internal/game/iteration9_test.go` (신규) | I9-T1~T7 (총 7 케이스) | H |
| 10 | `internal/game/end_test.go` 등 시나리오 회귀 | clock.Advance + Tick 마이그레이션 (예상 4~6 곳) | I |

총 9개 기존 파일 수정 + 1개 신규 + 1개 테스트 헬퍼 보강 + 시나리오 마이그레이션.

---

## 2. Step 체크리스트

### Step A — `internal/game/types.go` ✅
- [x] `PendingGameEnd` struct 신규 정의
- [x] `State.PendingGameEnd *PendingGameEnd` 필드 추가
- [x] `defaultFinalResultBufferSeconds = 5` 상수 추가
- [x] `go vet ./internal/game/...` 통과

### Step B — `internal/game/state_clone.go` ✅
- [x] `PendingGameEnd` 깊은 복사 분기 추가 (Winner 포인터 깊은 복사 포함)
- [x] `go vet` 통과

### Step C — `internal/game/end.go` ✅
- [x] `evaluateEnd() (EndReason, Team, bool)` 신규 — 판정만 분리
- [x] `scheduleGameEnd(reason, winner)` 신규 — idempotent
- [x] `firePendingEnd(now)` 신규 — endGame 위임 + PendingGameEnd 클리어 + LastTickAt 갱신
- [x] 기존 `checkEnd` 시그니처는 `evaluateEnd → endGame` 로 재구성 (하위 호환)
- [x] `time` import 추가
- [x] `go vet` 통과

### Step D — `tally.go` + `resolve_night.go` ✅
- [x] `tally.applyElimination`: `evaluateEnd → scheduleGameEnd` 분기로 치환, transitionVoteToNight 생략
- [x] `resolve_night.resolveNight`: `evaluateEnd → scheduleGameEnd` 분기로 치환 (Phase=Day 유지)
- [x] `go vet` 통과

### Step E — `internal/game/tick.go` ✅
- [x] `Tick(now)` 진입부 PendingGameEnd 만료 분기 추가 (Paused/LastTickAt 가드 직후)
- [x] firePendingEnd 분기 진입 시 prev/LastTickAt 갱신을 우회하도록 본문 정리
- [x] `go vet` 통과

### Step F — `internal/game/handlers_lifecycle.go` ✅
- [x] `handlePauseGame`: PendingGameEnd != nil 분기 추가 (phase 무관 허용, INTRO 가드는 pending 없을 때만)
- [x] `handleResumeGame`: PendingGameEnd.Deadline 도 shift
- [x] `handleForceEnd`: 시작 시 PendingGameEnd 클리어
- [x] `go vet` 통과

### Step G — `internal/game/fixtures_test.go` ✅
- [x] `runPendingEndTick(t, e, clock)` 헬퍼 추가 — Advance 후 Tick wrap, `t.Helper()` 호출

### Step H — `internal/game/iteration9_test.go` (신규) ✅
- [x] I9-T1 `TestI9_VoteEndSchedulesPendingGameEnd` PASS
- [x] I9-T2 `TestI9_VoteEndTickFiresGameEnded` PASS
- [x] I9-T3 `TestI9_NightEndSchedulesPendingGameEnd` PASS (6p / 2 mafia 시나리오)
- [x] I9-T4 `TestI9_NightEndTickFiresGameEnded` PASS
- [x] I9-T5 `TestI9_PauseResumeShiftsPendingGameEnd` PASS
- [x] I9-T6 `TestI9_HostForceEndClearsPending` PASS
- [x] I9-T7 `TestI9_SnapshotRoundTripPreservesPendingGameEnd` PASS
- [x] `go test ./internal/game/... -run TestI9_ -count=1 -v` 7/7 PASS

### Step I — 기존 시나리오 마이그레이션 ✅ (마이그레이션 불필요)
- [x] 후보 검색 (`grep`): `end_test.go::TestCheckEnd_*` 는 `checkEnd()` 직접 호출 (즉시 emit 보존), `handlers_lifecycle_test.go::TestForceEndGame_TerminalState` 는 ForceEndGame 즉시 emit 보존. 그 외 vote/night-end 시나리오 직접 검증 케이스 없음
- [x] `go test ./... -count=1 -race` — 6 패키지 PASS
- [x] `go build -o /tmp/mafia-game-iter9 ./cmd/mafia-game` — 17 MB 성공
- [x] `go test -cover ./internal/game/` — 92.5% (Iter8 91.8% → +0.7pp)
- [x] `npm test` — 66/66 PASS (Iter8 동일)
- [x] `npm run build` — JS gzip 65.62 KB / CSS 3.21 KB (Iter8 동일)

---

## 3. 검증 게이트 (각 Step 종료 시)

| Step | 게이트 |
|---|---|
| A~G | `go vet ./internal/game/...` 통과 + 컴파일 통과 |
| H | `go test ./internal/game/... -count=1 -race -run Iteration9` PASS |
| I | `go test ./... -count=1 -race` 6 패키지 PASS, `go build` 성공, `go test -cover ./internal/game/` ≥ 91.8% (Iter8 baseline) |

각 Step 완료 후 본 plan 의 체크박스를 `[x]` 로 업데이트하고, 모든 Step 완료 후 `aidlc-state.md` 의 U1 Code Generation 체크박스를 `[x]` 마킹.

---

## 4. Story / Acceptance Criteria 추적

| Step | 충족 AC |
|---|---|
| Step A~B | NFR-3 (snapshot 호환), 직렬화 기본 동작 |
| Step C | FR-1 (양 경로 통합), FR-2 (5초 버퍼 상수), FR-7 (서버 단일 emit) |
| Step D | FR-1 의 vote/night 양 진입점 통합 |
| Step E | FR-2 의 발화 시점 (Tick 기반) |
| Step F | FR-4 (Pause 영향), FR-5 (HOST_FORCE_END 즉시) |
| Step G~H | AC-1~AC-7 (Vote-end / Night-end / Pause / Force / Snapshot) |
| Step I | NFR-6 (회귀 영향 최소화) |

---

## 5. Out of Scope

- U2 카탈로그 — 변경 없음
- U3 와이어 프로토콜 — 변경 없음
- U4 HTTP — 변경 없음
- U5 reducer/뷰 — 변경 없음
- 음성 cue 길이/순서 — 변경 없음 (Q3=A — 기존 흐름 보존)
- 호스트 옵션 노출 — 변경 없음 (Q3=B 와 동일 정책)

---

## 6. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-30 | 최초 작성 |
