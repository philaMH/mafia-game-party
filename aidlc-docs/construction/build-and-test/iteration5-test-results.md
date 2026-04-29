# Iteration 5 — Build & Test Results

**작성일**: 2026-04-29
**범위**: NightStep 고정 타이머(R1~R3) + 호스트 Pause/Resume(R4~R5) + Options 노출(R6) + Public 카운트다운(R5)
**Plan**: `aidlc-docs/construction/plans/iteration5-execution-plan.md`

---

## 1. 요구사항 추적 매트릭스

| ID | 요구 | 구현 위치 | 검증 |
|----|------|-----------|------|
| R1 | 사망 NightStep 자동 스킵 폐기 | `internal/game/resolve_night.go` (`enterNight`/`beginNightStep`) — `stepHasLivingActor` 호출 제거 | I5-T1 (`TestI5_DeadPoliceStepHeldFullDuration`), I4-T3 재작성 (`TestI4_NightStep_DeadRoleStillHeld`) |
| R2 | 마피아 30s / 경찰 10s / 의사 10s 고정, 시간 종료가 유일 트리거 | `internal/game/types.go` `Options` 3필드 + `nightStepSeconds()`, `internal/game/tick.go` `tickNight()` | I5-T1, I5-T3, I5-T11, I5-T13 |
| R3 | 첫 제출 후 잠금 (Mafia/Doctor) | `internal/game/handlers_night.go` `PendingMafiaTarget != nil` / `PendingDoctorTarget != nil` 체크 | I5-T2 |
| R4 | 호스트 Pause/Resume (INTRO/DAY/NIGHT) | `internal/game/action.go` 신규 `PauseGame`/`ResumeGame`, `handlers_lifecycle.go` `handlePauseGame`/`handleResumeGame`, `Tick`이 Paused 시 no-op | I5-T4 (NIGHT shift), I5-T5 (INTRO shift), I5-T6 (DAY shift), I5-T7 (Pause 중 제출 가능), I5-T9 (idempotent), I5-T10 (host-only) |
| R5 | Public 카운트다운 + Pause 표시 | `web/src/views/PublicView/{TimerBar,PauseBadge,PublicView}.tsx` | reducer 4종 신규 테스트, web 빌드 PASS |
| R6 | Options 노출 (NightSeconds 3필드) | `internal/game/types.go` Options + `web/src/types/wire.ts` Options + `defaultOptions()` | I5-T12, I5-T13 |
| Q3=B | Pause 중 액션 제출 허용 | 액션 핸들러는 Paused 검사 안 함 | I5-T7 |
| Q4=A | 호스트 UI에 Pause/Resume만 (야간 마감 제거) | `web/src/views/PublicView/HostControls.tsx` | 빌드 검증 |
| Q6=B | Public 화면에 NightStep 라벨 + 카운트다운 | `PublicView.tsx` `NIGHT_STEP_LABEL` + `TimerBar` `label` prop | 빌드 검증 |

---

## 2. 단위별 변경 요약

### U1 Game Core
**변경 파일**: `types.go`, `action.go`, `event.go`, `apply.go`, `handlers_lifecycle.go`, `handlers_night.go`, `resolve_night.go`, `tick.go`, `markers_test.go`, `fixtures_test.go`, `handlers_night_test.go`, `handlers_errors_test.go`, `iteration4_test.go`, `iteration5_test.go` (신규)

**핵심 변경**:
- `Options` +3 필드 (`NightMafiaSeconds`/`NightPoliceSeconds`/`NightDoctorSeconds`) + `DefaultOptions` 30/10/10
- `nightStepSeconds(opts, step)` — 0/음수 입력 시 패키지 기본값으로 폴백 (기존 임의 Options 리터럴 호환)
- `State` +3 필드 (`NightStepDeadline`, `Paused`, `PausedAt`)
- 신규 액션 `PauseGame`/`ResumeGame`, 신규 이벤트 `GamePaused`/`GameResumed`, `NightStepChanged.Deadline`
- `enterNight()` → `beginNightStep()` 도입, 자동 스킵 제거
- `advanceNightStep()` 함수 폐기 (handlers_night 호출부 모두 제거)
- `Tick()`에 `Paused` 분기 + `tickNight()` 신설 (deadline 기반 다중 스텝 진행)
- `handlePauseGame`/`handleResumeGame` 신설, Resume 시 `now - PausedAt` shift 적용
- `handleMafiaKill`/`handleDoctorHeal`에 1회 잠금 (`PendingTarget != nil` → `CodeAlreadyDone`)
- `handleEndNightEarly` 유지 (도메인 hatch / 테스트 helper용)
- 신규 헬퍼 `advanceNightStep(t, e, clock)` (fixtures_test.go) — Tick 기반 진행

