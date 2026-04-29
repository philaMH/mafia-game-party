# AI-DLC State Tracking

## Project Information
- **Project Name**: mafia-game
- **Project Type**: Greenfield
- **Start Date**: 2026-04-25T00:00:00Z
- **Current Stage**: ITERATION 7 완료 (사용자 승인 2026-04-29). 호스트 첫 페이지 분리(메인 메뉴 + `/public/settings`) + localStorage 영속(`mafia.options.v1`) + 신규 wire `host:save-options`. 영향 단위 U2/U3/U5 (U1, U4 SKIP). 구현: U2 SaveHostOptions/SavedHostOptions + 검증기, U3 wire dispatch + ValidationErrors 매핑, U5 신규 라우트 + HostHomeView/HostSettingsView + optionsStorage. 회귀: Go 6 패키지 PASS / session 87.3% / ws 82.9% / npm test 60 PASS / build JS gzip 65.62 KB(+0.69 KB). 부수 결함 2건 수정. Operations 단계는 사용자 트리거 대기.

## Workspace State
- **Existing Code**: No
- **Programming Languages**: 미정 (요구사항 분석 단계에서 결정 예정)
- **Build System**: 미정
- **Project Structure**: Empty (신규 프로젝트)
- **Reverse Engineering Needed**: No
- **Workspace Root**: `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game`

## Code Location Rules
- **Application Code**: 워크스페이스 루트 (절대로 `aidlc-docs/` 안에 두지 않음)
- **Documentation**: `aidlc-docs/` 전용
- **Structure patterns**: 추후 code-generation.md Critical Rules 참고

## Execution Plan Summary
- **Total Stages (실행)**: 7 (Application Design, Units Generation, Functional Design, NFR Requirements, NFR Design, Code Generation, Build and Test)
- **Stages Skipped**: Reverse Engineering (Greenfield), User Stories (단순 페르소나/소규모 도구), Infrastructure Design (단일 바이너리)
- **Risk Level**: Low–Medium

## Extension Configuration
| Extension | Enabled | Decided At |
|---|---|---|
| Security Baseline | No | Requirements Analysis (Q14=B, 사용자 선택) |

## Stage Progress

### 🔵 INCEPTION PHASE
- [x] Workspace Detection
- [x] Reverse Engineering (SKIP — Greenfield)
- [x] Requirements Analysis (사용자 승인 완료, v1.1)
- [x] User Stories (SKIP)
- [x] Workflow Planning (사용자 승인 완료)
- [x] Application Design (사용자 승인 완료, 2026-04-26)
- [x] Units Generation (사용자 승인 완료, 2026-04-26)

### 🟢 CONSTRUCTION PHASE (per-unit loop, 진행 순서: U1 → U2 → U3 → U4 → U5)

#### U1 Game Core
- [x] Functional Design (사용자 승인 완료, 2026-04-26)
- [x] NFR Requirements (사용자 승인 완료, 2026-04-26)
- [x] NFR Design (사용자 승인 완료, 2026-04-26)
- [x] Code Generation (사용자 승인 완료, 2026-04-26 — 21 Go 파일 + 16 테스트, 커버리지 90.4%)

#### U2 Session, Persistence & Announce
- [x] Functional Design (사용자 승인 완료, 2026-04-26)
- [x] NFR Requirements (사용자 승인 완료, 2026-04-26)
- [x] NFR Design (사용자 승인 완료, 2026-04-26)
- [x] Code Generation (사용자 승인 완료, 2026-04-26 — 18 Go 파일 + 14 테스트 + 2 문서, 커버리지 86.5%)

#### U3 Realtime Transport
- [x] Functional Design (사용자 승인 완료, 2026-04-26)
- [x] NFR Requirements (사용자 승인 완료, 2026-04-26)
- [x] NFR Design (사용자 승인 완료, 2026-04-26)
- [x] Code Generation (사용자 승인 완료, 2026-04-26 — 8 ws 코드 + 6 ws 테스트 + U2 확장 + 2 문서, 합산 커버리지 87.4%)

