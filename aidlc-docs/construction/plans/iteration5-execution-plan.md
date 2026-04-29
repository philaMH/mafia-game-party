# Iteration 5 — Execution Plan

**작성일**: 2026-04-29  
**유형**: Brownfield Light Patch (도메인 + wire + 클라이언트)  
**입력**: 사용자 결함 보고 (audit.md "Iteration 5 — Intake") + Q1~Q7 답변

---

## 1. 목표 (사용자 요구사항)

**R1. NightStep 자동 스킵 제거 (사망자 정보 누설 차단)**
- 현재 `setNightStep()`은 `stepHasLivingActor`가 false면 즉시 다음 단계로 자동 전환. 결과: 경찰/의사가 죽으면 그 단계가 즉시 스킵되어 사망 사실이 유추됨.
- 수정: 모든 NightStep은 사망 여부와 무관하게 정해진 시간 동안 유지된다.

**R2. NightStep 고정 타이머**
- 마피아 30초 / 경찰 10초 / 의사 10초 (기본값)
- 시간이 모두 흘러야만 다음 단계로 자동 전환 (Q1=A: 시간 종료가 유일 트리거)
- 행동 제출은 단계 진행을 앞당기지 않음

**R3. 행동 제출 1회 잠금**
- 마피아/경찰/의사의 첫 제출 후 동일 단계 내 추가 제출 거부 (Q2=B)
- 기존 last-write-wins 동작은 폐기

**R4. 호스트 일시정지 / 재개**
- Pause/Resume 두 버튼만 호스트 화면에 노출 (Q4=A)
- INTRO 발언 타이머 / DAY 토론 타이머 / NIGHT step 타이머 모두 일시정지 대상 (Q5=B)
- 일시정지 중에도 클라이언트는 액션 제출 가능 (Q3=B)

**R5. Public 화면에 NightStep 카운트다운 표시 (Q6=B)**
- 현재 단계명 + 남은 시간 표시
- INTRO 발언자/DAY 토론과 별도의 NightStep deadline 사용

**R6. 타이머 값을 Options 필드로 노출 (Q7=B)**
- `NightMafiaSeconds`, `NightPoliceSeconds`, `NightDoctorSeconds` 필드 추가
- 기본값 30/10/10
- 호스트가 OpenRoom 시 변경 가능

---

## 2. 영향 범위

| 단위 | 파일/모듈 | 변경 정도 |
|------|-----------|----------|
| U1 Game Core | `internal/game/{types,action,apply,event,handlers_lifecycle,handlers_night,resolve_night,tick,state_clone}.go` + 신규 `iteration5_test.go` | **대** |
| U2 Session/Announce | `internal/announce/{catalog_data,catalog_default}.go` + `internal/session/view.go` (선택) | 소 |
| U3 Realtime Transport | `internal/transport/ws/{protocol,handlers,dispatch}.go` + `iteration5_test.go` | 중 |
| U4 HTTP Bootstrap | 없음 | 0 |
| U5 Web Frontend | `web/src/types/wire.ts`, `web/src/context/reducer.ts`, `web/src/views/PublicView/{HostControls,TimerBar,PauseBadge(신규),PublicView,PhaseHeader}.tsx`, `web/src/views/PlayerView/PhaseInputs.tsx`(선택) | 중 |

`EndNightEarly`/`submit:end-night` (호스트 야간 마감) — Q4=A 결정으로 **호스트 UI에서 버튼 제거**. 도메인/wire 레벨에서는 호환을 위해 유지하되 `HostControls`에서 노출하지 않음(타이머가 자동 진행하므로 사용 불필요). 호스트가 일시정지를 원하면 Pause 버튼으로 처리.

---

## 3. 설계 결정

### 3.1 도메인 (U1)