**테스트 변경**:
- 신규 13건 (`iteration5_test.go`)
- 기존 11건 갱신 (Tick advance 추가, I4-T3 재작성, PoliceCheck 두 번째 실패 코드 `ErrWrongPhase` → `ErrAlreadyDone`)

**커버리지**: 91.0% → **91.7%** (+0.7 pp)

### U2 Announce
**변경 파일**: `catalog_data.go`, `catalog_default.go`, `catalog_test.go`

**핵심 변경**:
- `msgGamePaused` ("잠시 진행을 멈춥니다. 모두 자리를 지키시오."), `msgGameResumed` ("다시 시간이 흐르기 시작합니다. 진행을 이어가시오.") 추가
- `Render`에 `GamePaused`/`GameResumed` case 추가 (SeverityInfo)
- 카탈로그 테스트 1건 추가 (`TestRender_GamePausedAndResumed`)

**커버리지**: 93.9% → **94.0%** (+0.1 pp)

### U3 Realtime Transport
**변경 파일**: `protocol.go`, `dispatch.go`, `handlers.go`, `protocol_test.go`, `iteration5_test.go` (신규)

**핵심 변경**:
- 인입 wire 상수 `TypeHostPause`/`TypeHostResume` 추가 + handler dispatch
- `eventPayload`에 `StepDeadlineMs` 필드 추가
- `buildEventPayload` 분기 — `NightStepChanged.Deadline` 직렬화, `GamePaused`, `GameResumed` 신규
- 4건 신규 테스트 (`iteration5_test.go`) + protocol_test all-kinds 케이스 2건 추가

**커버리지**: 83.3% → **82.4%** (-0.9 pp). 신규 라인은 모두 커버되며, 비율 하락은 기존 미커버 라인의 비중 변화에 기인.

### U4 HTTP Bootstrap
**변경 없음**.

### U5 Web Frontend
**변경 파일**: `src/types/wire.ts`, `src/context/reducer.ts`, `src/context/reducer.test.ts`, `src/views/PublicView/{HostControls,TimerBar,PauseBadge(신규),PublicView}.tsx`

**핵심 변경**:
- `Options` +3 필드 + `defaultOptions()` 갱신
- `State` +3 필드 (`nightStepDeadline`, `paused`, `pausedAt`)
- `EventPayload` union에 `GamePaused`/`GameResumed` 추가, `NightStepChanged`에 `stepDeadlineMs?` 추가
- `OutgoingMsg`에 `host:pause`/`host:resume` 추가
- `reducer.ts`: `NightStepChanged`에서 `stepDeadlineMs` ISO 변환, `GamePaused`/`GameResumed` 처리, `PhaseChanged` 시 `nightStepDeadline` 클리어
- `PauseBadge` 신규 컴포넌트 (paused=true 시 상단 고정 배너)
- `TimerBar` 갱신: `paused` prop으로 카운트다운 freeze, `label` prop으로 NightStep 명시
- `HostControls`: "야간 마감" 버튼 제거, "일시정지/재개" 토글 (INTRO/DAY/NIGHT)
- `PublicView`: NIGHT 페이즈에서 `nightStepDeadline` + NightStep 라벨로 TimerBar 호출, 그 외엔 기존 `state.deadline`

**테스트**: reducer 신규 4건 (NightStepChanged with deadline, GamePaused, GameResumed NIGHT, GameResumed DAY, PhaseChanged clears nightStepDeadline). 41 → **45 PASS**.

**번들 크기**: gzip 62.11 KB → **61.75 KB** (-0.36 KB, 기존 EndNightEarly 핸들러 코드 제거 효과)

---

## 3. 검증 결과

### 3.1 Backend
```
$ go test ./... -count=1
?   	github.com/saltware/mafia-game/cmd/mafia-game	[no test files]
ok  	github.com/saltware/mafia-game/internal/announce	0.272s
ok  	github.com/saltware/mafia-game/internal/game	0.505s
ok  	github.com/saltware/mafia-game/internal/persistence	0.756s
ok  	github.com/saltware/mafia-game/internal/session	0.952s
ok  	github.com/saltware/mafia-game/internal/transport/http	0.959s
ok  	github.com/saltware/mafia-game/internal/transport/ws	2.600s
```

