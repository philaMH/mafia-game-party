# Iteration 8 — Build & Test Results

**Status**: 사용자 최종 승인 대기
**Workflow Date**: 2026-04-29
**Branch**: `worktree-fix+vote-result`
**Type**: Bug Fix · UX Domain Timing
**Source Documents**:
- `aidlc-docs/inception/requirements/iteration8-fix-vote-result-requirements.md` v1.0
- `aidlc-docs/construction/plans/iteration8-execution-plan.md` v1.0
- `aidlc-docs/construction/plans/iteration8-u1-code-generation-plan.md` v1.0
- `aidlc-docs/construction/plans/iteration8-u2-code-generation-plan.md` v1.0
- `aidlc-docs/construction/plans/iteration8-u5-code-generation-plan.md` v1.0
- (U3) `aidlc-docs/construction/u3-realtime-transport/functional-design/iteration8-patch.md` v1.0 — 검증 only

---

## 1. 결함 해결 요약

VOTE → NIGHT 전이 시 발생하던 두 결함을 모두 해결:
1. **주 결함**: 마피아 NightStepDeadline 이 `phase.night` 안내 음성과 동시에 시작되어 시간이 잠식되던 현상.
2. **잠재 결함**: NIGHT → DAY 전이 시 사망 발표 음성과 토론 카운트다운이 동시 시작되던 현상.

해결 방식: NightStep enum 에 `INTRO` 단계 추가(5초 무음 버퍼) + `resolveNight()` 의 Day Deadline 에 5초 announcement 버퍼 가산.

---

## 2. FR-1 ~ FR-8 추적 매트릭스

| Req | 단위 | 변경 내용 | 검증 테스트 | 결과 |
|---|---|---|---|---|
| **FR-1** NightStepIntro enum | U1 | `types.go` enum 1건 + `resolve_night.go` enterNight/nextNightStep | I8-T1 (entry NightStep=INTRO), I8-T2 (INTRO→MAFIA 자동 전이) | PASS |
| **FR-2** 도메인 상수 | U1 | `defaultNightIntroSeconds = 5`, `defaultDayIntroSeconds = 5` | I8-T1, I8-T3 (Options 무시) | PASS |
| **FR-3** NIGHT→DAY 버퍼 | U1 | `resolveNight()` Deadline = now + (5 + DiscussionSeconds) * Second | I8-T4 (185s 검증), I8-T5 (첫째날 180s 검증, 버퍼 없음) | PASS |
| **FR-4** 카탈로그 silent | U2 | `catalog_default.go` NightStepIntro 분기 → `Announcement{}` | I8-A1 (`render(NightStepChanged{INTRO})` IsEmpty 단언) | PASS |
| **FR-5** Pause 거부 | U1 | `handlePauseGame` PhaseNight + NightStep == NightStepIntro 거부 | I8-T6 (`PauseGame` ErrWrongPhase) | PASS |
| **FR-6** Wire 직렬화 | U3 | 코드 변경 0건 (string passthrough) | `TestBuildEventPayload_NightStepIntroSerializes` (`"step":"INTRO"` 포함) | PASS |
| **FR-7** UI 라벨 | U5 | `wire.ts` 유니온 + `PublicView::NIGHT_STEP_LABEL` INTRO 라벨 | I8-W1 (reducer NightStepChanged{INTRO}) | PASS |
| **FR-8** Legacy snapshot 호환 | U1 | (코드 변경 없음 — 자연 호환) | I8-T7 (`Restore(NightStep=MAFIA)` 후 Tick 정상) | PASS |

---

## 3. 패키지별 커버리지

| 패키지 | Iteration 7 baseline | Iteration 8 결과 | 변동 |
|---|---|---|---|
| `internal/announce` | 94.0% | **94.3%** | +0.3pp |
| `internal/game` | 91.7% | **91.8%** | +0.1pp |
| `internal/persistence` | 80.2% | 80.2% | 0 |
| `internal/session` | 87.3% | 87.3% | 0 |
| `internal/transport/http` | 90.3% | 90.3% | 0 |
| `internal/transport/ws` | 82.4% | 82.3% | -0.1pp (신규 테스트로 분모 증가, 신규 라인 100% 커버) |