#### 3.1.1 Options 추가
```go
type Options struct {
    // ...existing fields...
    NightMafiaSeconds  int `json:"nightMafiaSeconds"`
    NightPoliceSeconds int `json:"nightPoliceSeconds"`
    NightDoctorSeconds int `json:"nightDoctorSeconds"`
}
```
- `DefaultOptions`은 30/10/10 기본값 채움
- 0/음수 입력 시 `validation.go`에서 거부 (또는 `recommendedNightSeconds()`로 보정)

#### 3.1.2 State 추가
```go
type State struct {
    // ...existing fields...
    NightStepDeadline time.Time `json:"nightStepDeadline,omitempty"` // 현재 NightStep의 만료 시각
    Paused            bool      `json:"paused"`
    PausedAt          time.Time `json:"pausedAt,omitempty"`
}
```
- `Pause` action 처리 시 `Paused=true`, `PausedAt=now`
- `Resume` action 처리 시 `shift = now - PausedAt`을 다음 timer 필드들에 가산:
  - `IntroSpeakerStartedAt += shift` (INTRO 중)
  - `Deadline += shift` (DAY 중)
  - `NightStepDeadline += shift` (NIGHT 중)
- `Tick`: `Paused`이면 `LastTickAt`만 갱신하고 즉시 반환 (deadline 비교 안 함)

#### 3.1.3 Action 신설
```go
type PauseGame struct {
    sealedAction
    HostID PlayerID
}

type ResumeGame struct {
    sealedAction
    HostID PlayerID
}
```
- 둘 다 host-only 권한 (`ensureHost`)
- Pause: 이미 paused면 거부 / 또는 멱등 (선택: 멱등 처리)
- Resume: paused 아니면 거부 / 또는 멱등
- Phase ∈ {LOBBY, END}일 때는 거부 (의미 없음)

#### 3.1.4 Event 신설
```go
type GamePaused struct {
    sealedEvent
    Phase Phase
}

type GameResumed struct {
    sealedEvent
    Phase    Phase
    Deadline time.Time // 활성 timer의 갱신된 deadline (없으면 zero)
}
```
- 둘 다 VisPublic
- catalog는 별도 안내 문구 (msgGamePaused/msgGameResumed) 추가

#### 3.1.5 NightStep 진행 흐름 변경
- `enterNight()`: NightStep=MAFIA, NightStepDeadline=now+MafiaSeconds, **자동 스킵 호출 안 함**
- `setNightStep()`에서 `stepHasLivingActor` 체크 + 재귀 스킵 **삭제**
- `advanceNightStep()` 함수 자체 **폐기** (`handleMafiaKill/handlePoliceCheck/handleDoctorHeal` 호출부 모두 제거)
- 모든 단계 전환은 `Tick`에서만 일어남: `now >= NightStepDeadline && !Paused`이면 다음 단계로
- DOCTOR 단계 만료 시 `resolveNight()` 호출
- `NightStepChanged` 이벤트의 페이로드에 `Deadline time.Time` 추가

#### 3.1.6 1회 제출 잠금
- `handleMafiaKill`: `PendingMafiaTarget != nil`이면 `CodeAlreadyDone`
- `handlePoliceCheck`: `PoliceCheckedThisNight` 분기는 이미 존재 — 그대로 유지
- `handleDoctorHeal`: `PendingDoctorTarget != nil`이면 `CodeAlreadyDone`

#### 3.1.7 EndNightEarly 동작 정리
- `handleEndNightEarly`는 도메인에 남기지만, U5 Host UI에서 노출 제거
- 향후 회귀 안전망(테스트/관리 도구)으로 활용 가능

### 3.2 Announce/View (U2)
- `catalog_data.go`에 두 문구 추가: `msgGamePaused`, `msgGameResumed`
- `catalog_default.go`에 `GamePaused`/`GameResumed` 케이스 추가
- 톤 예: `"진행이 잠시 멈춥니다."` / `"진행을 다시 시작합니다."` (사용자 톤과 일치)
- BuildPrivateView 변경 없음 (Pause는 view 마스킹과 무관)

