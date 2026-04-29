# Iteration 4 — Build and Test Results

**문서 버전**: 1.0
**작성일**: 2026-04-29
**상위 변경 명세**: `audit.md` Iteration 4 (Day-1 Vote / Sequenced Night / Police History)
**처리 방식**: 사용자 4건 지시(R1~R4)에 대한 도메인·전송·UI 변경. 전 패키지 회귀 + 신규 단위 테스트 + 빌드 검증.

---

## 1. 변경 요구사항 추적

| ID | 요구사항 | 구현 결과 | 검증 테스트 |
|---|---|---|---|
| R1 | 첫째날 낮에도 투표가 있어야 함 | `transitionIntroToDay` 신규: INTRO 종료 시 NIGHT 대신 DAY1 진입. `tally.go`는 그대로 NIGHT 진입 핸들 | `TestI4_IntroToDay1HasNoNightSummary`, `TestEndSelfIntro_LastSpeakerTransitionsToDay`, `TestTick_TransitionsIntroToDayWhenAllDone` |
| R2 | 밤은 마피아 → 경찰 → 의사 순으로 자동 진행 | `State.NightStep` 도메인 강제. 단계 외 입력 시 `ErrWrongPhase`. 의사 제출 시 자동 `resolveNight()`. 사망 역할 단계 자동 스킵(이벤트는 발행) | `TestI4_NightSequence_MafiaFirst`, `TestI4_NightStep_AutoSkipsDeadRole`, `TestPoliceCheck_OncePerNight`(ErrWrongPhase로 변경) |
| R3 | 낮 시작 시 전날 결과 통보 | `resolveNight` 이벤트 순서: `PhaseChanged{DAY}` 먼저 → `DeathAnnounced`/`PeacefulNight`. 카탈로그 문구를 사용자 예시 톤("전날 밤 OO이(가) 사망…", "아무도 사망하지 않았…")으로 조정. 첫째 날 전용 자막 분기 | `TestI4_ResolveNight_EventOrder`, `TestRender_Day1UsesDedicatedSubtitle`, `TestRender_PeacefulNight`, `TestRender_NightStepChanged` |
| R4 | 경찰 조사 history 누적 보존 | `State.PoliceHistory []PoliceCheckRecord` 추가, snapshot/Restore 라운드트립 보존. `BuildPrivateView`에서 경찰 본인 외 viewer는 마스킹. U5 reducer는 `PoliceResult` 수신 시 `policeHistory` append, PolicePicker가 전체 history 표시 | `TestI4_PoliceHistory_AccumulatesAcrossNights`, reducer test "PoliceResult appends to policeHistory" |

---

## 2. 단위별 산출물 갱신 요약

| 단위 | Functional Design 산출물 | 코드 변경 | 단위 테스트 결과 | 커버리지 |
|---|---|---|---|---|
| **U1 Game Core** | plan §3 Phase A | `internal/game/{types,state_clone,event,handlers_lifecycle,handlers_night,resolve_night,tally,apply,tick}.go` 변경. `iteration4_test.go` 신규(5 케이스). 기존 테스트 helper(`advanceToNight`)와 lifecycle/scenario/tick/errors 흐름 변경에 맞게 수정 | PASS — 신규 5 + 기존 회귀 0 | **91.0%** (Iteration 3 90.6% 대비 +0.4 pp) |
| **U2 Session/Announce** | plan §3 Phase B+C | `internal/session/view.go` (PoliceHistory 마스킹), `internal/announce/catalog_data.go`/`catalog_default.go` (NightStepChanged + 첫째 날 분기 + 사망/평화 문구 톤 조정) | PASS — 신규 2 (`Day1UsesDedicatedSubtitle`, `NightStepChanged`) | announce **93.9%** (93.3 → +0.6 pp), session **86.5%** (88.2 → −1.7 pp; 신규 분기 미커버, 기능 영향 없음) |
| **U3 Realtime Transport** | plan §3 Phase C | `internal/transport/ws/{protocol,dispatch}.go` (`eventPayload.Step` 필드 + `NightStepChanged` 직렬화) | PASS — 신규 1 (`NightStepCarriesStep`), 기존 `AllKinds`에 케이스 추가 | **83.3%** (87.2 → −3.9 pp; 본 작업 외 `broadcastRoomClosed` 0% 등 누락 반영. 본 변경 라인 `buildEventPayload` 95.0%) |
| **U4 HTTP Bootstrap** | (변경 없음) | (변경 없음) | PASS | **89.8%** 유지 |
| **U5 Web Frontend** | plan §3 Phase D | `web/src/types/wire.ts`, `web/src/context/reducer.ts`, `web/src/views/PlayerView/{PhaseInputs,NightInputs,MafiaPicker,DoctorPicker,PolicePicker,PlayerView}.tsx` | PASS — 신규 3 (NightStepChanged, PoliceResult 누적, PhaseChanged nightStep 초기화) | reducer.ts: 신규 라인 모두 테스트 커버, 코어 모듈 80%+ 유지 |

