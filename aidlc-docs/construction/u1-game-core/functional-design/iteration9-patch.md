# U1 Game Core · Functional Design Patch — Iteration 9 (Fix · 최종 결과 발표 → 승리 화면 전환)

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration9-fix-final-result-requirements.md` v1.0 (사용자 승인 2026-04-30T00:35Z)
**Plan**: `aidlc-docs/construction/plans/iteration9-execution-plan.md` v1.0 (사용자 승인 2026-04-30T00:45Z)
**Predecessor FDs**: Iteration 4 (NightStep), Iteration 5 (Pause/Resume + 시간 기반 NightStep), Iteration 8 (INTRO/Day 안내 버퍼)
**Type**: Minimal Patch (State 필드 1건 + 도메인 상수 1건 + helper 2건 + tally/resolveNight/Tick 분기 + Pause/Resume 보강)

---

## 1. 목표

VOTE/RECOUNT 처형(`Eliminated`) 또는 NIGHT→DAY 사망 발표(`DeathAnnounced`/`PeacefulNight`) 와 동시에 게임 종료 조건이 충족될 때, `GameEnded` 이벤트를 즉시 emit 하지 말고 5초 버퍼 후 emit 하여 호스트/플레이어 화면이 결과 자막을 충분히 인식한 뒤 EndScreen 으로 전환되도록 한다.

`HostEndGame` (HOST_FORCE_END) 경로는 본 패치 적용 대상이 아니며 즉시 emit 동작을 보존한다 (Q5=A).

---

## 2. State 변경 — `PendingGameEnd` 필드 추가

```go
// internal/game/types.go

// PendingGameEnd holds a deferred GameEnded payload while the engine keeps
// the previous phase visible for the result-announcement buffer (FR-2 of
// Iteration 9). When non-nil, Tick will emit GameEnded once Deadline
// passes; if nil, the engine has no pending end.
//
// 와이어로 노출되긴 하나 클라이언트는 본 필드를 사용하지 않는다 (Q7=A —
// GameEnded emit 시점에 모든 화면이 일치한다).
type PendingGameEnd struct {
    Reason   EndReason `json:"reason"`
    Winner   *Team     `json:"winner,omitempty"`
    Deadline time.Time `json:"deadline"`
}

// State 추가 필드
type State struct {
    // ... 기존 필드 ...

    // NEW (Iteration 9): pending GameEnded 의 발행 시점/내용. Vote/Night
    // 결판 직후 채워지고, Tick 이 deadline 도달 시 firePendingEnd 가
    // 비우면서 실제 GameEnded + State.Phase=PhaseEnd 전환을 수행한다.
    PendingGameEnd *PendingGameEnd `json:"pendingGameEnd,omitempty"`
}
```

`State.Clone()` 에서 깊은 복사:

```go
// internal/game/state_clone.go (PoliceHistory 복사 부근)
if s.PendingGameEnd != nil {
    p := *s.PendingGameEnd
    if p.Winner != nil {
        w := *p.Winner
        p.Winner = &w
    }
    out.PendingGameEnd = &p
}
```

---

## 3. 도메인 상수 신설

```go
// internal/game/types.go (defaultDayIntroSeconds 그룹 인접)

// NEW (Iteration 9) — Q2=A: 결과 자막 노출 시간. Iter8 의
// defaultDayIntroSeconds 와 같은 5초로 정렬하여 사용자 체감 일관성 유지.
// 호스트 옵션으로 노출하지 않음 (Q3=B 와 동일 정책).
const defaultFinalResultBufferSeconds = 5
```

`Options` 구조체는 변경하지 않음.

---

## 4. 신규 헬퍼 — `scheduleGameEnd` / `firePendingEnd` / `evaluateEnd`

```go
// internal/game/end.go

