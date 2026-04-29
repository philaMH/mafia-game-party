# Iteration 4 — Execution Plan

**Started**: 2026-04-29
**Scope**: 사용자 지시 4건의 도메인/UI 변경
**Approach**: Brownfield Patch (단위 5개 구조 유지)

---

## 1. 변경 요구사항 (User Intent)

| ID | 요구사항 | 결정 |
|---|---|---|
| R1 | 첫쨰날 낮에도 투표가 있어야 한다 | INTRO → DAY1 → VOTE → NIGHT1 → DAY2 → … 흐름. 첫째 날은 사망/평화 통보 생략. |
| R2 | 밤은 마피아 → 경찰 → 의사 순서로 진행 | `State.NightStep` 추가. 마피아 제출 전엔 경찰/의사 거부, 경찰 제출 전엔 의사 거부. 의사 제출 시 자동 `resolveNight()`. |
| R3 | 낮이 되면 전날 밤 결과를 통보 | `resolve_night.go` 이벤트 순서 정리: `PhaseChanged{DAY}` 먼저, 이어서 `DeathAnnounced` 또는 `PeacefulNight`. |
| R4 | 경찰은 이전 페이즈 조사결과를 확인 | `State.PoliceHistory []PoliceCheckRecord` 누적. snapshot 마스킹은 경찰 본인에게만 노출. |

## 2. 영향 모듈

- **U1 `internal/game`**
  - `types.go` — `NightStep` 타입, `State.NightStep`, `State.PoliceHistory`, `PoliceCheckRecord`
  - `event.go` — `NightStepChanged` 이벤트 (사회자 안내용)
  - `handlers_lifecycle.go` — `transitionIntroToDay1` 신규, 초기 진입 NIGHT 대신 DAY1
  - `handlers_night.go` — 단계 강제, 단계별 입력 거부 에러
  - `handlers_day_vote.go` — 첫째 날 VOTE 후 NIGHT로 전환 (기존 `tally`가 DAY → NIGHT 흐름 처리하도록 확인)
  - `resolve_night.go` — 이벤트 순서 변경, 단계 초기화
  - `tally.go` — VOTE 종료 후 NIGHT로 전환 (현재는 처형/끝/재투표 처리). 첫 DAY VOTE 흐름 검토.
  - `apply.go` — `allNightActionsSubmitted()` 제거 또는 단순화, 단계별 진입 시점에 자동 진행 로직
  - `engine.go` — Start 시 초기 phase=`PhaseIntro` 유지, 첫 INTRO 종료 흐름은 lifecycle 핸들러가 담당
- **U2 `internal/announce`**
  - `catalog_data.go` — `msgNightStepMafia/Police/Doctor`, `msgNightSummaryDeath`, `msgNightSummaryPeaceful`(첫째 날 외) 메시지 추가
  - `catalog_default.go` — `NightStepChanged`, `PhaseChanged{DAY}` 시 첫째 날과 일반 낮 분기 처리
- **U3 `internal/transport/ws`**
  - 신규 이벤트 직렬화: `NightStepChanged` (필요 시)
  - snapshot 마스킹: `PoliceHistory`는 경찰 본인 시점에만 포함
- **U5 `web/`**
  - `types/wire.ts` — 새 타입/이벤트 추가
  - `context/reducer.ts` — `policeHistory` 상태 누적 (PoliceResult 들어올 때마다 추가)
  - `views/PlayerView/PolicePicker.tsx` — history 전체 렌더 (예: "1일차 밤: A=마피아", "2일차 밤: B=시민")
  - `views/PlayerView/NightInputs.tsx` — 단계별 자기 차례 표시. 마피아/경찰/의사 차례가 아니면 "기다리세요" UI
  - `views/PublicView/PhaseHeader.tsx` — 밤 단계 표시(선택)

## 3. 작업 단계 (Plan-level Checklist)

### Phase A: U1 게임 코어 도메인 변경
- [x] A1. `types.go` — `NightStep`, `State.NightStep`, `PoliceCheckRecord`, `State.PoliceHistory` 추가
- [x] A2. `event.go` — `NightStepChanged` 이벤트 추가
- [x] A3. `handlers_lifecycle.go` — `transitionIntroToDay` 신규(첫 DAY 진입), `transitionIntroToNight` 폐기
- [x] A4. `engine.go` — Start 흐름 영향 없음 (PhaseIntro 진입 그대로)
- [x] A5. `handlers_night.go` — 단계 강제 (`CodeWrongPhase`). PoliceHistory append. 의사 제출 시 자동 resolve.
- [x] A6. `resolve_night.go` — `PhaseChanged{DAY}` 먼저 발행, 이어서 사망/평화. NightStep 초기화. `enterNight` helper 신설(NightStep=Mafia + 자동 스킵)
- [x] A7. `tally.go` — VOTE 종료 후 `enterNight()` 사용
- [x] A8. `apply.go` — `allNightActionsSubmitted` 제거
- [x] A9. 단위 테스트 추가 — `iteration4_test.go` (5 케이스): I4-T1 IntroToDay1, I4-T2 NightSequence_MafiaFirst, I4-T3 AutoSkipsDeadRole, I4-T4 ResolveNight_EventOrder, I4-T5 PoliceHistory_AccumulatesAcrossNights. 기존 테스트는 새 흐름(advanceToNight: INTRO→DAY1→VOTE→NIGHT)에 맞춰 수정.
- [x] A10. `go test ./internal/game/... -count=1` PASS, 커버리지 91.0%