#### U4 HTTP Bootstrap & Static
- [x] Functional Design (사용자 승인 완료, 2026-04-26)
- [x] NFR Requirements (사용자 승인 완료, 2026-04-26)
- [x] NFR Design (사용자 승인 완료, 2026-04-26)
- [x] Code Generation (사용자 승인 완료, 2026-04-26 — 6 httpx 코드 + 6 테스트 + main.go + placeholder + 2 문서, 합산 커버리지 87.6%)

#### U5 Web Frontend
- [x] Functional Design (사용자 승인 완료, 2026-04-26)
- [x] NFR Requirements (사용자 승인 완료, 2026-04-26)
- [x] NFR Design (사용자 승인 완료, 2026-04-26)
- [x] Code Generation (사용자 승인 완료, 2026-04-26 — 49 파일 + npm 의존 11종, gzip 60.14 KB, 핵심 모듈 커버리지 78.72%)

#### 공통
- [ ] Infrastructure Design (SKIP — 단일 바이너리, 인프라 없음)
- [ ] Build and Test (산출물 5종 작성 완료 — build/unit-test/integration/performance/summary, 사용자 승인 대기)

### 🟡 OPERATIONS PHASE
- [ ] Operations (PLACEHOLDER)

---

## Iteration 2 Stage Progress (2026-04-29)

### 🔵 INCEPTION
- [x] Workspace Detection (Brownfield 변경)
- [x] Reverse Engineering — SKIP (Intake CR1=A 결정, 기존 산출물 활용)
- [x] Requirements Analysis — `requirements-iteration2-patch.md` v2.0-patch (사용자 승인 2026-04-29)
- [x] User Stories — SKIP (Iteration 2 Plan)
- [x] Workflow Planning — `iteration2-execution-plan.md` (사용자 승인 2026-04-29)
- [x] Application Design (Partial Update) — `application-design/iteration2-patch.md` (자동 위임 통과)
- [x] Units Generation — SKIP (5단위 구조 유지)

### 🟢 CONSTRUCTION
#### U1 Game Core
- [x] Functional Design Patch — `u1-game-core/functional-design/iteration2-patch.md`
- [x] Code Generation — `internal/game/{types,action,apply,handlers_lifecycle,validation,engine}.go` 변경 + 신규 테스트 6개. 커버리지 90.6%.

#### U2 Session/Persistence/Announce
- [x] Functional Design Patch — `u2-session-persistence-announce/functional-design/iteration2-patch.md`
- [x] Code Generation — `internal/session/host_authority.go` (신규), `session.go`, `lifecycle.go`, `types.go`, `action.go` 변경 + 신규 테스트 7개. 커버리지 87.4%.

#### U3 Realtime Transport (Light)
- [x] Functional Design Patch — `u3-realtime-transport/functional-design/iteration2-patch.md`
- [x] Code Generation — `internal/transport/ws/{protocol,client,handlers,dispatch}.go` 변경 + 통합 테스트 3개. 커버리지 87.0%.

#### U4 HTTP Bootstrap & Static
- [x] 모든 단계 SKIP (변경 없음, 정적 자산은 U5 빌드 산출로 갱신)

#### U5 Web Frontend
- [x] Functional Design Patch — `u5-web-frontend/functional-design/iteration2-patch.md`
- [x] Code Generation — `web/src/types/wire.ts`, `context/reducer.ts`, `views/PublicView/PublicView.tsx`, `views/PlayerView/{PlayerView,IntroView,PhaseInputs}.tsx` 변경 + reducer 테스트 3개. gzip 60.84 KB.

#### 공통
- [x] Build and Test — `aidlc-docs/construction/build-and-test/iteration2-test-results.md`. 모든 회귀 PASS, 빌드 성공.

### 🟡 OPERATIONS
- [ ] Chrome DevTools MCP 다중 컨텍스트 골든패스 (사용자 깨어난 후 수동 트리거 권장)

---

## Post-Construction Maintenance (Cross-Unit Changes)