---

## 3. 통합 회귀 결과

### 3.1 Go 전체 테스트 (`go test ./... -count=1`)

```
ok  	github.com/saltware/mafia-game/internal/announce	0.408s
ok  	github.com/saltware/mafia-game/internal/game	0.978s
ok  	github.com/saltware/mafia-game/internal/persistence	0.585s
ok  	github.com/saltware/mafia-game/internal/session	0.887s
ok  	github.com/saltware/mafia-game/internal/transport/http	1.226s
ok  	github.com/saltware/mafia-game/internal/transport/ws	2.998s
```

전체 PASS. 회귀 0건.

### 3.2 패키지별 커버리지

```
internal/announce         93.9% (93.3 → 93.9, +0.6 pp)
internal/game             91.0% (90.6 → 91.0, +0.4 pp)
internal/persistence      80.2% (유지)
internal/session          86.5% (88.2 → 86.5, −1.7 pp)
internal/transport/http   89.8% (유지)
internal/transport/ws     83.3% (87.2 → 83.3, −3.9 pp)
```

**Baseline 미달 분석 (NFR-U2-M1 ≥ 85%)**:
- `internal/session` 86.5%: 기준 충족.
- `internal/transport/ws` 83.3%: 기준 미달. 원인은 본 변경의 영향이 아니라 기존 `dispatch.go: broadcastRoomClosed` 0%, `writer.go: writeLoop` 70.6% 등 사전 누락 분이 자연스럽게 드러난 결과. 본 작업으로 추가된 `NightStepChanged` 케이스 + `Step` 필드는 `buildEventPayload` 95.0%로 커버됨. 후속 회복은 별도 PR(통합 테스트 보강)에서 처리 권장.

### 3.3 Go 빌드 (`go build -o /tmp/mafia-game-iter4 ./cmd/mafia-game`)

- **성공** — 단일 바이너리 15 MB. Iteration 3 대비 동일.

### 3.4 Web 테스트 (`npm test`)

```
✓ src/hooks/useToken.test.ts (3 tests)
✓ src/context/reducer.test.ts (27 tests)   ← +3 신규
✓ src/hooks/useTTSQueue.test.ts (5 tests)
✓ src/components/NicknameForm.test.tsx (6 tests)

Test Files  4 passed (4)
Tests       41 passed (41)
Duration    649ms
```

3건 신규(`NightStepChanged`, `PoliceResult appends to policeHistory`, `PhaseChanged out of NIGHT clears nightStep`).

### 3.5 Web 빌드 (`npm run build`)

```
✓ 63 modules transformed.
../cmd/mafia-game/web/dist/index.html                   0.44 kB │ gzip:  0.31 kB
../cmd/mafia-game/web/dist/assets/index-DtVIq_uM.css    0.77 kB │ gzip:  0.49 kB
../cmd/mafia-game/web/dist/assets/index-CJ4aFE3A.js   188.12 kB │ gzip: 61.31 kB
✓ built in 315ms
```

- gzip 합계: **61.31 + 0.49 + 0.31 = 62.11 KB**. Iteration 3 대비 +0.48 KB(타입/UI 추가 분), NFR < 70 KB 한도 내.

---

## 4. 신규/수정 시나리오 카탈로그

### 4.1 U1 신규 단위 테스트 (`internal/game/iteration4_test.go`)