// evaluateEnd reports whether the current state already meets a win
// condition, returning the winning team and end reason. Pure inspection;
// does not mutate state. Mirrors checkEnd's logic so both immediate-end
// (force) and deferred-end (vote/night) paths share the same rules.
func (e *engine) evaluateEnd() (EndReason, Team, bool) {
    if e.state.Phase == PhaseEnd {
        return "", "", false
    }
    mafia := e.state.LiveMafiaCount()
    citizens := e.state.LiveCitizenSideCount()
    switch {
    case mafia == 0:
        return EndCitizenWin, TeamCitizen, true
    case mafia >= citizens:
        return EndMafiaWin, TeamMafia, true
    }
    return "", "", false
}

// scheduleGameEnd records that the game has met an end condition but
// defers the actual GameEnded event by defaultFinalResultBufferSeconds
// (Iteration 9 FR-2). The current Phase is preserved so that the result
// subtitle (Eliminated / DeathAnnounced / PeacefulNight) remains visible.
// Returns no events — the caller has already emitted the result event.
//
// Idempotent: if a PendingGameEnd is already scheduled, the new request
// is ignored to avoid resetting the deadline (defensive — should not
// happen in the normal vote/night flow).
func (e *engine) scheduleGameEnd(reason EndReason, winner Team) {
    if e.state.PendingGameEnd != nil {
        return
    }
    w := winner
    e.state.PendingGameEnd = &PendingGameEnd{
        Reason:   reason,
        Winner:   &w,
        Deadline: e.clock.Now().Add(time.Duration(defaultFinalResultBufferSeconds) * time.Second),
    }
}