### LOBBY Membership Events (옵션 A — 도메인 이벤트 정공법)
- **트리거**: 2026-04-27 Chrome DevTools MCP 6+1명 시나리오 검증 중 LOBBY broadcast 부재 결함 재현
- **Plan**: `aidlc-docs/construction/plans/lobby-membership-events-plan.md`
- **영향 단위**: U1 (event 타입), U2 (lifecycle dispatch), U3 (wire 변환), U5 (reducer/뷰)
- **사용자 결정**: 옵션 A 채택, 다음 세션에서 코드 수정 (2026-04-27)
- **상태**: [x] 코드 수정 완료 (2026-04-27 후속 세션, Q1=옵션1 / Q2=session 발행 추천안 채택). Stage A~E DoD 통과:
  - U1: 90.4 % 커버리지 유지, `PlayerJoined` sealed event + 2개 테스트 추가
  - U2: 88.5 % 커버리지 (이전 86.5% → +2.0 pp), `lobbyStateFromMembers` + Subscribe broadcast 검증 2개 테스트
  - U3: 89.3 % 커버리지 유지, `protocol.eventPayload.Name` + `buildEventPayload` PlayerJoined 케이스, 통합 시나리오 테스트 (1 PUBLIC + 1 host + 5 joiner)
  - U5: 79.95 % 커버리지 (reducer.ts 92.2 %), `applyPlayerJoined` (stub init 포함) + `PlayersGrid` LOBBY 표시, gzip 60.23 KB
  - 전체: `go test ./...` PASS, `go build -o /tmp/mafia-game ./cmd/mafia-game` 성공
- **승인**: 2026-04-27T00:59:00Z 사용자 "승인" — 변경 완료
- **검증 완료**: 2026-04-27T01:05:00Z Chrome DevTools MCP 7-context (host + p1..p6) 시나리오 — GM 화면에 7명 실시간 누적 → "게임 시작" 활성 → INTRO 진입 + player 역할 수신까지 정상. 임시 ws workaround 불필요해짐.
- **부수 발견 (별도 plan 필요)**: 마피아 cohort revealed 화면에서 일부 PlayerID 가 raw hex 로 노출 (catalog GetName fallback 발동). 본 작업 범위 외.

---

## Iteration 3 Stage Progress (2026-04-29)

### 🔵 INCEPTION
- [x] Workspace Detection — Brownfield, 기존 산출물/코드 재사용
- [x] Reverse Engineering — SKIP (직전 Iteration 산출물 활용)
- [x] Requirements Analysis — Intake (사용자 결함 보고 → 옵션 A 선택, 본 audit 항목으로 대체)
- [x] User Stories — SKIP (단일 인프라 결함 패치)
- [x] Workflow Planning — 본 항목 (U2 → U3 per-unit 패치)
- [x] Application Design — SKIP (컴포넌트 변경 없음, U2 인터페이스 1개 추가)
- [x] Units Generation — SKIP (5단위 구조 유지)

### 🟢 CONSTRUCTION

#### U2 Session/Persistence/Announce
- [x] Functional Design Patch — `u2-session-persistence-announce/functional-design/iteration3-patch.md` (사용자 승인 2026-04-29T08:55Z)
- [x] NFR Requirements — SKIP (변경 없음)
- [x] NFR Design — SKIP
- [x] Infrastructure Design — SKIP
- [x] Code Generation Plan — `construction/plans/iteration3-code-generation-plan.md` (사용자 승인 2026-04-29T09:05Z)
- [x] Code Generation — `internal/session/{types,session}.go` 변경 + `iteration3_test.go` 6 테스트, 커버리지 88.2% (이전 87.4 → +0.8 pp)

#### U3 Realtime Transport
- [x] Functional Design Patch — `u3-realtime-transport/functional-design/iteration3-patch.md` (사용자 승인 2026-04-29T08:55Z)
- [x] NFR Requirements — SKIP (변경 없음)
- [x] NFR Design — SKIP
- [x] Infrastructure Design — SKIP
- [x] Code Generation Plan — `construction/plans/iteration3-code-generation-plan.md` (사용자 승인 2026-04-29T09:05Z)
- [x] Code Generation — `internal/transport/ws/{dispatch,hub}.go` 변경 + `iteration3_test.go` 5 테스트, 커버리지 87.2% (이전 87.0 → +0.2 pp)