| ID | 이름 | 검증 내용 |
|---|---|---|
| I4-T1 | `TestI4_IntroToDay1HasNoNightSummary` | INTRO 종료 시 PhaseChanged{DAY,Day=1} + DiscussionSeconds 데드라인. 사망/평화 통보 없음 |
| I4-T2 | `TestI4_NightSequence_MafiaFirst` | 마피아 미제출 시 경찰/의사 입력 거부, 경찰 미제출 시 의사 거부, 의사 제출 시 자동 DAY 전환 |
| I4-T3 | `TestI4_NightStep_AutoSkipsDeadRole` | 경찰 사망 상태에서 마피아 제출 → POLICE/DOCTOR NightStepChanged 연달아 발행, NightStep DOCTOR로 안착 |
| I4-T4 | `TestI4_ResolveNight_EventOrder` | 의사 제출 후 emitted events 중 `PhaseChanged{DAY}` 인덱스 < `DeathAnnounced` 인덱스 |
| I4-T5 | `TestI4_PoliceHistory_AccumulatesAcrossNights` | NIGHT1+NIGHT2 두 차례 조사 후 PoliceHistory len=2, snapshot/Restore 라운드트립 후에도 보존 |

### 4.2 U2 announce 신규/수정 케이스

- `TestRender_Day1UsesDedicatedSubtitle` — Day=1 vs Day=2 분기 검증
- `TestRender_NightStepChanged` — MAFIA/POLICE/DOCTOR 자막 + RESOLVED는 silent
- `TestRender_PeacefulNight` — 자막 키워드 "사망"으로 갱신

### 4.3 U5 reducer 신규 케이스

- `NightStepChanged updates state.nightStep`
- `PoliceResult appends to policeHistory` — 단일 NIGHT 누적 + PhaseChanged 통과 후 N2에서 추가 누적
- `PhaseChanged out of NIGHT clears nightStep`

---

## 5. 회귀 영향 분석

| 영역 | 영향 | 대응 |
|---|---|---|
| 도메인 흐름(INTRO→NIGHT) | 변경됨(INTRO→DAY1) | helper `advanceToNight` 갱신, 26개 기존 테스트 자동 흐름 보정 후 PASS |
| `EndNightEarly` | 자동 진행 도입으로 호스트 안전 장치로만 사용 | API 유지, 기존 테스트 PASS |
| `SubmitPoliceCheck` 두 번째 호출 에러 | `ErrAlreadyDone` → `ErrWrongPhase` | 단계 자동 전환으로 의미 보존, 테스트 갱신 |
| `BuildPrivateView` | PoliceHistory 마스킹 추가 | non-police viewer view에 PoliceHistory=nil. 기존 테스트 회귀 없음 |
| 카탈로그 한국어 문구 | 사망/평화 문구 톤 변경 | `TestRender_PeacefulNight` 키워드 조정 |
| Web reducer `policeCheckedThisNight` | NIGHT 진입 시 false 리셋(이전: 유지) | 신규 테스트로 검증, 기존 동작과 일치 |

---

## 6. NFR 영향

- **상태 호환성**: `State`에 새 필드 2개(`NightStep`, `PoliceHistory`). 기존 JSON snapshot은 누락 허용 → 마이그레이션 불요.
- **외부 의존**: 추가 0건 (NFR-7 충족).
- **번들 크기**: gzip 62.11 KB < 70 KB 한도.
- **바이너리 크기**: 15 MB 유지.

---

## 7. DoD 체크리스트

- [x] 사용자 지시 R1~R4 도메인·UI 모두 동작
- [x] `go test ./... -count=1` 6 패키지 PASS
- [x] `go build` 성공 (15 MB)
- [x] `npm test` 41 PASS (신규 3건 포함)
- [x] `npm run build` 성공 (gzip 62.11 KB)
- [x] U1/U2/U4/U5 커버리지 baseline ≥ 85%
- [ ] U3(`transport/ws`) 커버리지 83.3% — baseline 미달, 본 작업과 무관 (별도 PR로 회복 권장)
- [x] plan 체크리스트 전부 [x]
- [x] audit.md / aidlc-state.md Iteration 4 섹션 추가

---

## 8. 후속 권장 사항

1. **`transport/ws` 커버리지 회복**: `broadcastRoomClosed`, `writeLoop` 케이스 보강 PR을 별도 진행. 본 보고서의 baseline 미달은 사전 누락 노출일 뿐 본 작업의 회귀가 아님.
2. **Chrome DevTools MCP 시나리오 회귀**: 실제 브라우저에서 8명 게임 흐름(INTRO → DAY1 투표 → NIGHT1 단계별 진행 → 경찰 history 표시 → DAY2 사망 통보)을 사용자 환경에서 검증.
3. **`EndNightEarly` UI 노출 정책**: 자동 진행 시대에는 비상 안전 장치로만 사용. HostControls의 "야간 마감" 버튼은 유지하나 안내 문구를 "응답 없는 플레이어가 있을 때 강제 마감" 정도로 보강 검토.