요약: 기존 Iteration 7 baseline 대비 모든 패키지 baseline 미만 없음.

---

## 4. 빌드 & 정적 자산

| 항목 | Iteration 7 baseline | Iteration 8 결과 | 변동 |
|---|---|---|---|
| `go build ./cmd/mafia-game` | 17.97 MB | 17.97 MB | 0 |
| JS gzip (`index-*.js`) | 65.62 KB | 65.62 KB | 0 |
| CSS gzip (`index-*.css`) | 3.21 KB | 3.21 KB | 0 |
| dist/audio | 2.3 MB | 2.3 MB | 0 (신규 mp3 발주 없음, Q4=B) |
| `npm test` | 60 PASS | **66 PASS** | +6 (cumulative; 신규 I8-W1 1건 + 자체 6개 누적) |
| `go test ./...` | 6 패키지 PASS | 6 패키지 PASS | 0 |

---

## 5. 회귀 영향 분석

### 5.1 Iteration 4 (NightStep 도입)
- `tests/iteration4_test.go::TestI4_PoliceHistory_AccumulatesAcrossNights` — NIGHT 2 진입 직후 `drainNightIntro` 호출 추가 (1 라인). 동작 회귀 없음.

### 5.2 Iteration 5 (시간 기반 NightStep / Pause-Resume)
- `tests/iteration5_test.go::TestI5_CustomNightSecondsRespected` — `advanceToNight` 헬퍼가 INTRO 를 1ms 슬롭으로 드레인하므로 MAFIA gap = `NightMafiaSeconds - 1ms` 로 명시 갱신 (3 라인).
- 모든 Pause/Resume 테스트(I5-T4~T11) 는 헬퍼 갱신만으로 회귀 PASS.

### 5.3 Iteration 7 (Voice mp3 cue)
- `phase.night` cue 발화 흐름 변동 없음 (catalog 의 `PhaseChanged{NIGHT}` 분기 그대로).
- INTRO 단계 진입 시 `NightStepChanged{INTRO}` 는 silent → 호스트 mp3 큐는 `phase.night` 만 재생 후 5초 대기, 그 다음 `night.mafia` 재생.
- VoteTallied{Recount}, mafia/citizen 승리 cue 등 다른 분기 영향 없음.

### 5.4 회귀 보정 4건 (테스트만)
- `internal/game/resolve_night_test.go::TestResolveNight_DiscussionDeadlineSet` — Day Deadline 5s 버퍼 반영 (`time.Duration(defaultDayIntroSeconds+180) * time.Second`).
- `internal/game/tick_test.go::TestTick_DayDiscussionDeadlineTransitions` — `clock.Advance` 에 5s 버퍼 추가.
- `internal/game/tick_test.go::TestTick_DiscussionTimerThresholds` — `clock.Advance(150 + 5)` 로 30s remaining 임계 시점 보정.
- `internal/game/iteration5_test.go::TestI5_CustomNightSecondsRespected` — MAFIA gap `45s - 1ms` 명시.

---

## 6. NFR 영향

| NFR | 요구치 | 결과 |
|---|---|---|
| Performance | 1 일 cycle +10s (INTRO 5s + DAY intro 5s) ≈ +3% | 의도된 변동 — 사용자 정책 |
| Compat (영속) | legacy NightStep=MAFIA 스냅샷 호환 | I8-T7 PASS |
| Compat (wire) | 클라이언트가 INTRO 미인식 시 fallback | TimeBar label fallback 무난 (라벨 미정의여도 컴포넌트 렌더 정상) |
| Test | 6 패키지 race PASS | PASS |
| Coverage | game ≥ 91.0%, announce ≥ 93%, reducer ≥ 90% | game 91.8% / announce 94.3% / reducer 90%+ 유지 |
| Build | go binary + JS gzip 65 KB 대 | 17.97 MB / 65.62 KB |

---

## 7. 사용자 체감 흐름 (해결 후)

투표 종료 시점 t=0 에서 (사후 튜닝 2026-04-29T23:35Z 반영 — `defaultNightIntroSeconds = 20`):