#### U1/U4/U5
- [x] 모든 단계 SKIP (변경 없음 — U5 reducer는 기존 `room:opened`/`snapshot` 핸들러로 자동 커버)

#### 공통
- [x] Build and Test — `aidlc-docs/construction/build-and-test/iteration3-test-results.md` 작성. `go test ./... -count=1` 6 패키지 PASS, `go build -o /tmp/mafia-game-iter3 ./cmd/mafia-game` 성공 (15 MB), `npm test` 38 PASS, `npm run build` gzip 61.63 KB, Chrome DevTools MCP 회귀 검증 PASS. 사용자 승인 게이트 대기.

### 🟡 OPERATIONS
- [ ] (placeholder)

---

## Iteration 4 Stage Progress (2026-04-29)

### 🔵 INCEPTION
- [x] Workspace Detection — Brownfield, 5단위 구조 유지
- [x] Reverse Engineering — SKIP (기존 산출물 활용)
- [x] Requirements Analysis — 사용자 4건 지시 (R1 첫째날 투표 / R2 밤 순차 진행 / R3 낮 시작 시 결과 통보 / R4 경찰 조사 history 누적). 모호점 2건 사용자 답변 수신: R2=자동 진행, R4=history 누적, 사망 단계=(가) 안내 후 자동 스킵.
- [x] User Stories — SKIP
- [x] Workflow Planning — `construction/plans/iteration4-execution-plan.md` 작성, 사용자 시나리오 확인 후 Phase A~E 순차 진행

### 🟢 CONSTRUCTION

#### U1 Game Core
- [x] Functional Design Patch — plan 내 R1~R4 반영 (NightStep enum, State 필드 2개, NightStepChanged 이벤트, resolveNight 이벤트 순서)
- [x] Code Generation — `internal/game/{types,state_clone,event,handlers_lifecycle,handlers_night,resolve_night,tally,apply,tick}.go` 변경. `iteration4_test.go` 5 신규 케이스(I4-T1~T5). 기존 테스트(advanceToNight helper, lifecycle/scenario/tick/errors) 흐름 변경에 맞게 수정. 커버리지 91.0%.

#### U2 Session/Announce
- [x] Functional Design Patch — `BuildPrivateView`에 PoliceHistory 마스킹, 카탈로그에 NightStep 안내 + 첫째날 분기
- [x] Code Generation — `internal/session/view.go`, `internal/announce/catalog_{data,default}.go` 변경. 카탈로그 테스트 2건 추가(`Day1UsesDedicatedSubtitle`, `NightStepChanged`), 1건 메시지 변경 반영. 커버리지 announce 93.9%, session 86.5%.

#### U3 Realtime Transport
- [x] Functional Design Patch — eventPayload에 Step 필드 추가, NightStepChanged 직렬화
- [x] Code Generation — `internal/transport/ws/{protocol,dispatch}.go` 변경. `protocol_test.go` 케이스 추가. 커버리지 83.3% (본 변경 신규 라인은 95.0% 커버; baseline 미달은 기존 `broadcastRoomClosed` 0% 등 누락 분으로 본 작업과 무관).

#### U4 HTTP Bootstrap
- [x] 모든 단계 SKIP (변경 없음)

#### U5 Web Frontend
- [x] Functional Design Patch — wire 타입 갱신, reducer가 NightStepChanged/PoliceResult를 영속 상태에 누적, picker는 `state.nightStep` 기반으로 잠금
- [x] Code Generation — `web/src/types/wire.ts`, `web/src/context/reducer.ts`, `web/src/views/PlayerView/{PhaseInputs,NightInputs,MafiaPicker,DoctorPicker,PolicePicker,PlayerView}.tsx` 변경. reducer 신규 테스트 3건 (NightStepChanged, PoliceResult history 누적, PhaseChanged nightStep 초기화). `npm test` 41 PASS.