// firePendingEnd consumes the deferred end record and emits GameEnded
// while transitioning to PhaseEnd. Called only by Tick when
// PendingGameEnd.Deadline has been reached.
func (e *engine) firePendingEnd(now time.Time) (State, []EventEnvelope, error) {
    pending := e.state.PendingGameEnd
    e.state.PendingGameEnd = nil
    e.state.LastTickAt = now
    if pending.Winner == nil {
        // Defensive — schedule path always sets Winner. Fall through to
        // a degenerate end with EndForce semantics (no team revealed).
        return e.state.Clone(), e.endGameForceful(pending.Reason), nil
    }
    return e.state.Clone(), e.endGame(pending.Reason, *pending.Winner), nil
}
```

> 설명: `evaluateEnd` 는 `checkEnd` 와 같은 판정을 부수효과 없이 수행. 기존 `checkEnd` 는 호환을 위해 유지되며 `HostEndGame` 또는 미래 다른 경로에서 즉시 종료가 필요할 때 사용 가능. `firePendingEnd` 의 nil-Winner 분기는 안전망(현 schedule 경로는 항상 Winner 보장).

---

## 5. `tally.applyElimination` 변경

```go
// internal/game/tally.go
func (e *engine) applyElimination(id PlayerID) []EventEnvelope {
    events := make([]EventEnvelope, 0, 3)
    p, ok := e.state.FindPlayer(id)
    if !ok {
        return events
    }
    p.Alive = false
    events = append(events, pub(Eliminated{PlayerID: id, Role: p.Role}))
    if id == e.state.MafiaRepresentativeID {
        events = append(events, e.reassignMafiaRepresentative(id)...)
    }
    // CHANGED (Iteration 9): 즉시 endGame → schedule 로 전환.
    if reason, winner, ok := e.evaluateEnd(); ok {
        e.scheduleGameEnd(reason, winner)
        return events // Phase=Vote/Recount 유지, transitionVoteToNight 생략
    }
    events = append(events, e.transitionVoteToNight()...)
    return events
}
```

이벤트 발행 순서 (vote-end 시민 승리 예시):
1. `VoteTallied{Eliminated: mafia, Recount: false}` (catalog suppressed)
2. `Eliminated{PlayerID: mafia, Role: MAFIA}` → catalog `eliminated.mafia` cue
3. (5초 경과 후 Tick) `GameEnded{Winner: CITIZEN, Reason: CITIZEN_WIN}` → catalog `end.citizen` cue + State.Phase=PhaseEnd

---

## 6. `resolve_night.resolveNight` 변경

```go
// internal/game/resolve_night.go
func (e *engine) resolveNight() ([]EventEnvelope, error) {
    // ... (victim 결정 / 상태 리셋 / Day 진행 / PhaseChanged{DAY} / DeathAnnounced/PeacefulNight 발행 — 변경 없음)

    // CHANGED (Iteration 9): 즉시 endGame → schedule 로 전환.
    if reason, winner, ok := e.evaluateEnd(); ok {
        e.scheduleGameEnd(reason, winner)
    }
    return events, nil
}
```

이벤트 발행 순서 (night-end 마피아 승리 예시):
1. `PhaseChanged{DAY, Day=2, Deadline=now+5s+180s}` → catalog `phase.day` cue
2. `DeathAnnounced{Victim: citizen}` → catalog `death.announced` cue
3. (5초 경과 후 Tick) `GameEnded{Winner: MAFIA, Reason: MAFIA_WIN}` → catalog `end.mafia` cue + State.Phase=PhaseEnd

> Note: DAY 의 `Deadline` (5+180=185s 후) 은 어차피 도달하기 전에 PendingGameEnd 가 5s 후 발화되므로 무관. tickDay 의 DiscussionTimerTick 도 발화되지 않는다 (PendingGameEnd 분기가 Tick 진입 직후 우선).

---

## 7. `tick.Tick` 변경 — 진입 분기

```go
// internal/game/tick.go
func (e *engine) Tick(now time.Time) (State, []EventEnvelope, error) {
    if e.state.Paused {
        return e.state.Clone(), nil, nil
    }
    if !now.After(e.state.LastTickAt) {
        return e.state.Clone(), nil, nil
    }
    // NEW (Iteration 9): pending end 가 만료되었으면 다른 어떤 phase
    // 진행보다 우선해 GameEnded 를 발화한다. 이 분기 이후의 phase
    // switch 는 실행되지 않는다 (게임이 끝났으므로).
    if e.state.PendingGameEnd != nil && !now.Before(e.state.PendingGameEnd.Deadline) {
        return e.firePendingEnd(now)
    }

    prev := e.state.LastTickAt
    e.state.LastTickAt = now

    switch e.state.Phase {
    case PhaseIntro:
        return e.tickIntro(now)
    case PhaseDay:
        return e.tickDay(now, prev)
    case PhaseNight:
        return e.tickNight(now)
    default:
        return e.state.Clone(), nil, nil
    }
}
```

> Note: `firePendingEnd` 가 자체적으로 `LastTickAt = now` 를 설정하므로, 분기 진입 시 LastTickAt 갱신 코드를 통과하지 않는 구조로 구성한다. 만약 이번 Tick 이 PendingGameEnd 도달 시각에 미달이면 분기를 통과하지 않고 아래 phase switch 로 진행한다.

---

## 8. `handlers_lifecycle.handleEndGame` (HOST_FORCE_END) 보강

```go
// internal/game/handlers_lifecycle.go
func (e *engine) handleForceEnd(a ForceEndGame) (State, []EventEnvelope, error) {
    if err := ensureHost(&e.state, a.HostID); err != nil {
        return e.state.Clone(), nil, err
    }
    if e.state.Phase == PhaseEnd {
        return e.state.Clone(), nil, errf(CodeWrongPhase, "already ended")
    }
    // NEW (Iteration 9): 자연 결판 대기 중이라도 호스트 강제 종료가
    // 즉시 우선한다. Pending 을 클리어하고 그대로 즉시 emit.
    e.state.PendingGameEnd = nil

    reason := EndHostForceEnd
    e.state.Phase = PhaseEnd
    e.state.EndReason = &reason
    e.state.Winner = nil
    reveal := make([]Player, len(e.state.Players))
    copy(reveal, e.state.Players)
    return e.state.Clone(), []EventEnvelope{pub(GameEnded{
        Winner:    nil,
        EndReason: reason,
        Reveal:    reveal,
    })}, nil
}
```

---

## 9. `handlers_lifecycle.canPause` / `handleResumeGame` 보강

```go
// internal/game/handlers_lifecycle.go

// CHANGED (Iteration 9): pending end 대기 중에는 phase 와 무관하게
// pause 를 허용한다. Q4=A — 결과 발표 버퍼는 Pause 영향을 받음.
func canPauseState(s *State) bool {
    if s.PendingGameEnd != nil {
        return true
    }
    return canPause(s.Phase)
}