### Phase B: U2 announce 카탈로그
- [x] B1. `catalog_data.go` 신규 문구 추가 (`msgPhaseDayFirst`, `msgNightStepMafia/Police/Doctor`, 사망/평화 메시지 사용자 예시 톤으로 조정)
- [x] B2. `catalog_default.go` `NightStepChanged` 처리
- [x] B3. `PhaseChanged{DAY, Day=1}` 분기 — 첫째 날 전용 자막
- [x] B4. 단위 테스트 — `TestRender_Day1UsesDedicatedSubtitle`, `TestRender_NightStepChanged` 추가
- [x] B5. `go test ./internal/announce/... -count=1` PASS, 커버리지 93.9%

### Phase C: U3 transport (snapshot 마스킹)
- [x] C1. `internal/session/view.go` — `BuildPrivateView`에서 경찰이 아닌 viewer의 `view.PoliceHistory = nil`
- [x] C2. `internal/transport/ws/protocol.go` `eventPayload.Step` 추가 + `dispatch.go` `NightStepChanged` 케이스
- [x] C3. `protocol_test.go` — `TestBuildEventPayload_AllKinds`/`NightStepCarriesStep` 갱신, 전 패키지 PASS

### Phase D: U5 web 프론트엔드
- [x] D1. `types/wire.ts` — `NightStep`, `PoliceCheckRecord`, `State.nightStep/policeHistory`, `EventPayload.NightStepChanged` 추가
- [x] D2. `context/reducer.ts` — `NightStepChanged` 처리, `PoliceResult` 시 `policeHistory` 누적, `PhaseChanged` 시 `nightStep` 초기화
- [x] D3. `PolicePicker.tsx` — `state.policeHistory` 누적 표시. `lastResult` 의존 제거
- [x] D4. `NightInputs.tsx`/`Mafia/Doctor/Police Picker` — `state.nightStep` 검사 후 본인 차례가 아니면 비활성화 + 안내문
- [x] D5. (생략) PhaseHeader 변경 없음 — sub-step 안내는 카탈로그 자막 + TTS로 전달
- [x] D6. `npm test` PASS (41 tests, 신규 reducer 테스트 3건 포함)

### Phase E: 통합 검증
- [x] E1. `go test ./... -count=1` 6 패키지 PASS
- [x] E2. `go build -o /tmp/mafia-game-iter4 ./cmd/mafia-game` 성공 (15 MB)
- [x] E3. `npm run build` 성공, gzip 61.31 KB (이전 61.63 KB 대비 -0.32 KB)
- [x] E4. 커버리지: announce 93.9%, game 91.0%, persistence 80.2%, session 86.5%, transport/http 89.8%, transport/ws 83.3%. transport/ws baseline 미달은 본 작업과 무관(`broadcastRoomClosed` 0% 등 기존 누락 분).
- [x] E5. plan 체크박스 [x] 처리, audit.md 결과 append, aidlc-state.md Iteration 4 갱신

---

## 4. NFR / 호환성 영향

- **상태 호환성**: `State`에 새 필드 2개 추가. 기존 snapshot 복원 시 NightStep 빈 값/PoliceHistory nil 처리 필요. 영속화 layer는 JSON이므로 자동 누락 허용 → 별도 마이그레이션 불요.
- **외부 의존**: 추가 없음.
- **커버리지 정책**: NFR-U2-M1 ≥ 85% 유지.
- **결정 변경**: 사회자 진행 버튼 도입 없음 — Iteration 4 Q1 결정에 따라 자동 진행 유지.

## 5. 위험 / 고려사항

- 첫째 날 DAY VOTE에서 처형 발생 시 첫 NIGHT 진입 직전 `Eliminated` 이벤트가 정상 발행되어야 함 → 기존 `tally.go` 검토 필수.
- `EndNightEarly` 액션 폐기 또는 No-op 처리 — 호스트 컨트롤 UI에서 "야간 마감" 버튼 제거 필요.
- PoliceHistory 영속화로 snapshot 사이즈 증가. 한 게임 최대 ~12명, NIGHT 수 한정이므로 무시 가능.