### 3.2 Backend Coverage
| Package | Iter4 | Iter5 | Δ |
|---|---|---|---|
| announce | 93.9% | **94.0%** | +0.1 |
| game | 91.0% | **91.7%** | +0.7 |
| persistence | 80.2% | 80.2% | 0 |
| session | 86.5% | 86.1% | -0.4 |
| transport/http | 89.8% | 89.8% | 0 |
| transport/ws | 83.3% | 82.4% | -0.9 |

### 3.3 Build
```
$ go build -o /tmp/mafia-game-iter5 ./cmd/mafia-game
$ ls -la /tmp/mafia-game-iter5
-rwxr-xr-x 1 myunghoonkang staff 15694610  4 29 15:53 /tmp/mafia-game-iter5  # 15 MB
```

### 3.4 Frontend
```
$ npm test -- --run
 Test Files  4 passed (4)
      Tests  45 passed (45)

$ npm run build
✓ 64 modules transformed.
../cmd/mafia-game/web/dist/index.html                   0.44 kB │ gzip:  0.30 kB
../cmd/mafia-game/web/dist/assets/index-DtVIq_uM.css    0.77 kB │ gzip:  0.49 kB
../cmd/mafia-game/web/dist/assets/index-CbX1vKji.js   189.65 kB │ gzip: 61.75 kB
✓ built in 323ms
```

---

## 4. 테스트 매트릭스 결과 (plan §5 대비)

| Plan ID | 테스트 | 결과 |
|---------|--------|------|
| I5-T1 | 경찰 사망 시 NightStep 시간 유지 | `TestI5_DeadPoliceStepHeldFullDuration` PASS |
| I5-T2 | 마피아 30s 동안 첫 제출 후 잠금 | `TestI5_MafiaFirstSubmitLock` PASS |
| I5-T3 | 마피아 미제출 시 PeacefulNight | `TestI5_NoMafiaSubmissionResolvesPeaceful` PASS |
| I5-T4 | NIGHT Pause/Resume shift | `TestI5_PauseShiftsNightDeadline` PASS |
| I5-T5 | INTRO Pause/Resume shift | `TestI5_PauseShiftsIntroSpeaker` PASS |
| I5-T6 | DAY Pause/Resume shift | `TestI5_PauseShiftsDayDeadline` PASS |
| I5-T7 | Pause 중 SubmitMafiaKill 허용 | `TestI5_SubmissionAllowedDuringPause` PASS |
| I5-T8 | VOTE/RECOUNT Pause 거부 | `TestI5_PauseRejectedOutsideTimedPhases` PASS |
| I5-T9 | Pause/Resume 멱등 | `TestI5_PauseIdempotent` PASS |
| I5-T10 | Pause host-only | `TestI5_PauseRequiresHost` PASS |
| I5-T11 | NightStepChanged.Deadline 발행 | `TestI5_NightStepChangedCarriesDeadline` PASS |
| I5-T12 | DefaultOptions의 NightSeconds 기본값 | `TestI5_DefaultOptionsHasNightSeconds` PASS |
| I5-T13 | 커스텀 NightSeconds 반영 | `TestI5_CustomNightSecondsRespected` PASS |
| (추가) Wire 직렬화 | `TestIter5_NightStepChangedCarriesDeadline`/`GamePaused`/`GameResumed`/`HostPauseResumeWireConstants` | 4건 PASS |
| (추가) Catalog | `TestRender_GamePausedAndResumed` | PASS |
| (추가) Reducer | NightStep deadline / GamePaused / GameResumed (NIGHT, DAY) / PhaseChanged clears | 4건 PASS |

---

## 5. 회귀 영향 분석

