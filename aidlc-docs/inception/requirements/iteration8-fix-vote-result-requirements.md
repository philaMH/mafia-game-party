# Iteration 8 — Fix · 밤 진입 안내 (worktree-fix+vote-result) Requirements v1.0

**Status**: Draft v1.0 — 사용자 승인 대기
**Branch**: `worktree-fix+vote-result`
**Workflow Date**: 2026-04-29
**Predecessor**: Iteration 7 Voice 개편 + Host 메인 메뉴 (PR#2/PR#3 머지 완료)
**Type**: Bug Fix (UX · Domain Timing)
**Risk Level**: Low–Medium

---

## 1. 결함 보고 (Intent)

### 1.1 사용자 보고 원문
> 낮에 투표가 진행된 후 밤으로 넘어갈 때 바로 마피아의 시간이 시작되기 때문에 플레이어들이 안내를 받고 밤을 준비하는 동안 마피아의 시간이 촉박합니다. 플레이어들에게 밤이 되었음을 안내한 뒤 마피아의 시간을 시작하고 싶습니다.

### 1.2 결함 진단

#### 1.2.1 NIGHT 진입 (주 결함)
- **위치**: `internal/game/resolve_night.go::enterNight()`
- **현재 동작**: VOTE 종료 → `tally()` → `enterNight()` 1 호출 사이클에서:
  1. `state.Phase = PhaseNight` 설정
  2. `pub(PhaseChanged{Phase: PhaseNight, Day: D})` emit → 카탈로그 `phase.night` cue 발화 ("밤이 되었습니다…")
  3. 즉시 `beginNightStep(NightStepMafia, now)` 호출 → `NightStepDeadline = now + NightMafiaSeconds` 확정 → `pub(NightStepChanged{Step: MAFIA, Deadline: D})` emit → 카탈로그 `night.mafia` cue 발화 ("마피아의 시간입니다…")
- **결과**: 호스트의 오디오 큐(`useAudioCueQueue`)는 두 cue 를 FIFO 직렬 재생하지만, **마피아 카운트다운 타이머는 `beginNightStep` 호출 시각부터 이미 흐름**. 두 mp3 cue 합산 ~ 5–7초 가량 안내가 진행되는 동안 마피아가 사용 가능한 시간이 잠식됨 (대략 30s → 23s 수준).

#### 1.2.2 NIGHT→DAY 전이 (잠재 동일 결함)
- **위치**: `internal/game/resolve_night.go::resolveNight()`
- **현재 동작**:
  1. `Phase = PhaseDay`, `Deadline = now + DiscussionSeconds * Second` 설정
  2. `pub(PhaseChanged{Phase: PhaseDay, Day: D+1, Deadline: D})` emit → `phase.day`/`phase.day.first` cue
  3. `pub(DeathAnnounced{Victim})` 또는 `pub(PeacefulNight{})` emit → `death.announced` 또는 `peaceful.night` cue
- **결과**: 토론 카운트다운이 사망 발표 전에 이미 흐르기 시작. DiscussionSeconds 가 180초로 길어 체감 영향은 작지만 동일 패턴.

### 1.3 사용자 결정 (Q&A 결과)
| 질문 | 답변 | 결정 |
|---|---|---|
| Q1 | A | 신규 `NightStep = INTRO` enum 추가, `INTRO → MAFIA → POLICE → DOCTOR → RESOLVED` |
| Q2 | A | 기본 5초 (`defaultNightIntroSeconds = 5`) |
| Q3 | B | `Options` 필드 추가하지 않음, 도메인 상수 고정 |
| Q4 | B | 신규 mp3 cue 발주 없음 — `NightStepChanged{INTRO}` 카탈로그는 빈 Announcement |
| Q5 | A | NIGHT→DAY 전이도 동일 처리 (`defaultDayIntroSeconds = 5`) |
| Q6 | B | INTRO step 은 Pause 불가 |

---

## 2. 기능 요구사항 (Functional Requirements)

### FR-1. 신규 NightStep `INTRO` 도입
- `NightStep` enum 에 `NightStepIntro NightStep = "INTRO"` 추가.
- 순서: `INTRO → MAFIA → POLICE → DOCTOR → RESOLVED`. `nextNightStep(NightStepIntro) = NightStepMafia`.
- `enterNight()` 가 기존 `beginNightStep(NightStepMafia, now)` 대신 `beginNightStep(NightStepIntro, now)` 호출.
- `beginNightStep` 자체 로직 변경 없음 — `nightStepSeconds(opts, NightStepIntro)` 가 `defaultNightIntroSeconds` 반환.

### FR-2. 도메인 상수 신설
- `internal/game/types.go`:
  - `defaultNightIntroSeconds = 5`
  - `defaultDayIntroSeconds = 5` (FR-3 용)
- `nightStepSeconds(opts Options, step NightStep) int` switch 에 `NightStepIntro` 분기 추가 — 항상 `defaultNightIntroSeconds` 반환 (Options 노출 없음, Q3=B).

### FR-3. NIGHT→DAY 진입 버퍼
- `resolveNight()` 가 설정하는 `e.state.Deadline` 을 `now + (defaultDayIntroSeconds + DiscussionSeconds) * Second` 로 변경.
- 첫째날 (`transitionIntroToDay`) 은 DeathAnnounced/PeacefulNight 를 emit 하지 않으므로 **버퍼 없이 기존 그대로** (`now + DiscussionSeconds * Second`).
- DiscussionTimerTick 임계값 (30/10/0) 은 `Deadline` 에서 역산되므로 자동으로 보정됨 — 별도 변경 불필요.

### FR-4. 카탈로그 (announce)
- `defaultCatalog.Render` 의 `NightStepChanged` 분기에 `NightStepIntro` 케이스 추가, **빈 `Announcement{}` 반환** (Q4=B).
- `PhaseChanged{NIGHT}` 의 기존 `phase.night` cue 는 그대로 유지 — INTRO 단계의 안내 음성을 담당.
- `NightStepResolved` 의 기존 빈값 처리도 그대로 유지.

### FR-5. Pause 정책
- `canPause` 는 `Phase` 단위에서만 판단하면 부족 — `PhaseNight + NightStep == NightStepIntro` 인 경우 Pause 불가.
- `handlePauseGame` 에서 명시적으로 거부: `errf(CodeWrongPhase, "cannot pause during night intro")`.
- (Q6=B 의 정책 의도: 안내 중 일시정지가 의미가 작고, Resume 시 deadline shift 처리만 복잡해짐.)

### FR-6. Wire Protocol (U3)
- `NightStep` 은 문자열로 직렬화되므로 추가 enum 노출만 필요. `protocol.go` 에 별도 상수가 있으면 추가, 없으면 변경 없음.
- 클라이언트(U5) 는 `wire.ts` 의 `NightStep` 유니온 타입에 `"INTRO"` 추가.

### FR-7. 클라이언트 UI (U5)
- `reducer.ts` — 현재 `nightStep` 필드는 단순 문자열 보존이므로 자동 호환. 회귀 테스트로 INTRO 보존 검증 1건 추가.
- `views/PlayerView/NightInputs.tsx` (또는 `MafiaPicker`/`PolicePicker`/`DoctorPicker`) — INTRO 단계에서는 어떤 역할 입력도 활성화하지 않음 (이미 step 명 비교로 잠금되어 있어 자동 호환 가능, 회귀 검증 필요).
- `views/PublicView/PublicView.tsx` — INTRO 단계 진입 시 화면 라벨/서브타이틀 — 기존 `phase.night` 자막이 자연스럽게 노출되므로 TimerBar 가 INTRO 단계 deadline 으로 카운트다운만 표시.

### FR-8. 호환성/마이그레이션
- 영속(SQLite) 스냅샷 호환: 기존 INTRO 가 없는 스냅샷이 복구되어 NightStep 값이 `MAFIA` 인 경우 그대로 동작 (transition 함수가 변경된 것은 신규 진입에만 영향). 회귀 테스트 1건 (recovery snapshot with NightStep=MAFIA still proceeds normally).

---

## 3. 비기능 요구사항 (Non-Functional)

| 항목 | 요구치 |
|---|---|
| NFR-Perf | 추가 5초 인터벌이 게임 페이스에 큰 영향 없음 (1 일 cycle 기준 +10초 = ~3% 증가) |
| NFR-Compat | Iteration 7 산출물(Voice 카탈로그/wire/UI) 회귀 0 |
| NFR-Test | `go test ./...` 6 패키지 PASS, `npm test` 50/50 → 신규 테스트 추가 후 PASS, race detector PASS |
| NFR-Coverage | game 패키지 90% 이상 유지 (현 91.7%), announce 93%↑ 유지 (현 94.0%), reducer.ts 90%↑ 유지 (현 90.72%) |
| NFR-Build | `go build ./cmd/mafia-game` 성공, `npm run build` 성공, JS gzip 65 KB 대 유지 |

---

## 4. 영향 분석 (Impact Map)

| 단위 | 변경 | 산출물 |
|---|---|---|
| **U1 Game Core** | NightStep enum / 도메인 상수 2건 / `nightStepSeconds` 분기 / `enterNight` step 변경 / `nextNightStep` / `resolveNight` Deadline 계산 / `canPause` 또는 `handlePauseGame` INTRO 금지 / 신규 `iteration8_test.go` | FD patch + Code Gen |
| **U2 Session/Announce** | catalog `NightStepChanged{NightStepIntro}` 빈값 분기 추가, catalog 테스트 1건 보강 | FD patch + Code Gen |
| **U3 Realtime Transport** | (검증) NightStep 직렬화가 unknown enum 안전. wire enum 통과 확인. 추가 테스트 1건 | FD patch (Minimal) + Code Gen |
| **U4 HTTP Bootstrap** | SKIP | — |
| **U5 Web Frontend** | `wire.ts` NightStep 유니온, reducer 회귀 테스트, PlayerView Picker 가드 회귀 테스트 | FD patch + Code Gen |

---

## 5. 추적 매트릭스 (Traceability)

| Req | 단위 | 코드 위치 (예정) | 테스트 |
|---|---|---|---|
| FR-1 | U1 | `internal/game/types.go`, `resolve_night.go` | I8-T1 (NightStep=INTRO 진입), I8-T2 (INTRO→MAFIA 자동 전이) |
| FR-2 | U1 | `internal/game/types.go::nightStepSeconds` | I8-T3 (defaultNightIntroSeconds 적용) |
| FR-3 | U1 | `internal/game/resolve_night.go::resolveNight` | I8-T4 (resolveNight Deadline 버퍼), I8-T5 (Day1 transitionIntroToDay 버퍼 없음) |
| FR-4 | U2 | `internal/announce/catalog_default.go` | I8-A1 (NightStepChanged{INTRO} silent) |
| FR-5 | U1 | `internal/game/handlers_lifecycle.go::handlePauseGame` | I8-T6 (INTRO 단계에서 PauseGame 거부) |
| FR-6 | U3 | `internal/transport/ws/protocol.go` (변경 시) | (자동: 기존 직렬화로 통과) |
| FR-7 | U5 | `web/src/types/wire.ts`, reducer, PlayerView Picker | I8-W1 (NightStepChanged{INTRO} reducer), I8-W2 (Picker 가드) |
| FR-8 | U2 | `internal/persistence/recovery.go` (검증만) | I8-T7 (legacy snapshot with NightStep=MAFIA still works) |

---

## 6. Definition of Done

- [ ] `internal/game` 단위 — INTRO step 진입 + 5초 hold + MAFIA 자동 전이 검증 (I8-T1~T2)
- [ ] `internal/game` 단위 — defaultNightIntroSeconds / defaultDayIntroSeconds 상수 적용 검증 (I8-T3~T5)
- [ ] `internal/game` 단위 — INTRO step 에서 PauseGame 거부 (I8-T6)
- [ ] `internal/game` 단위 — legacy snapshot 호환 (I8-T7)
- [ ] `internal/announce` — `NightStepChanged{INTRO}` silent 검증 (I8-A1)
- [ ] U5 — wire 타입 갱신 + reducer/Picker 회귀 (I8-W1~W2)
- [ ] `go test ./...` 6 패키지 PASS, race detector PASS
- [ ] `npm test` PASS (신규 케이스 포함), `npm run build` 성공
- [ ] `go build -o /tmp/mafia-game-iter8 ./cmd/mafia-game` 성공
- [ ] aidlc-docs 동기화 (audit, aidlc-state, plan, build-and-test results)
- [ ] 사용자 승인 게이트 통과

---

## 7. Out of Scope (명시적 비포함)

- mp3 cue 추가 녹음 (Q4=B)
- `Options` 화면 노출 (Q3=B)
- INTRO 단계의 Pause/Resume 지원 (Q6=B)
- DAY→VOTE 전이 버퍼 — 토론 종료 시 사용자가 명시적으로 종료를 누르는 흐름이라 안내 누락 위험 없음
- VOTE→RECOUNT 전이 버퍼 — 동일

---

## 8. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성, 사용자 답변 Q1=A/Q2=A/Q3=B/Q4=B/Q5=A/Q6=B 반영 |