### 3.3 Wire (U3)

#### 3.3.1 Outgoing(client→server) 추가
- `host:pause` (게임 중 일시정지)
- `host:resume`

#### 3.3.2 Event payload 변경
- `NightStepChanged`에 `deadlineMs` 추가
- 신규 kind: `GamePaused {phase}`, `GameResumed {phase, deadlineMs}`
- `EventPayload` 타입 union 갱신 (web/src/types/wire.ts)

#### 3.3.3 snapshot.state 변경
- Backend State JSON에 `paused`, `pausedAt`, `nightStepDeadline` 추가됨에 따라 wire State도 동일 필드 노출
- `Options`에 3개 필드 추가

### 3.4 Web Frontend (U5)

#### 3.4.1 reducer.ts
- `applyEvent`에서 신규 케이스 처리
  - `GamePaused`: `state.paused=true`
  - `GameResumed`: `state.paused=false`, INTRO/DAY/NIGHT의 deadline 갱신 (서버가 새 deadline 반영한 snapshot/이벤트 보내므로 클라이언트는 단순 반영)
  - `NightStepChanged`: `state.nightStep=ev.step`, `state.nightStepDeadline=ev.deadlineMs>0 ? ISO : undefined`
- `applyIncoming` `snapshot` 처리에서 `paused`/`nightStepDeadline` 그대로 전달

#### 3.4.2 PublicView
- **`PauseBadge` 신규 컴포넌트**: `state.paused`이면 화면 상단에 "일시정지 중" 배지 + 음영
- `PhaseHeader`: NIGHT 중일 때 NightStep 라벨 추가 ("밤 — 마피아의 시간" 등)
- `TimerBar`: 두 모드 지원
  - DAY/INTRO: 기존 `state.deadline` 사용
  - NIGHT: `state.nightStepDeadline` 사용
  - `paused`이면 카운트다운 멈춤(현재 표시값 고정)
- `HostControls`:
  - 기존 "토론 조기 종료"(DAY) 유지
  - 기존 "야간 마감"(NIGHT) **제거**
  - 신규: "일시정지" / "재개" 토글 버튼 (INTRO/NIGHT/DAY 페이즈에서 표시)

#### 3.4.3 PlayerView
- `PhaseInputs`/NightInputs는 NightStep + paused 상태에 따라 입력 가능 여부 결정
- **Pause 중에도 입력 허용**(Q3=B): paused=true는 picker 잠그지 않음
- 단계 비활성화 조건은 기존과 동일: 본인 역할이 현재 NightStep과 일치 + 첫 제출 전

---

## 4. 실행 순서 (Phase A~F)

### Phase A. U1 도메인 변경
- [ ] A1: Options 3개 필드 + DefaultOptions/validation
- [ ] A2: State 3개 필드 + state_clone 갱신
- [ ] A3: Action 2개(`PauseGame`, `ResumeGame`) + apply.go 분기
- [ ] A4: Event 2개(`GamePaused`, `GameResumed`) + `NightStepChanged.Deadline`
- [ ] A5: enterNight/setNightStep 자동 스킵 제거 + advanceNightStep 폐기 + 핸들러 호출부 제거
- [ ] A6: Tick에 NIGHT step 만료 처리 + Paused 분기
- [ ] A7: 1회 제출 잠금 (Mafia/Doctor 핸들러)
- [ ] A8: handlers_lifecycle.go에 `handlePauseGame`/`handleResumeGame` 추가, INTRO speaker startedAt 보정
- [ ] A9: tick 의 INTRO/Day 분기에 Paused 처리
- [ ] A10: iteration5_test.go 신규 (T1~T6 시나리오)
- [ ] A11: 기존 테스트(advanceToNight helper, scenario_test, handlers_night_test, tick_test, resolve_night_test) 신규 흐름에 맞게 갱신

