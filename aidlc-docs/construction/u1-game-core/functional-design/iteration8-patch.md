# U1 Game Core · Functional Design Patch — Iteration 8 (Fix · 밤 진입 안내)

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration8-fix-vote-result-requirements.md` v1.0 (사용자 승인 2026-04-29T21:25Z)
**Plan**: `aidlc-docs/construction/plans/iteration8-execution-plan.md` v1.0 (사용자 승인 2026-04-29T21:35Z)
**Predecessor FDs**: Iteration 4 (NightStep 도입), Iteration 5 (시간 기반 NightStep 진행 + Pause/Resume)
**Type**: Minimal Patch (enum 1건 + 도메인 상수 2건 + transition 분기 + Pause 가드)

---

## 1. 목표

VOTE → NIGHT 진입 시 마피아 카운트다운이 안내 음성과 동시에 시작되어 시간이 잠식되는 결함을 해결.
NIGHT → DAY 진입 시에도 사망 발표 음성과 토론 카운트다운이 동시 시작되는 잠재 결함을 동일 정책으로 처리.

---

## 2. NightStep enum 변경

```go
// internal/game/types.go
type NightStep string

const (
    // NEW (Iteration 8): INTRO step is a 5s buffer that lets the
    // host's `phase.night` cue finish before mafia begins.
    NightStepIntro    NightStep = "INTRO"
    NightStepMafia    NightStep = "MAFIA"
    NightStepPolice   NightStep = "POLICE"
    NightStepDoctor   NightStep = "DOCTOR"
    NightStepResolved NightStep = "RESOLVED"
)
```

Visible order (state machine): `INTRO → MAFIA → POLICE → DOCTOR → RESOLVED`.

---

## 3. 도메인 상수 신설

```go
// internal/game/types.go (defaultNight*Seconds 그룹 내부)
const (
    // NEW (Iteration 8) — Q3=B: 호스트 옵션으로 노출하지 않음.
    defaultNightIntroSeconds = 5

    // NEW (Iteration 8) — Q5=A: NIGHT→DAY 진입 시 사망 안내 시간을 위해
    // resolveNight() 가 추가로 부여하는 토론 시작 전 버퍼.
    defaultDayIntroSeconds = 5
)
```

`Options` 구조체는 변경하지 않음 (Q3=B / Q5=A — 노출 없음).

---

## 4. `nightStepSeconds` 분기

```go
// internal/game/types.go
func nightStepSeconds(opts Options, step NightStep) int {
    var v, def int
    switch step {
    case NightStepIntro:
        // NEW: 항상 도메인 상수. 호스트 설정 노출 없음.
        return defaultNightIntroSeconds
    case NightStepMafia:
        v, def = opts.NightMafiaSeconds, defaultNightMafiaSeconds
    case NightStepPolice:
        v, def = opts.NightPoliceSeconds, defaultNightPoliceSeconds
    case NightStepDoctor:
        v, def = opts.NightDoctorSeconds, defaultNightDoctorSeconds
    default:
        return 0
    }
    if v <= 0 {
        return def
    }
    return v
}
```

---

## 5. `enterNight()` 진입점

```go
// internal/game/resolve_night.go
func (e *engine) enterNight() []EventEnvelope {
    now := e.clock.Now()
    e.state.Phase = PhaseNight
    // ... (clear pending fields — 변경 없음)
    e.state.NightStep = ""
    e.state.NightStepDeadline = time.Time{}
    e.state.LastTickAt = now

    events := []EventEnvelope{pub(PhaseChanged{Phase: PhaseNight, Day: e.state.Day})}
    // CHANGED (Iteration 8): MAFIA 가 아니라 INTRO 부터 시작.
    events = append(events, e.beginNightStep(NightStepIntro, now)...)
    return events
}
```

이벤트 발행 순서:
1. `PhaseChanged{NIGHT, Day}` — 카탈로그 `phase.night` cue 발화 ("밤이 되었습니다…")
2. `NightStepChanged{INTRO, Day, Deadline=now+5s}` — 카탈로그 silent (Q4=B)
3. (5초 경과 후 Tick 에 의해) `NightStepChanged{MAFIA, Day, Deadline=now+5s+30s}` — 카탈로그 `night.mafia` cue 발화 ("마피아의 시간입니다…")

---

## 6. `nextNightStep` 표

```go
// internal/game/resolve_night.go
func nextNightStep(s NightStep) NightStep {
    switch s {
    case NightStepIntro:    // NEW
        return NightStepMafia
    case NightStepMafia:
        return NightStepPolice
    case NightStepPolice:
        return NightStepDoctor
    case NightStepDoctor:
        return NightStepResolved
    default:
        return NightStepResolved
    }
}
```

| 현재 | 다음 | 비고 |
|---|---|---|
| INTRO | MAFIA | 신규 (Iteration 8) |
| MAFIA | POLICE | 변경 없음 |
| POLICE | DOCTOR | 변경 없음 |
| DOCTOR | RESOLVED | 변경 없음 (RESOLVED 시 `resolveNight()` 호출) |
| (그 외) | RESOLVED | terminal |

---

## 7. `resolveNight()` Deadline 식

```go
// internal/game/resolve_night.go
func (e *engine) resolveNight() ([]EventEnvelope, error) {
    now := e.clock.Now()
    // ... (victim 결정 / 상태 리셋 — 변경 없음)
    e.state.Day++
    e.state.Phase = PhaseDay
    // CHANGED (Iteration 8): 사망 발표 cue 가 흐를 시간을 부여한 뒤 토론 시작.
    e.state.Deadline = now.Add(time.Duration(
        defaultDayIntroSeconds + e.state.Settings.DiscussionSeconds,
    ) * time.Second)
    // ... (events 발행 — 변경 없음)
}
```

`transitionIntroToDay`(첫째날 진입) 은 변경 없음 — DeathAnnounced/PeacefulNight 가 발행되지 않으므로 버퍼가 불필요.

`DiscussionTimerTick` 임계값(30/10/0) 은 `Deadline` 에서 역산하므로 자동 보정됨. 사용자 체감 토론 길이 = `defaultDayIntroSeconds + DiscussionSeconds` = 기본 5+180 = 185 초.

---

## 8. `handlePauseGame` INTRO 거부

옵션 — `canPause` 가 `Phase` 만 받기 때문에 NightStep 까지 보려면:
- 옵션 1: `canPause(s State) bool` 로 시그니처 변경 — 호출자 1곳만 갱신
- 옵션 2: `handlePauseGame` 내부에서 `Phase + NightStep` 명시 검사

**채택**: 옵션 2 (변경 면적 최소화).

```go
// internal/game/handlers_lifecycle.go
func (e *engine) handlePauseGame(a PauseGame) (State, []EventEnvelope, error) {
    if err := ensureHost(&e.state, a.HostID); err != nil {
        return e.state.Clone(), nil, err
    }
    if !canPause(e.state.Phase) {
        return e.state.Clone(), nil, errf(CodeWrongPhase,
            "cannot pause during phase %s", e.state.Phase)
    }
    // NEW (Iteration 8): INTRO 안내 단계는 Pause 불가 (Q6=B).
    if e.state.Phase == PhaseNight && e.state.NightStep == NightStepIntro {
        return e.state.Clone(), nil, errf(CodeWrongPhase,
            "cannot pause during night intro")
    }
    if e.state.Paused { // idempotent
        return e.state.Clone(), nil, nil
    }
    // ... (변경 없음)
}
```

`handleResumeGame` / `canPause(p Phase)` 는 변경 없음.

---

## 9. 카탈로그 (U2 영향, 본 patch 에는 노트로만)

`internal/announce/catalog_default.go` 의 `NightStepChanged` switch 에 `NightStepIntro` 케이스를 추가하고 `return Announcement{}` (silent) — Q4=B. U2 patch 에서 처리.

---

## 10. Wire / UI (U3/U5 영향, 본 patch 에는 노트로만)

- U3: `NightStep` 직렬화는 string passthrough — 변경 없음. 추가 검증 테스트만.
- U5: `wire.ts` 의 NightStep 유니온에 `"INTRO"` 추가. Picker 들은 정확한 step 명 비교라 자동 잠금 호환.

---

## 11. 테스트 케이스 (I8-T1 ~ I8-T7)

| ID | 시나리오 | 검증 |
|---|---|---|
| I8-T1 | NIGHT 진입 직후 NightStep 검사 | `state.NightStep == NightStepIntro` 이고 `NightStepDeadline = now + 5s` |
| I8-T2 | INTRO 만료 → Tick → MAFIA 자동 진입 | `nextDeadline = introDeadline + NightMafiaSeconds`, `NightStepChanged{MAFIA}` emit |
| I8-T3 | `nightStepSeconds(Options, NightStepIntro)` | 항상 5 (Options 무시) |
| I8-T4 | `resolveNight()` 후 Day Deadline | `Deadline = now + (defaultDayIntroSeconds + DiscussionSeconds) * Second` |
| I8-T5 | 첫째날 (`transitionIntroToDay`) 의 Day Deadline | `Deadline = now + DiscussionSeconds * Second` (버퍼 없음) |
| I8-T6 | `PauseGame` 가 INTRO 단계에서 거부 | `CodeWrongPhase`, "cannot pause during night intro" |
| I8-T7 | legacy snapshot 호환 | NightStep=MAFIA 인 스냅샷을 Restore 후 정상 진행 (INTRO 우회 가능, Tick 동작) |

테스트 헬퍼 변경:
- `internal/game/handlers_night_test.go::advanceToNight` — VOTE 종료 직후 NightStep=INTRO 진입을 인식하고 `clock.Advance(NightIntroSeconds + 1ms)` + `Tick` 으로 MAFIA 까지 자동 진행. 호출 측 테스트는 변경 없음.
- `internal/game/iteration5_test.go::TestI5_PauseShiftsNightDeadline` — 헬퍼가 MAFIA 까지 진행시키므로 기존 동작 유지.

---

## 12. 영향 받는 파일 (예상)

| 파일 | 변경 종류 | 라인 수 (대략) |
|---|---|---|
| `internal/game/types.go` | enum 1건, 상수 2건, switch 분기 1건 | +6 |
| `internal/game/resolve_night.go` | enterNight step / nextNightStep / resolveNight Deadline | +8 -2 |
| `internal/game/handlers_lifecycle.go` | handlePauseGame INTRO 거부 1 분기 | +5 |
| `internal/game/handlers_night_test.go` | advanceToNight 헬퍼 INTRO 자동 진행 | +6 |
| `internal/game/iteration8_test.go` (신규) | I8-T1~T7 | +200 |

다른 테스트(`iteration4_test`, `iteration5_test`, `tick_test`, `resolve_night_test`) 는 헬퍼 갱신만으로 회귀 PASS 예상.

---

## 13. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 |