func (e *engine) handlePauseGame(a PauseGame) (State, []EventEnvelope, error) {
    if err := ensureHost(&e.state, a.HostID); err != nil {
        return e.state.Clone(), nil, err
    }
    if !canPauseState(&e.state) {
        return e.state.Clone(), nil, errf(CodeWrongPhase,
            "cannot pause during phase %s", e.state.Phase)
    }
    if e.state.Phase == PhaseNight && e.state.NightStep == NightStepIntro && e.state.PendingGameEnd == nil {
        return e.state.Clone(), nil, errf(CodeWrongPhase,
            "cannot pause during night intro")
    }
    // ... (idempotent 체크 + Paused/PausedAt 세팅 — 변경 없음)
}

func (e *engine) handleResumeGame(a ResumeGame) (State, []EventEnvelope, error) {
    // ... (기존 phase별 deadline shift — 변경 없음)

    // NEW (Iteration 9): pending end 도 동일 shift.
    if e.state.PendingGameEnd != nil {
        e.state.PendingGameEnd.Deadline = e.state.PendingGameEnd.Deadline.Add(shift)
    }

    // ... (Paused/PausedAt 클리어 + LastTickAt 재설정 + GameResumed emit — 변경 없음)
}
```

> Note: `canPause(p Phase)` 는 기존 호출자(`canPauseState`) 만 사용하도록 정리하면 다른 회귀 영향이 없음. `canPauseState` 신설 대신 `handlePauseGame` 내부에서 직접 `if e.state.PendingGameEnd != nil || canPause(e.state.Phase)` 로 표현해도 동등 — 실 구현 시 가독성 우선으로 선택.

---

## 10. `applyElimination` 분기 도식

```
applyElimination(id)
 ├─ Eliminated{id, role} emit
 ├─ (옵션) MafiaRepresentativeReassigned (마피아 대표 사망 시)
 ├─ evaluateEnd()?
 │   ├─ true → scheduleGameEnd(reason, winner) → return events
 │   │       [Phase 그대로, Tick 5s 후 GameEnded]
 │   └─ false → transitionVoteToNight() → enterNight()
 │           [Phase=Night, INTRO 진입]
 └─ return events
```

```
resolveNight()
 ├─ victim 결정 / 상태 리셋
 ├─ Day++, Phase=Day, Deadline=now+5+180
 ├─ PhaseChanged{Day} emit
 ├─ DeathAnnounced or PeacefulNight emit
 ├─ evaluateEnd()?
 │   └─ true → scheduleGameEnd(reason, winner) (Phase=Day 그대로, Tick 5s 후 GameEnded)
 └─ return events
```

```
Tick(now)
 ├─ if Paused → no-op
 ├─ if !now.After(LastTickAt) → no-op
 ├─ if PendingGameEnd != nil && now ≥ Deadline → firePendingEnd(now)
 │       └─ Phase=End, GameEnded emit
 └─ else → 기존 phase switch