### Phase B. U2 Announce
- [ ] B1: catalog_data에 msgGamePaused/msgGameResumed 상수
- [ ] B2: catalog_default에 두 케이스 추가
- [ ] B3: catalog_test 케이스 2건 추가

### Phase C. U3 Wire
- [ ] C1: protocol.go에 `TypeHostPause`/`TypeHostResume` + eventPayload `DeadlineMs` (NightStepChanged)
- [ ] C2: handlers.go에 두 새 인입 case
- [ ] C3: dispatch.go의 `buildEventPayload`에 `NightStepChanged.Deadline`/`GamePaused`/`GameResumed` 분기
- [ ] C4: protocol_test/dispatch_test/iteration5_test 추가

### Phase D. U5 Web Frontend
- [ ] D1: wire.ts 갱신 (Options 3필드, State 3필드, EventPayload 신규 kind 2종, OutgoingMsg 2종)
- [ ] D2: reducer.ts 신규 이벤트 처리 + paused 반영
- [ ] D3: PauseBadge.tsx 신규
- [ ] D4: HostControls.tsx에서 "야간 마감" 제거 + "일시정지/재개" 버튼 추가
- [ ] D5: TimerBar.tsx에 NightStep 모드 + paused 일시정지
- [ ] D6: PublicView.tsx에 PauseBadge + NightStep 라벨 표시
- [ ] D7: reducer.test.ts 케이스 3건 추가 (GamePaused, GameResumed, NightStepChanged with deadline)

### Phase E. 통합 검증
- [ ] E1: `go build -o /tmp/mafia-game-iter5 ./cmd/mafia-game`
- [ ] E2: `go test ./... -count=1` 모든 패키지 PASS
- [ ] E3: `npm test --prefix web` 기존 + 신규 테스트 PASS
- [ ] E4: `npm run build --prefix web` 성공
- [ ] E5: `aidlc-docs/construction/build-and-test/iteration5-test-results.md` 작성
- [ ] E6: aidlc-state.md Iteration 5 섹션 추가

### Phase F. (선택) Chrome DevTools MCP 회귀 검증
- [ ] F1: 호스트 + 6명 시나리오에서 경찰 사망 후 NIGHT 흐름 확인 (10초 동안 단계 유지되는지 시각 확인)
- [ ] F2: Pause/Resume 동작 확인

---

## 5. 테스트 매트릭스

| ID | 시나리오 | 단위 | 검증 포인트 |
|----|----------|------|-------------|
| I5-T1 | 경찰 사망 상태에서 NIGHT 진입 | U1 | NightStep=POLICE 단계가 정확히 `NightPoliceSeconds`초 유지 후 DOCTOR로 전환. NightStepChanged 두 번 발행 (POLICE → DOCTOR). |
| I5-T2 | 마피아가 30초 안에 target 제출 | U1 | PendingMafiaTarget 즉시 기록되지만 NightStep은 그대로 유지, deadline 변경 없음. 두 번째 SubmitMafiaKill은 `CodeAlreadyDone`. |
| I5-T3 | 마피아 미제출로 30초 경과 | U1 | DOCTOR까지 자동 진행 후 resolveNight → PeacefulNight (mafia 미선택). |
| I5-T4 | NIGHT 중 PauseGame → 5초 대기 → ResumeGame | U1 | NightStepDeadline이 정확히 5초 시프트. Tick은 paused 동안 단계 전환 없음. |
| I5-T5 | INTRO 중 Pause/Resume | U1 | IntroSpeakerStartedAt 시프트, 발언자 전환 시점이 시프트만큼 지연. |
| I5-T6 | DAY 중 Pause/Resume | U1 | Deadline 시프트, DiscussionTimerTick 임계값이 시프트 후 시각에 발화. |
| I5-T7 | Pause 중 SubmitMafiaKill | U1 | 정상 수락 (Q3=B). |
| I5-T8 | Pause 중복 호출 | U1 | 두 번째 PauseGame은 거부 또는 멱등(설계 결정). |
| I5-T9 | Options 0 또는 음수 | U1 | StartGame validation에서 거부. |
| I5-T10 | wire snapshot에 paused/nightStepDeadline 포함 | U3 | JSON 직렬화 검증. |
| I5-T11 | reducer GamePaused/GameResumed/NightStepChanged | U5 | state.paused/nightStepDeadline 반영. |

