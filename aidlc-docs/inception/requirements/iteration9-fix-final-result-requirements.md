# Iteration 9 — 최종 결과 발표 → 승리 화면 전환 결함 수정 요구사항 v1.0

| 항목 | 내용 |
|---|---|
| 작성 | 2026-04-30T00:35:00Z |
| 상태 | 사용자 승인 대기 |
| 영향 단위 | U1 (Game Core) · U2 (Session/Announce) — **announce 분기 미변경 예상** · U3 (Realtime Transport) — **wire 변경 거의 없음 예상** · U5 (Web Frontend) — **분석 결과 변경 없음 예상, 회귀 테스트만 추가** |
| 작업 브랜치 | `worktree-fix+final-result` |

## 1. 의도 분석 (Intent Analysis)

- **사용자 요청 (원문)**: "투표 결과 또는 전날 밤의 결과가 호스트 화면에서 보여지고 다음 페이즈로 진행되었으면 좋겠습니다. 왜: 투표 또는 전날 밤의 액션으로 승리자가 결정되는 경우, 화면에서 바로 마피아 또는 시민의 승리 화면이 나옵니다. 게임의 흥미를 지속하기 위해 투표 결과 또는 전날 밤의 결과가 먼저 공지되었으면 좋겠습니다."
- **요청 유형**: Bug Fix (UX 결함)
- **범위 추정**: Single Component(엔진) + 회귀 테스트 (전체 5단위 중 U1 한정)
- **복잡도 추정**: Simple — Iteration 8 의 INTRO/Day 버퍼와 동일한 패턴(Tick 기반 지연) 재사용

## 2. 결함 분석 (Root Cause)

### 2.1 코드 경로
- `internal/game/tally.go::applyElimination` (line ~101–118)
  ```go
  events = append(events, pub(Eliminated{...}))
  // ...
  if endEv, ok := e.checkEnd(); ok {
      events = append(events, endEv...)   // ❌ 같은 batch 에서 GameEnded 즉시 발행
      return events
  }
  events = append(events, e.transitionVoteToNight()...)
  ```
- `internal/game/resolve_night.go::resolveNight` (line ~88–140)
  ```go
  events = append(events, pub(PhaseChanged{Phase: PhaseDay, ...}))
  events = append(events, deathFollowUp...)              // DeathAnnounced or PeacefulNight
  if endEv, ok := e.checkEnd(); ok {
      events = append(events, endEv...)                  // ❌ 같은 batch 에서 GameEnded
  }
  ```

### 2.2 클라이언트 영향
- U5 `reducer.ts` 가 `GameEnded` 를 받으면 `state.phase = "END"` 로 즉시 갱신.
- `PlayerView` 는 `EndScreen` 으로 전환, `PublicView` 는 `phase === "END"` 분기로 EndScreen-grade 배경 + 자막 영역 비표시.
- `audioCues` FIFO 가 `eliminated.mafia` / `death.announced` mp3 (~3s) + `end.mafia` mp3 (~3s) 를 연속 enqueue 하지만, 시각적으로는 결과 자막이 거의 보이지 않음.

## 3. 요구사항 (사용자 답변 Q1~Q8 = 모두 A 기준)

### 3.1 Functional Requirements

- **FR-1 (Q1=A)** — VOTE/RECOUNT 처형(`Eliminated`) 으로 게임이 끝나는 경로와 NIGHT→DAY 사망 발표(`DeathAnnounced` 또는 `PeacefulNight`) 로 게임이 끝나는 경로 **둘 다** 에서 `GameEnded` 이벤트 발행을 지연시킨다.
- **FR-2 (Q2=A)** — 결과 자막(처형/사망/평화) 이 화면에 노출된 시점부터 정확히 **5초** 후 `GameEnded` 가 발행되도록 한다. 새 상수 `defaultFinalResultBufferSeconds = 5` (Iter8 `defaultDayIntroSeconds` 와 동일 시각·동일 수치) 를 도입한다.
- **FR-3 (Q3=A)** — `end.mafia` / `end.citizen` 음성 cue 는 `GameEnded` 발행 시점에 발화된다 (현재 announce 카탈로그 동작 그대로). 자막 노출 시간만 늘어난다.
- **FR-4 (Q4=A)** — 결과 버퍼는 호스트 Pause 토글의 영향을 받는다. Iter5 Pause 정책과 동일하게, Pause 동안 카운트다운 정지 → Resume 시 남은 시간만큼 EndScreen 전환이 추가 지연된다.
- **FR-5 (Q5=A)** — `HostEndGame` (HOST_FORCE_END) 경로는 **버퍼 없이 즉시** `GameEnded` 를 발행한다 (현재 동작 보존, 본 fix 가 영향을 주지 않아야 함).
- **FR-6 (Q6=A)** — 신규 Phase 는 도입하지 않는다. State 에 새 필드 `PendingGameEnd` 를 추가하여 (a) 종료 사유와 승자, (b) wall-clock 마감 시점을 보유하고, `engine.Tick` 이 deadline 도달 시 실제 `GameEnded` 이벤트를 발행하면서 `PendingGameEnd` 를 클리어한다.
- **FR-7 (Q7=A)** — 서버에서 `GameEnded` emit 시점만 늦추므로 호스트 PublicView, 모든 PlayerView, 전광판 PUBLIC 이 자동으로 동일 시점에 EndScreen 으로 전환된다 (별도 클라이언트 작업 없음).
- **FR-8 (Q8=A)** — 음성이 꺼져 있거나 mp3 가 graceful skip 되어도 결과 자막은 동일하게 5초 노출된다 (시간은 cue 길이와 무관).