#### 공통
- [x] Build and Test — `aidlc-docs/construction/build-and-test/iteration4-test-results.md` 작성. R1~R4 추적 매트릭스, 패키지별 커버리지(announce 93.9% / game 91.0% / persistence 80.2% / session 86.5% / transport/http 89.8% / transport/ws 83.3%), 회귀 영향 분석, NFR 영향, DoD 체크리스트, 후속 권장 사항 포함. `go test ./... -count=1` 6 패키지 PASS, `go build -o /tmp/mafia-game-iter4` 15 MB, `npm test` 41 PASS, `npm run build` gzip 62.11 KB. 사용자 승인 게이트 대기.

---

## Iteration 5 Stage Progress (2026-04-29)

### 🔵 INCEPTION
- [x] Workspace Detection — Brownfield, 5단위 구조 유지
- [x] Reverse Engineering — SKIP
- [x] Requirements Analysis — 사용자 1건 결함 보고 (NightStep 사망자 정보 누설). 모호점 7건 사용자 답변 수신 (Q1=A 시간 종료 트리거 / Q2=B 첫 제출 후 잠금 / Q3=B Pause 중 제출 허용 / Q4=A Pause/Resume 두 버튼만 / Q5=B INTRO/DAY/NIGHT 모두 Pause / Q6=B Public 카운트다운 / Q7=B Options 노출).
- [x] User Stories — SKIP
- [x] Workflow Planning — `construction/plans/iteration5-execution-plan.md` 작성, 사용자 승인 (2026-04-29T12:50:00Z)
- [x] Application Design — SKIP (도메인 인터페이스 변경 — 신규 액션 2개/이벤트 2개, 컴포넌트 추가 없음)
- [x] Units Generation — SKIP (5단위 구조 유지)

### 🟢 CONSTRUCTION

#### U1 Game Core
- [x] Functional Design Patch — plan §3.1 직접 반영 (Options 3필드 / State 3필드 / NightStep enterNight 자동스킵 제거 / advanceNightStep 폐기 / Tick에 tickNight 신설 / Pause·Resume 핸들러)
- [x] Code Generation — `internal/game/{types,action,event,apply,handlers_lifecycle,handlers_night,resolve_night,tick,markers_test,fixtures_test,handlers_night_test,handlers_errors_test,iteration4_test}.go` 변경 + 신규 `iteration5_test.go` (13 케이스). 커버리지 91.0% → 91.7% (+0.7 pp).

#### U2 Session/Announce
- [x] Functional Design Patch — Pause/Resume 안내 문구 (msgGamePaused/msgGameResumed) 추가
- [x] Code Generation — `internal/announce/{catalog_data,catalog_default,catalog_test}.go` 변경. 커버리지 93.9% → 94.0%.

#### U3 Realtime Transport
- [x] Functional Design Patch — wire 신규 `host:pause`/`host:resume` + eventPayload `StepDeadlineMs` + GamePaused/GameResumed 직렬화
- [x] Code Generation — `internal/transport/ws/{protocol,handlers,dispatch,protocol_test}.go` 변경 + 신규 `iteration5_test.go` (4 케이스). 커버리지 83.3% → 82.4% (-0.9 pp; 신규 라인 100% 커버, 비율 하락은 기존 미커버 분 비중 변화).

#### U4 HTTP Bootstrap
- [x] 모든 단계 SKIP (변경 없음)

#### U5 Web Frontend
- [x] Functional Design Patch — wire Options/State/EventPayload/OutgoingMsg 갱신, reducer paused/nightStepDeadline 처리, PauseBadge 신규, TimerBar paused/label, HostControls "야간 마감" 제거 + Pause/Resume 토글, PublicView NIGHT 분기
- [x] Code Generation — `web/src/types/wire.ts`, `web/src/context/{reducer,reducer.test}.ts`, `web/src/views/PublicView/{HostControls,TimerBar,PublicView}.tsx`, 신규 `web/src/views/PublicView/PauseBadge.tsx`. `npm test` 41 → 45 PASS, `npm run build` gzip 62.11 KB → 61.75 KB (-0.36 KB).

