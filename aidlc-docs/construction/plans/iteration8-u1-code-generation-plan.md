# U1 Game Core · Code Generation Plan — Iteration 8

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/construction/u1-game-core/functional-design/iteration8-patch.md` v1.0 (사용자 승인 2026-04-29T21:50Z)
**Workflow Plan**: `aidlc-docs/construction/plans/iteration8-execution-plan.md` v1.0
**Type**: Bug Fix (Domain timing — NightStep enum 추가)

---

## 1. Step 개요

```
Step A — 도메인 상수 + enum (types.go)
Step B — transition 로직 (resolve_night.go)
Step C — Pause 가드 (handlers_lifecycle.go)
Step D — 테스트 헬퍼 갱신 (handlers_night_test.go: advanceToNight)
Step E — 신규 테스트 케이스 (iteration8_test.go: I8-T1~T7)
Step F — 검증 + audit/state 동기화
```

각 Step 은 독립 변경 단위. Step E 는 Step A~D 의 결과를 활용.

---

## 2. Step A — `internal/game/types.go`

### A.1 NightStep enum 상수 추가
```go
const (
    NightStepIntro    NightStep = "INTRO"     // ← 신규
    NightStepMafia    NightStep = "MAFIA"
    NightStepPolice   NightStep = "POLICE"
    NightStepDoctor   NightStep = "DOCTOR"
    NightStepResolved NightStep = "RESOLVED"
)
```
- 위치: 기존 `NightStep constants.` 블록 최상단

### A.2 도메인 상수 2건
```go
const (
    defaultNightMafiaSeconds  = 30
    defaultNightPoliceSeconds = 10
    defaultNightDoctorSeconds = 10
    defaultNightIntroSeconds  = 5  // ← 신규 (Iteration 8, Q3=B)
    defaultDayIntroSeconds    = 5  // ← 신규 (Iteration 8, Q5=A)
)
```
- 위치: 기존 `Default night step durations` 블록

### A.3 `nightStepSeconds` 분기
```go
func nightStepSeconds(opts Options, step NightStep) int {
    var v, def int
    switch step {
    case NightStepIntro:
        return defaultNightIntroSeconds  // ← 신규: Options 무시
    case NightStepMafia:
        v, def = opts.NightMafiaSeconds, defaultNightMafiaSeconds
    // ...
    }
    // 기존 로직 동일
}
```

### A.4 코멘트 보강
- `NightStep` 타입 docstring 의 "MAFIA -> POLICE -> DOCTOR" 표현을 "INTRO -> MAFIA -> POLICE -> DOCTOR" 로 갱신

### 체크리스트
- [ ] A.1 NightStepIntro 상수 추가
- [ ] A.2 defaultNightIntroSeconds / defaultDayIntroSeconds 상수 2건
- [ ] A.3 `nightStepSeconds` switch 에 INTRO 케이스
- [ ] A.4 NightStep 타입 docstring 갱신

---

## 3. Step B — `internal/game/resolve_night.go`

### B.1 `enterNight()` 시작 step 변경
```go
events := []EventEnvelope{pub(PhaseChanged{Phase: PhaseNight, Day: e.state.Day})}
events = append(events, e.beginNightStep(NightStepIntro, now)...)  // ← MAFIA → INTRO
return events
```

### B.2 `enterNight()` 함수 docstring 갱신
- 기존: "sets NightStep to MAFIA"
- 변경: "sets NightStep to INTRO (a 5s announcement buffer; Tick advances it to MAFIA)"

### B.3 `nextNightStep` switch 에 INTRO 케이스 추가
```go
func nextNightStep(s NightStep) NightStep {
    switch s {
    case NightStepIntro:    // ← 신규
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

### B.4 `resolveNight()` Day Deadline 식
```go
e.state.Phase = PhaseDay
e.state.Deadline = now.Add(time.Duration(
    defaultDayIntroSeconds + e.state.Settings.DiscussionSeconds,
) * time.Second)
```

### B.5 `resolveNight()` 함수 docstring 갱신
- §3 의 deadline 문구를 "now + DayIntroSeconds + DiscussionSeconds (DayIntroSeconds 는 사망 발표 cue 가 흐를 시간을 위한 5초 버퍼)" 로 갱신
- `transitionIntroToDay`(첫째날) 은 변경 없음을 함수 docstring 에 명시

### 체크리스트
- [ ] B.1 enterNight 시작 step
- [ ] B.2 enterNight docstring
- [ ] B.3 nextNightStep INTRO 케이스
- [ ] B.4 resolveNight Deadline 식
- [ ] B.5 resolveNight docstring

---

## 4. Step C — `internal/game/handlers_lifecycle.go`

### C.1 `handlePauseGame` INTRO 거부 분기
```go
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
    if e.state.Paused {
        return e.state.Clone(), nil, nil
    }
    // ... (변경 없음)
}
```

### 체크리스트
- [ ] C.1 INTRO 거부 분기 1건

---

## 5. Step D — `internal/game/handlers_night_test.go::advanceToNight`

### D.1 INTRO 자동 진행
```go
func advanceToNight(t *testing.T, e Engine) State {
    t.Helper()
    state := e.Snapshot()
    // ... (INTRO/DAY1/VOTE 진행 — 변경 없음)
    state = e.Snapshot()
    if state.Phase != PhaseNight {
        t.Fatalf("expected NIGHT after Day 1 vote, got %s", state.Phase)
    }
    // NEW (Iteration 8): Night 진입 직후는 NightStep=INTRO. Tick 으로 MAFIA 까지.
    if state.NightStep == NightStepIntro {
        introDeadline := state.NightStepDeadline
        // FakeClock 접근을 위해 engine 의 clock 을 fixtures 헬퍼로 사용.
        // 헬퍼가 engine.clock 을 모르면, deadline 까지 점프하는 별도 헬퍼 필요.
        if err := tickPastDeadline(e, introDeadline); err != nil {
            t.Fatalf("tickPastDeadline (INTRO): %v", err)
        }
        state = e.Snapshot()
    }
    if state.NightStep != NightStepMafia {
        t.Fatalf("expected NightStep=MAFIA at NIGHT entry, got %q", state.NightStep)
    }
    return state
}
```

### D.2 `tickPastDeadline` 헬퍼 신설 (fixtures_test.go)
```go
// tickPastDeadline advances the engine's FakeClock past the given deadline
// and runs Tick. Used by advanceToNight to drain the INTRO 5s buffer.
func tickPastDeadline(e Engine, deadline time.Time) error {
    // engine 의 clock 은 testEngine 가 *FakeClock 으로 주입한 것.
    impl, ok := e.(*engine)
    if !ok {
        return errors.New("tickPastDeadline: not an *engine")
    }
    fc, ok := impl.clock.(*FakeClock)
    if !ok {
        return errors.New("tickPastDeadline: clock is not *FakeClock")
    }
    fc.T = deadline.Add(time.Millisecond)
    if _, _, err := e.Tick(fc.T); err != nil {
        return err
    }
    return nil
}
```
- **대안**: 기존 `advanceNightStep(t, e, clock)` 헬퍼는 `clock` 인자를 받음. `advanceToNight` 의 시그니처에 clock 을 추가하지 않으려면 D.2 처럼 engine 에서 역참조하거나, advanceToNight 시그니처를 `(t, e, clock)` 로 확장.
- **채택**: D.2 의 `tickPastDeadline` 헬퍼 (시그니처 호환성 우선). 단, engine 내부 접근이 필요하므로 `internal/game` 패키지 내 같은 디렉터리에서 동작.

### D.3 호출 측 영향
- `iteration4_test.go`, `iteration5_test.go`, `tick_test.go`, `resolve_night_test.go`, `iteration5_test.go::TestI5_PauseShiftsNightDeadline` 등 — 모두 `advanceToNight(t, e)` 호출만 하며 헬퍼 내부 변경으로 자동 호환

### 체크리스트
- [ ] D.1 advanceToNight INTRO 자동 진행
- [ ] D.2 tickPastDeadline 헬퍼 신설
- [ ] D.3 회귀 영향 없음 검증

---

## 6. Step E — `internal/game/iteration8_test.go` (신규)

### E.1 테스트 케이스

```go
// I8-T1 — Night 진입 직후 NightStep=INTRO, Deadline = now + 5s
func TestI8_NightStepIntroOnEntry(t *testing.T) { ... }

// I8-T2 — INTRO 만료 후 Tick 으로 MAFIA 자동 전이
//   - 5s 미만 Tick: NightStep=INTRO 유지
//   - 5s + 1ms Tick: NightStep=MAFIA, Deadline = introDeadline + NightMafiaSeconds
//   - NightStepChanged{MAFIA} 이벤트 emit 확인
func TestI8_IntroExpiresToMafia(t *testing.T) { ... }

// I8-T3 — nightStepSeconds(opts, NightStepIntro) == 5 (Options 영향 없음)
func TestI8_NightStepSecondsIntroFixed(t *testing.T) {
    cases := []Options{{}, {NightMafiaSeconds: 60}, {NightMafiaSeconds: 1}}
    for _, opts := range cases {
        if v := nightStepSeconds(opts, NightStepIntro); v != 5 {
            t.Errorf("opts=%+v: nightStepSeconds(INTRO)=%d, want 5", opts, v)
        }
    }
}

// I8-T4 — resolveNight() 후 Day Deadline 에 5초 버퍼 적용
//   - DiscussionSeconds=180 가정 → Deadline-now == 185s
func TestI8_ResolveNightAddsDayIntroBuffer(t *testing.T) { ... }

// I8-T5 — 첫째날 (transitionIntroToDay) 의 Day Deadline 은 버퍼 없음
//   - DiscussionSeconds=180 가정 → Deadline-now == 180s
func TestI8_FirstDayHasNoDayIntroBuffer(t *testing.T) { ... }

// I8-T6 — INTRO 단계에서 PauseGame 거부
//   - Apply 결과: ErrWrongPhase (CodeWrongPhase + "cannot pause during night intro")
//   - Paused 필드 false 유지
func TestI8_PauseDuringIntroRejected(t *testing.T) { ... }

// I8-T7 — legacy snapshot 호환
//   - Restore(state with NightStep=NightStepMafia) 후 Tick → 정상 진행
//   - INTRO 단계 우회 가능, mafia kill 정상 수신
func TestI8_LegacySnapshotMafiaStepStillWorks(t *testing.T) { ... }
```

### E.2 빌드 패턴
- 기존 `iteration5_test.go` / `iteration4_test.go` 의 시드/플레이어셋 헬퍼(`newTestEngine`, `mustStart`, `playerSet(8)`, `allRoles`) 를 그대로 재사용
- `advanceToNight` 호출은 INTRO 까지 통과시키므로 일부 테스트(특히 T1)는 헬퍼를 우회하여 직접 VOTE 종료까지 단계 진행

### 체크리스트
- [ ] E.1 I8-T1
- [ ] E.2 I8-T2
- [ ] E.3 I8-T3
- [ ] E.4 I8-T4
- [ ] E.5 I8-T5
- [ ] E.6 I8-T6
- [ ] E.7 I8-T7

---

## 7. Step F — 검증 + 동기화

### F.1 빌드 / 테스트
- [ ] `go vet ./internal/game/...` PASS
- [ ] `go test ./internal/game/... -count=1 -race` PASS
- [ ] `go test ./... -count=1` 6 패키지 PASS
- [ ] `go test ./internal/game -coverprofile=/tmp/iter8-game.out` → 커버리지 ≥ 91.0% 유지

### F.2 audit/state 동기화
- [ ] audit.md 에 Step A~F 실행 결과 + 커버리지 + 회귀 영향 기록
- [ ] aidlc-state.md U1 Code Generation 체크박스 [x] 마킹

### 체크리스트
- [ ] F.1 빌드/테스트 PASS
- [ ] F.2 audit/state 동기화

---

## 8. 영향 받는 파일 (예상)

| 파일 | 라인 변동 (대략) |
|---|---|
| `internal/game/types.go` | +6 (enum + 상수 + switch) |
| `internal/game/resolve_night.go` | +8 -2 |
| `internal/game/handlers_lifecycle.go` | +5 (Pause 가드) |
| `internal/game/handlers_night_test.go` | +6 (advanceToNight INTRO drain) |
| `internal/game/fixtures_test.go` | +18 (tickPastDeadline 헬퍼) |
| `internal/game/iteration8_test.go` (신규) | +220 (7 테스트) |
| **합계** | **+263 -2** |

---

## 9. RISK / Mitigation

| RISK | 완화책 |
|---|---|
| `tickPastDeadline` 의 engine downcast 가 production 코드 의존 | `internal/game` 패키지 테스트 전용 헬퍼로 격리 (`fixtures_test.go`) |
| 일부 기존 테스트가 deadline 비교 시점을 절대값으로 가정 | `iteration5_test.go::TestI5_*` 의 명시적 시계 점프는 헬퍼 결과의 Snapshot 을 다시 읽도록 검증 |
| `resolveNight` Deadline 변경이 DiscussionTimerTick 임계값(30/10/0) 에 영향 | tickDay 가 deadline 에서 역산하므로 자동 보정. 회귀 검증은 기존 `tick_test.go` 가 담당 |

---

## 10. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 |