### 3.2 Non-Functional Requirements

- **NFR-1 (성능)** — 추가 Tick 처리 1건 추가, 기존 Tick 루프 비용 증가는 무시 가능 (O(1) 비교 1회).
- **NFR-2 (결정성)** — `engine.Tick` 의 시간 비교는 기존과 동일하게 `e.clock.Now()` 사용, 테스트는 `fakeClock` 으로 deterministic 진행.
- **NFR-3 (직렬화 호환)** — `PendingGameEnd` 가 nil 인 기존 스냅샷을 로드해도 동작 무결성 유지 (omitempty). 게임 진행 중 절대 nil 이 아닌 상태 외부로 노출되지 않음 (서버 내부 전이 상태).
- **NFR-4 (Pause 일관성)** — Iter5 Pause 처리와 동일하게 `Resume` 핸들러가 `PendingGameEnd.Deadline` 을 elapsed 만큼 shift 한다.
- **NFR-5 (커버리지)** — 신규 라인은 100% 커버 (vote 엔드, night 엔드, pause/resume, host force-end 의 4개 경로 + Tick 만료 처리).
- **NFR-6 (회귀 영향 최소화)** — 기존 67개 Go 테스트 + 66 npm 테스트 모두 PASS 유지. `end_test.go::scenario_*` 는 Tick 한 번 추가하는 헬퍼로 마이그레이션.

### 3.3 사용자 시나리오 (Acceptance Criteria)

- **AC-1 (Vote-end)** — 마피아 1, 시민 1, 의사 0, 경찰 0 상황에서 마피아가 처형되어 시민 승리 조건 충족.
  - 기대: SubtitleArea 가 `"○○○이(가) 마피아였습니다."` 자막 + `eliminated.mafia` cue 재생 → **5초 유지** → `EndScreen "CITIZENS WIN"` + `end.citizen` cue.
- **AC-2 (Night-end)** — 마피아 1, 시민 1 상황에서 마피아가 시민을 살해하여 마피아 승리 조건 충족.
  - 기대: PhaseChanged{DAY} → SubtitleArea 가 `"○○○이(가) 사망했습니다."` 자막 + `death.announced` cue → **5초 유지** → `EndScreen "MAFIA WINS"` + `end.mafia` cue.
- **AC-3 (Peaceful-end-impossible)** — `PeacefulNight` 으로는 종료 조건이 발생할 수 없음 (사망 0건이면 카운트 변동 없음). 그러나 만약 사후 변경으로 발생하더라도 동일하게 5초 버퍼 후 EndScreen.
- **AC-4 (Pause)** — 결과 자막 노출 후 1초 시점에 호스트 Pause → 30초 후 Resume → Resume 시점에서 4초 더 지난 후 EndScreen 전환.
- **AC-5 (HOST_FORCE_END)** — 호스트가 `host:end-game` 송신 시 `GameEnded{HOST_FORCE_END}` 가 즉시 emit (버퍼 없음).
- **AC-6 (Snapshot resume)** — `PendingGameEnd` 가 채워진 상태로 서버 프로세스가 재기동 → snapshot 복원 → 다음 Tick 에서 deadline 비교 후 정상 emit.

## 4. 영향 단위 매핑

- **U1 Game Core** — types.go, end.go, tally.go, resolve_night.go, handlers_lifecycle.go (Pause/Resume), tick.go, state_clone.go. 신규 테스트 파일 `iteration9_test.go` (5~7 케이스).
- **U2 Session/Persistence/Announce** — **변동 없음 예상**. `BuildPrivateView` 가 `PendingGameEnd` 를 mask 할지 검토 (기본은 nil 노출 안 함). 카탈로그는 변경 없음 — `eliminated.*` / `death.*` / `peaceful.*` cue 는 그대로.
- **U3 Realtime Transport** — **변동 없음 예상**. `GameEnded` wire 직렬화 그대로, `PendingGameEnd` 는 wire 노출 X (서버 내부 전이만).
- **U4 HTTP Bootstrap** — SKIP.
- **U5 Web Frontend** — **변동 없음 예상**. 서버에서 GameEnded 발행 시점만 미루므로 reducer/뷰는 변경 없음. 회귀 npm 테스트만 추가 검토.

## 5. Extension Compliance

| Extension | Enabled | Iteration 9 영향 |
|---|---|---|
| Security Baseline | No | N/A — 사용자 결정 (Requirements Q14=B) 보존, 변경 없음 |

## 6. 결정 일자 / 승인

- 사용자 답변 (전 항목 A) 수신: 2026-04-30T00:30:00Z
- 본 요구사항 문서 v1.0 작성: 2026-04-30T00:35:00Z
- 다음 게이트: 사용자 승인 → Workflow Planning (`construction/plans/iteration9-execution-plan.md`)