| 시각 | Public 화면 | 음성 cue | 플레이어 화면 |
|---|---|---|---|
| t+0 | "밤" 페이즈, TimerBar 20s, 라벨 **"밤이 시작됩니다"** | `phase.night` mp3 발화 (~3초) | NIGHT 진입, picker 모두 비활성 |
| t+20s | TimerBar 30s, 라벨 **"마피아의 시간"** | `night.mafia` mp3 발화 시작 | 마피아 picker 활성 (비-마피아 비활성) |
| t+20s+30s | 라벨 **"경찰의 시간"** + 10s | `night.police` mp3 | 경찰 picker 활성 |
| t+20s+40s | 라벨 **"의사의 시간"** + 10s | `night.doctor` mp3 | 의사 picker 활성 |
| t+20s+50s | DAY 진입 + 사망/평화 발표 | `phase.day` + `death.announced` 또는 `peaceful.night` | DAY 토론 화면 |
| t+20s+50s+5s | 토론 카운트다운 시작 (180s) | (없음) | 토론 진행 |

---

## 8. Definition of Done

- [x] FR-1 NightStepIntro enum 진입 + Tick 자동 전이 (I8-T1, I8-T2)
- [x] FR-2 도메인 상수 적용 + Options 무시 (I8-T3)
- [x] FR-3 resolveNight Day Deadline 5s 버퍼 / 첫째날 버퍼 없음 (I8-T4, I8-T5)
- [x] FR-4 catalog NightStepChanged{INTRO} silent (I8-A1)
- [x] FR-5 INTRO 단계 PauseGame 거부 (I8-T6)
- [x] FR-6 wire INTRO 직렬화 (TestBuildEventPayload_NightStepIntroSerializes)
- [x] FR-7 wire.ts 유니온 + PublicView 라벨 + reducer 회귀 (I8-W1)
- [x] FR-8 legacy snapshot 호환 (I8-T7)
- [x] `go test ./... -count=1 -race` 6 패키지 PASS
- [x] `go build -o /tmp/mafia-game-iter8-final ./cmd/mafia-game` 17.97 MB 성공
- [x] `npm run typecheck` PASS, `npm test` 66 PASS, `npm run build` 65.62 KB 성공
- [x] aidlc-docs 동기화 (audit, aidlc-state, plan, build-and-test)

---

## 9. RISK 결산

| RISK (Plan §4 / U1 §9) | 결과 |
|---|---|
| 기존 헬퍼/테스트의 NightStep=MAFIA 가정 | `advanceToNight` 1곳 갱신으로 회귀 PASS |
| legacy snapshot 호환 | I8-T7 PASS |
| 첫째날 transitionIntroToDay 버퍼 누설 | I8-T5 PASS (180s 그대로) |
| Pause 거부 누락 | I8-T6 PASS |
| `tickPastDeadline`/`engineFakeClock` engine downcast | 테스트 전용 헬퍼로 격리 |
| 기존 deadline 가정 테스트 | 회귀 보정 4건 명시적 갱신 |
| DiscussionTimerTick 임계값 영향 | tickDay 가 deadline 역산 → 자동 보정, `TestTick_DiscussionTimerThresholds` 갱신으로 검증 |

---

## 10. 후속 권장 사항 (OPERATIONS — 사용자 트리거 대기)

- **Chrome DevTools MCP 회귀**: 호스트 + 7 player 다중 컨텍스트로 골든 패스 검증
  1. 투표 종료 → "밤이 되었습니다" mp3 + 5s 무음 → "마피아의 시간" mp3 흐름
  2. 마피아 picker 가 INTRO 동안 비활성, 5초 후 활성화
  3. 경찰/의사 picker 가 정확한 시점에 토글
  4. 사망 발표 → 5초 → 토론 타이머 시작 (Day 2+)
  5. INTRO 중 호스트 Pause 누름 → ACCESS DENIED 또는 거부 메시지 확인
- **첫째날 회귀**: Day 1 진입은 버퍼 없이 토론 즉시 시작 — TimerBar 가 정확히 180s
- **PR 머지 전 빌드 캐시 정리**: `cmd/mafia-game/web/dist/` 가 npm build 결과로 갱신되어 있는지 확인 (이미 22:00 갱신 확인)

---

## 11. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성 — 모든 단위 종료 후 통합 |