#### 공통
- [x] Build and Test — `aidlc-docs/construction/build-and-test/iteration5-test-results.md` 작성. R1~R6 추적 매트릭스, 패키지별 커버리지(announce 94.0% / game 91.7% / persistence 80.2% / session 86.1% / transport/http 89.8% / transport/ws 82.4%), 회귀 영향 분석, NFR 영향, DoD 체크리스트. `go test ./... -count=1` 6 패키지 PASS, `go build -o /tmp/mafia-game-iter5` 15 MB, `npm test` 45 PASS, `npm run build` gzip 61.75 KB. 사용자 승인 게이트 대기.

### 🟡 OPERATIONS
- [ ] Chrome DevTools MCP 다중 컨텍스트 회귀 (사용자 트리거 권장: 경찰 사망 후 NightStep 시간 유지 / Pause·Resume 토글)

---

## Iteration 6 Stage Progress (2026-04-29)

### 🔵 INCEPTION
- [x] Workspace Detection — Brownfield, 5단위 구조 + Iteration 1~5 산출물 보존
- [x] Reverse Engineering — SKIP (기존 산출물 활용)
- [x] Requirements Analysis — `inception/requirements/iteration6-requirements.md` (사용자 승인 2026-04-29T07:55Z, Q1=D / Q2=B / Q3=A / Q4=A)
- [x] User Stories — SKIP (단일 시각 작업)
- [x] Workflow Planning — `construction/plans/iteration6-execution-plan.md` (사용자 승인 2026-04-29T08:05Z)
- [x] Application Design — SKIP (컴포넌트 추가/제거 없음)
- [x] Units Generation — SKIP (5단위 구조 유지)

### 🟢 CONSTRUCTION

#### U1 Game Core / U2 Session/Persistence/Announce / U3 Realtime Transport / U4 HTTP Bootstrap
- [x] 모든 단계 SKIP (Go 코드 변경 없음)

#### U5 Web Frontend
- [x] Functional Design Patch — plan §3 으로 갈음 (Minimal)
- [x] NFR Requirements — SKIP
- [x] NFR Design — SKIP
- [x] Infrastructure Design — SKIP
- [x] Code Generation Plan — plan §5 체크리스트로 갈음
- [x] Code Generation —
  - 신규: `web/src/styles/noir.css` (8.5 KB / 32 클래스), `web/public/assets/background.jpg` (198 KB, 1.9 MB → 90% 감소)
  - 수정: 27 파일 (8 PublicView + 12 PlayerView + 4 components + 3 bootstrap)
  - 검증: `npm test` 45 PASS, `npm run build` 성공 (JS gzip 64.93 KB / CSS gzip 3.21 KB), `go build -o /tmp/mafia-game-iter6` 성공 (15.2 MB), `go test ./...` 6 패키지 PASS

#### 공통
- [x] Build and Test — `aidlc-docs/construction/build-and-test/iteration6-test-results.md` 작성, 사용자 승인 완료 (2026-04-29T08:55Z).

### 🟡 OPERATIONS
- [ ] Chrome DevTools MCP 다중 컨텍스트 회귀 (노이르 배경 가시성 / role-card 5:7 / vote-tile target / PauseBadge pulse / EndScreen dossier 확인 권장)

---

## Iteration 7 Stage Progress (2026-04-29)

### 🔵 INCEPTION
- [x] Workspace Detection — Brownfield, 5단위 구조 + Iteration 1~6 산출물 보존
- [x] Reverse Engineering — SKIP (기존 산출물 활용)
- [x] Requirements Analysis — `inception/requirements/iteration7-requirements.md` v1.0 (사용자 승인 2026-04-29)
- [x] User Stories — SKIP (단일 호스트 페르소나, 작은 UX 분리)
- [ ] Workflow Planning — `construction/plans/iteration7-execution-plan.md` v1.0 (사용자 승인 대기)
- [ ] Application Design — SKIP (도메인 인터페이스 추가 없음, U5 View 신규 + wire 1건만)
- [ ] Units Generation — SKIP (5단위 구조 유지)

### 🟢 CONSTRUCTION (예정 시퀀스: U2 → U3 → U5)

#### U1 Game Core / U4 HTTP Bootstrap
- [ ] 모든 단계 SKIP (변경 없음)