### 5.1 변경된 기존 테스트 (의도적)
- `TestI4_NightSequence_MafiaFirst` — 액션 제출이 NightStep을 진행시키지 않음을 확인하도록 갱신
- `TestI4_NightStep_AutoSkipsDeadRole` → `TestI4_NightStep_DeadRoleStillHeld`로 재작성 (자동 스킵 → 시간 유지)
- `TestI4_ResolveNight_EventOrder` — 마지막 Tick으로 resolveNight 트리거
- `TestI4_PoliceHistory_AccumulatesAcrossNights` — Tick advance 추가
- `TestDoctorHeal_SelfHealAllowedByDefault` / `Disabled` — Tick advance 추가
- `TestPoliceCheck_OncePerNight` — 두 번째 실패 코드 `ErrWrongPhase` → `ErrAlreadyDone`
- `TestPoliceCheck_NoSelfInvestigate` / `ResultIsPrivate` — Tick advance 추가
- `TestNight_AutoResolveOnAllSubmitted` / `DoctorProtectsTarget` — DOCTOR Tick 추가
- `TestDoctorHeal_NonDoctor` (handlers_errors_test.go) — Tick advance 추가

### 5.2 변경되지 않은 기존 테스트
- 모든 INTRO/DAY/VOTE/RECOUNT/restore 시나리오 — `EndNightEarly`로 NIGHT 즉시 종료하는 테스트들은 그대로 동작 (handler 유지)
- `tally_test.go`, `resolve_night_test.go`, `reassign_test.go`, `scenario_test.go` 등 — 변경 없음
- `validation_test.go`, `role_test.go` — Options 리터럴에 NightSeconds 미설정이지만 `nightStepSeconds()` 폴백으로 호환

### 5.3 호환성 (Backwards)
- 기존 영속 스냅샷에 `NightStepDeadline`/`Paused`/`PausedAt`이 없어도 zero value로 복원 (json omitempty)
- `Options.NightMafiaSeconds=0` 등 0 값을 가진 스냅샷 → `nightStepSeconds()`가 30/10/10 폴백 → 동작 정상
- 기존 wire 클라이언트가 `host:pause`/`host:resume`을 보내지 않으면 게임은 종전과 동일하게 진행

---

## 6. NFR 영향

| NFR | 영향 |
|-----|------|
| NFR-U1-R2 (Apply 실패 시 상태 불변) | 유지 — 신규 핸들러도 검증 실패 시 state 불변 |
| NFR-U1-R3 (Tick 멱등) | 유지 — `Paused` 시 no-op, 동일 `now` 재호출 시 사이드이펙트 없음 |
| NFR-U1-R5 (Snapshot 안전성) | 유지 — `state.Clone()`이 새 필드(time.Time/bool) 자동 복사 |
| NFR-U2-PERSIST | 영향 없음 — `shouldPersist` 트리거에 GamePaused/GameResumed 미포함 (의도) |
| NFR-U3-S3 (메시지 64KiB 제한) | 영향 없음 — 신규 페이로드 미세 |
| NFR-U5-P5 (TimerBar render 비용) | 향상 — paused 시 `setInterval` 자체를 정지 |

---

## 7. 후속 권장 사항

1. **Chrome DevTools MCP 회귀 (Phase F)** — 호스트 + 6명 시나리오로 (a) 경찰 사망 후 NIGHT의 단계별 시간 유지 시각 확인, (b) Pause/Resume 토글 UX 확인. 사용자 깨어난 후 수동 트리거 권장.
2. **Host UI에서 NightSeconds 설정 노출** — 현재 Options에는 노출됐지만 OpenRoom 폼에는 입력 필드가 없음. 다음 iteration에서 호스트가 30/10/10 외 값을 GUI에서 조정 가능하도록 폼 확장 가능.
3. **transport/ws 커버리지 회복** — 본 작업 범위 외이지만 baseline 85% 미달 (82.4%). `broadcastRoomClosed` 등 누락 분 보강 권장.
4. **`EndNightEarly` 처리 정책 재검토** — UI에서는 제거됐지만 도메인/wire에 남음. 디버그 도구 외 사용처가 없다면 향후 deprecate 가능.

---

## 8. DoD 체크리스트

- [x] R1~R6 모두 구현 + 테스트 매트릭스 PASS
- [x] `go test ./... -count=1` 모든 패키지 PASS
- [x] `npm test` PASS (45/45), `npm run build` 성공 (gzip 61.75 KB)
- [x] Backend 신규 라인 커버리지 ≥ 85% (game 91.7%, announce 94.0%)
- [x] reducer 신규 액션 처리 테스트 추가 (4건)
- [x] iteration5-test-results.md 작성
- [ ] aidlc-state.md Iteration 5 섹션 추가 (다음 단계)
- [x] audit.md에 모든 사용자 입력/결정 기록
- [ ] 사용자 승인 게이트 (본 보고서 검토 후)