---

## 6. 모호점 / 결정사항

### 결정 1: Pause 중복 호출 처리
- **결정**: 멱등 (이미 paused면 추가 이벤트 발행 안 함, 에러도 안 냄). Resume도 동일.
- 사유: 호스트 화면에서 더블클릭 등 우발적 입력 보호.

### 결정 2: VOTE/RECOUNT 중 Pause
- **결정**: 거부. VOTE/RECOUNT는 시간 제한이 없으므로 의미 없음.
- Phase ∈ {LOBBY, VOTE, RECOUNT, END}: PauseGame/ResumeGame 모두 `CodeWrongPhase`.

### 결정 3: Pause 중 추가 제출 잠금 후 Resume 시 재제출 가능?
- **결정**: 첫 제출 잠금은 NightStep 단위. Pause/Resume과 무관. 한 번 제출하면 그 단계 끝까지 잠금.

### 결정 4: 사망한 단계 동안 호스트 화면 안내 음성
- 기존 `NightStepChanged` 안내(`msgNightStepPolice` 등)는 그대로 발화. 사망 정보 누설 차단의 핵심.

### 결정 5: NightStepDeadline 만료 시점에 단계 전환 이벤트 순서
- Tick 한 번에 여러 단계가 만료되더라도 순차 처리: MAFIA → POLICE → DOCTOR → RESOLVED. 각 전환마다 NightStepChanged 발행.

---

## 7. DoD (Definition of Done)
- [ ] R1~R6 모두 구현 + 테스트 매트릭스 PASS
- [ ] `go test ./... -count=1` 모든 패키지 PASS
- [ ] `npm test` PASS, `npm run build` 성공
- [ ] Backend 신규 라인 커버리지 ≥ 85%
- [ ] reducer 신규 액션 처리 테스트 추가
- [ ] iteration5-test-results.md 작성
- [ ] aidlc-state.md Iteration 5 섹션 추가
- [ ] audit.md에 모든 사용자 입력/결정/완료 보고 기록

---

## 8. Risk / Mitigation

| Risk | 영향 | 완화 |
|------|------|------|
| 기존 `advanceToNight` 테스트 헬퍼가 NightStep 즉시 진행에 의존 | 다수 테스트 무더기 실패 | helper를 시간 기반 진행으로 갱신 (`engine.Tick(deadline+1ms)` 호출) |
| Pause 시 Tick.LastTickAt 가 흐르지 않으면 Resume 후 단일 Tick에 한 번에 진행될 수 있음 | 의도치 않은 단계 전환 | Pause 동안 LastTickAt만 갱신하지 않고, Resume 시 LastTickAt=now로 명시 reset |
| NightStepDeadline 0값 의미 혼동 | wire 직렬화 오류 | `deadlineMs > 0`만 클라이언트에 노출, omitempty 사용 |
| INTRO speaker 자동 회전과 Pause 시점 동기화 | speaker 1명 발언 시간이 Pause 후 잘못 계산 | tickIntro 진입 시 `Paused` 체크, IntroSpeakerStartedAt 시프트 검증 테스트 |

---

## 9. 사용자 승인 게이트

본 plan은 변경 명세, 결정사항, 테스트 매트릭스를 모두 포함합니다. 승인 시 Phase A부터 순차 실행합니다.

**옵션 1**: 본 plan 그대로 승인 → Phase A 진입  
**옵션 2**: 변경 요청 (특정 결정/순서/테스트 케이스 수정)