#### U2 Session/Persistence/Announce
- [x] Functional Design Patch — `u2-session-persistence-announce/functional-design/iteration7-patch.md` v1.0 (사용자 승인 2026-04-29)
- [x] NFR Requirements — SKIP
- [x] NFR Design — SKIP
- [x] Infrastructure Design — SKIP
- [x] Code Generation Plan — `construction/plans/iteration7-u2-code-generation-plan.md` v1.0 (사용자 승인 2026-04-29)
- [x] Code Generation — `internal/session/{session.go(수정), host_options.go(신규), export_test.go(신규), iteration7_test.go(신규)}`. 테스트 6 케이스(T1~T6) PASS, race detector PASS, 패키지 커버리지 87.2% (이전 86.1% → +1.1pp). 사용자 승인 게이트.

#### U3 Realtime Transport
- [x] Functional Design Patch — `u3-realtime-transport/functional-design/iteration7-patch.md` v1.0 (사용자 승인 2026-04-29)
- [x] NFR Requirements — SKIP
- [x] NFR Design — SKIP
- [x] Infrastructure Design — SKIP
- [x] Code Generation Plan — `construction/plans/iteration7-u3-code-generation-plan.md` v1.0 (사용자 승인 2026-04-29)
- [x] Code Generation — `internal/transport/ws/{protocol.go(수정), handlers.go(수정 +errorCodeOf ValidationErrors 매핑), iteration7_test.go(신규)}`. 4 통합 테스트(T1~T4) PASS, 6 패키지 회귀 PASS, ws 커버리지 82.3% (≈ baseline 82.4%). 사용자 승인 게이트.

ℹ️ U2 인터페이스 추가 변경: `SavedHostOptions() (game.Options, bool)` 공개 메서드 추가 (테스트 가시성 확보 + 향후 재접속 복원 protocol 활용 대비). 이전 `export_test.go`는 제거.

#### U5 Web Frontend
- [x] Functional Design Patch — `u5-web-frontend/functional-design/iteration7-patch.md` v1.0 (사용자 승인 2026-04-29)
- [x] NFR Requirements — SKIP
- [x] NFR Design — SKIP
- [x] Infrastructure Design — SKIP
- [x] Code Generation Plan — `construction/plans/iteration7-u5-code-generation-plan.md` v1.0 (사용자 승인 2026-04-29)
- [x] Code Generation — 10 파일 변경 (신규 6: `lib/optionsStorage.{ts,test.ts}`, `views/PublicView/{HostHomeView,HostSettingsView}.{tsx,test.tsx}` / 수정 4: `App.tsx`, `types/wire.ts`, `context/GameContext.tsx`(GameContext export 추가 + hostOptions/saveHostOptions), `views/PublicView/PublicView.tsx`). 부수: PublicView의 host:claim useEffect 가드 강화(remount 시 false-positive ACCESS DENIED 방지). `npm run typecheck` PASS, `npm test` 60 PASS (45→60, 신규 15: 8+3+4), `npm run build` 성공 (JS gzip 65.62 KB / +0.69 KB · CSS 3.21 KB 동일), `go build`/`go test ./...` 6 패키지 PASS. 사용자 승인 게이트.

#### 공통
- [x] Build and Test — `aidlc-docs/construction/build-and-test/iteration7-test-results.md` 작성. FR-1~FR-6 + NFR-1~NFR-6 + AC-1~AC-8 추적, 패키지별 커버리지(announce 94.0% / game 91.7% / persistence 80.2% / session 87.3% +1.2pp / transport/http 89.8% / transport/ws 82.9% +0.5pp), 회귀 영향 분석, 부수 결함 2건 수정(errorCodeOf ValidationErrors 매핑 / PublicView remount 가드 강화), DoD 체크리스트 완료. `go test ./... -count=1` 6 패키지 PASS, `go build -o /tmp/mafia-game-iter7` 15 MB, `npm test` 60 PASS (45→60, +15), `npm run build` JS gzip 65.62 KB / +0.69 KB. 사용자 승인 게이트.

### 🟡 OPERATIONS
- [ ] Chrome DevTools MCP 다중 컨텍스트 회귀 (메인 메뉴 → 설정 라우팅 / localStorage 영속 / host:save-options 송수신 확인 권장)