```

---

## 11. 호환성 / 회귀 영향

| 영역 | 영향 |
|---|---|
| **Snapshot (legacy)** | `PendingGameEnd` omitempty + nil 기본 — 기존 스냅샷 무손상 호환. 신규 스냅샷에 본 필드가 채워진 채 서버 재기동 시, Tick 이 deadline 비교로 정상 발화. |
| **Wire (legacy 클라이언트)** | `pendingGameEnd` 필드는 클라이언트가 무시 (TS 의 `State` 인터페이스 unknown property tolerant). 와이어 변경 없음. |
| **announce 카탈로그** | 변경 없음. `eliminated.*` / `death.*` / `peaceful.*` / `end.*` cue 는 기존 트리거 그대로. |
| **U5 reducer** | 변경 없음. `GameEnded` 처리 로직 동일, 단지 도달이 5초 늦어짐. |
| **Pause/Resume** | Iter5 의 INTRO 가드는 PendingGameEnd 가 없을 때만 적용. PendingGameEnd 가 있으면 phase 무관 pause 가능 (Q4=A). |
| **HOST_FORCE_END** | 변경 없음 + PendingGameEnd 클리어로 race 안전 (Q5=A). |
| **Tick idempotency** | LastTickAt 갱신 위치를 firePendingEnd 분기에서 별도 처리. 같은 `now` 로 두 번 Tick 호출해도 두 번째는 LastTickAt 비교로 no-op. |

---

## 12. 테스트 케이스 (I9-T1 ~ I9-T7)

| ID | 시나리오 | 검증 |
|---|---|---|
| I9-T1 | Vote-end (시민 승리) | tally 직후 `Eliminated` 만 emit, GameEnded 없음. PendingGameEnd != nil, Deadline = now + 5s. State.Phase = PhaseVote 유지 |
| I9-T2 | 위 상태에서 `clock.Advance(5s)` + Tick → `GameEnded{CITIZEN_WIN, CITIZEN}` emit, State.Phase=PhaseEnd, PendingGameEnd=nil |
| I9-T3 | Night-end (마피아 승리) | resolveNight 직후 `PhaseChanged{Day}` + `DeathAnnounced` emit, GameEnded 없음. PendingGameEnd != nil. State.Phase=PhaseDay |
| I9-T4 | 위 상태에서 `clock.Advance(5s)` + Tick → `GameEnded{MAFIA_WIN, MAFIA}` emit, State.Phase=PhaseEnd |
| I9-T5 | Pause/Resume mid-buffer | Vote-end → 1s 경과 → PauseGame (허용, Phase=Vote 임에도) → 30s 경과 → ResumeGame (PendingGameEnd.Deadline 이 +30s shift) → 4s 경과 → Tick → GameEnded emit |
| I9-T6 | HOST_FORCE_END mid-buffer | Vote-end → 2s 경과 → ForceEndGame → 즉시 `GameEnded{HOST_FORCE_END, Winner=nil}`, PendingGameEnd=nil. 추가 Tick 호출은 no-op (이미 PhaseEnd) |
| I9-T7 | Snapshot resume mid-buffer | Vote-end → 2s → Snapshot 직렬화/역직렬화 → 3s 경과 → Tick → GameEnded emit (legacy nil 도 동일 nil 보존) |

테스트 헬퍼:
- `internal/game/fixtures_test.go` 에 `runPendingEndTick(e, clock, t)` 추가 — `clock.Advance(defaultFinalResultBufferSeconds * time.Second + 1ms)` + `engine.Tick(clock.Now())` wrap. 호출 측은 1줄.
- 기존 `end_test.go::TestEngine_GameEndsWhenAllMafiaEliminated` 유사 시나리오: PendingGameEnd 검증을 1줄 추가하고 Tick 발화로 마무리.

---

## 13. 영향 받는 파일

| 파일 | 변경 종류 | 라인 수 (대략) |
|---|---|---|
| `internal/game/types.go` | PendingGameEnd struct + State 필드 + 상수 | +18 |
| `internal/game/state_clone.go` | PendingGameEnd 깊은 복사 | +9 |
| `internal/game/end.go` | `evaluateEnd` / `scheduleGameEnd` / `firePendingEnd` 신규 | +35 |
| `internal/game/tally.go` | `applyElimination` 의 checkEnd 분기 변경 | +3 -2 |
| `internal/game/resolve_night.go` | `resolveNight` 의 checkEnd 분기 변경 | +3 -2 |
| `internal/game/tick.go` | Tick 진입부 PendingGameEnd 체크 + LastTickAt 처리 위치 조정 | +6 |
| `internal/game/handlers_lifecycle.go` | canPauseState / handleResumeGame shift / handleForceEnd 클리어 | +12 |
| `internal/game/iteration9_test.go` (신규) | I9-T1~T7 | +280 |
| `internal/game/fixtures_test.go` | runPendingEndTick 헬퍼 | +6 |
| `internal/game/end_test.go` 외 | 기존 시나리오 5건 마이그레이션 (clock.Advance + Tick) | +20 -5 |

다른 단위 (U2/U3/U4/U5) 는 변경 없음.

---

## 14. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-30 | 최초 작성 |
