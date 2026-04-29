# Iteration 9 — Build & Test 통합 결과 v1.0

| 항목 | 내용 |
|---|---|
| 작성 | 2026-04-30T01:40:00Z |
| 상태 | 사용자 최종 승인 대기 |
| 작업 브랜치 | `worktree-fix+final-result` |
| 변경 범위 | U1 (Game Core) 단일 단위. U2/U3/U4/U5 코드 변경 없음. |
| 결과 한 줄 요약 | VOTE/NIGHT 결판 후 5초 결과 자막 노출 → EndScreen 전환. 6 패키지 race PASS / npm 66 PASS / game 커버리지 +0.7pp. |

---

## 1. 요구사항 → 구현 추적 매트릭스

| ID | 요구사항 (요약) | 구현 위치 | 검증 (테스트 / 명령) |
|---|---|---|---|
| FR-1 | VOTE 처형 + NIGHT 사망 발표 양 경로 모두 5s 버퍼 | `tally.applyElimination` + `resolve_night.resolveNight` 가 각각 `evaluateEnd → scheduleGameEnd` 호출 | I9-T1 (vote-end), I9-T3 (night-end) |
| FR-2 | 결과 자막 노출 후 5초 후 GameEnded | `defaultFinalResultBufferSeconds = 5` (`types.go`), `scheduleGameEnd` 가 deadline = now + 5s | I9-T1 deadline 검증, I9-T2/T4 Tick 발화 |
| FR-3 | end.* cue 는 GameEnded emit 시점에 발화 (자막 노출만 늘어남) | announce 카탈로그 무변경, U5 reducer 무변경 | 카탈로그 회귀 (announce 94.3% 유지) |
| FR-4 | Pause 영향 받음 — Pause/Resume 시 deadline shift | `handleResumeGame` 의 PendingGameEnd.Deadline shift, `handlePauseGame` 의 PendingGameEnd nil 분기 | I9-T5 |
| FR-5 | HOST_FORCE_END 즉시 emit (버퍼 없음) | `handleForceEnd` 가 PendingGameEnd 클리어 + 즉시 endGame | I9-T6, 기존 `TestForceEndGame_TerminalState` |
| FR-6 | 신규 Phase 미도입 — `State.PendingGameEnd` + Tick 만료 처리 | `types.go::PendingGameEnd` struct + `tick.go` 진입 분기 + `firePendingEnd` | I9-T1~T7 모두 Phase 전이 검증 포함 |
| FR-7 | 서버 단일 emit 시점으로 모든 화면 자동 일치 | wire/U5 무변경 (GameEnded 도착 시점만 늦어짐) | npm 66 PASS, gzip 65.62 KB 무변동 |
| FR-8 | 음성 무관 5초 고정 (cue 길이/누락과 독립) | 버퍼는 wall-clock 기반, audioCues 무관 | I9-T2/T4 (음성 mock 없이 Tick 만으로 검증) |

| ID | NFR | 결과 |
|---|---|---|
| NFR-1 | 성능 (Tick 비용) | Tick 진입부 비교 1회 추가, O(1). 영향 측정 가능 수준 아님. |
| NFR-2 | 결정성 | FakeClock + I9-T1~T7 deterministic PASS, race detector PASS. |
| NFR-3 | 직렬화 호환 | I9-T7 JSON round-trip 검증 (PendingGameEnd 보존, deadline drift 없음). 기존 nil 스냅샷 무손상 (omitempty). |
| NFR-4 | Pause 일관성 | I9-T5 deadline shift 정확도 30s±0 검증. |
| NFR-5 | 신규 라인 커버리지 | `evaluateEnd`/`scheduleGameEnd`/`firePendingEnd` 모두 테스트로 직접 도달. game 커버리지 91.8% → 92.5%. |
| NFR-6 | 회귀 영향 최소화 | 기존 67 Go 테스트 + 66 npm 테스트 모두 PASS, 마이그레이션 0건. |

| ID | 사용자 시나리오 (AC) | 검증 |
|---|---|---|
| AC-1 | Vote-end 시민 승리 — 5s 자막 후 EndScreen | I9-T1 + I9-T2 |
| AC-2 | Night-end 마피아 승리 — 5s 자막 후 EndScreen | I9-T3 + I9-T4 (6p / 2 mafia 시나리오) |
| AC-3 | PeacefulNight 으로는 종료 발생 불가 (사망 0이면 카운트 변동 없음) | 코드 경로 분석 (resolveNight 의 `if reason, winner, ok := evaluateEnd(); ok` 는 PeacefulNight 분기에선 false) |
| AC-4 | Pause/Resume 시 deadline shift | I9-T5 |
| AC-5 | HOST_FORCE_END 즉시 (버퍼 없음) | I9-T6 |
| AC-6 | Snapshot resume mid-buffer | I9-T7 |

---

## 2. 패키지별 테스트 / 커버리지

| 패키지 | 결과 | 커버리지 | Iter8 baseline | Δ |
|---|---|---|---|---|
| `internal/announce` | PASS (1.37s) | 94.3% | 94.3% | 0 |
| `internal/game` | PASS (1.47s) | **92.5%** | 91.8% | **+0.7pp** |
| `internal/persistence` | PASS (1.94s) | 80.2% | 80.2% | 0 |
| `internal/session` | PASS (2.67s) | 87.3% | 87.3% | 0 |
| `internal/transport/http` | PASS (2.14s) | 90.3% | 90.3% | 0 |
| `internal/transport/ws` | PASS (4.04s) | 82.3% | 82.3% | 0 |

명령어:
```bash
go test ./... -count=1 -race -cover
```

---

## 3. Frontend 회귀

| 검증 | 결과 |
|---|---|
| `npm test` | **66/66 PASS** (Iter8 동일) |
| `npm run typecheck` | (build 단계 `tsc --noEmit` 통과) |
| `npm run build` | 성공 (vite v5.4.21, 68 modules) |
| JS gzip | **65.62 KB** (Iter8 baseline 동일) |
| CSS gzip | 3.21 KB (Iter8 baseline 동일) |
| dist/assets/index.js | 206.35 KB (raw) |

U5 코드 무변경이므로 동일 baseline 유지. wire 의 `state.pendingGameEnd` 새 필드는 TS 타입에 추가되지 않았으나, JSON unmarshaling 이 unknown 필드를 무시하므로 호환 OK.

---

## 4. 빌드 산출물

| 산출물 | 크기 | 비고 |
|---|---|---|
| `/tmp/mafia-game-iter9` | 17 MB | `go build ./cmd/mafia-game` 성공, mp3 자산 임베드 포함 |
| `cmd/mafia-game/web/dist/assets/index-8vnvgdor.js` | 206.35 KB (raw) / 65.62 KB (gzip) | 동일 |
| `cmd/mafia-game/web/dist/assets/index-DlA7cNKj.css` | 11.47 KB (raw) / 3.21 KB (gzip) | 동일 |
| `cmd/mafia-game/web/dist/assets/background.jpg` | 194 KB | 동일 |

---

## 5. 회귀 영향 분석

### 5.1 즉시 종료 경로 (변경 없음)
- `handleForceEnd` (HOST_FORCE_END) — `TestForceEndGame_TerminalState` PASS, ForceEnd 시 PendingGameEnd 클리어 신규 검증은 I9-T6 에 추가.
- `checkEnd()` 직접 호출 (`TestCheckEnd_CitizenWinsWhenAllMafiaDead`, `TestCheckEnd_MafiaWinsWhenEqual`) — 본 함수의 시그니처와 동작(즉시 emit) 보존, 테스트 PASS.

### 5.2 잠재 영향 후보 — 결과적으로 영향 없음
- `tally_test.go` — 테스트 시나리오가 단일 max 처형 후 다음 Phase=NIGHT 전이만 검증. 게임 종료 조건까지 도달하지 않아 PASS.
- `scenario_test.go` — TieRecount/HostRestart 등 종료 미도달 시나리오. PASS.
- `handlers_day_vote_test.go` — 처형 후 NIGHT 전이 검증, 종료 미도달. PASS.
- `iteration4/5/8_test.go` — 각각 NightStep, Pause, INTRO 버퍼에 집중. PASS.

### 5.3 Pause/INTRO 가드 변경 (Iter8 정책 보존)
- Iter8 의 `PauseGame` INTRO 거부는 `state.PendingGameEnd == nil` 인 경우에만 적용되도록 변경. PendingGameEnd 가 있을 수 있는 NIGHT-INTRO 상황은 시나리오상 발생하지 않음 (PendingGameEnd 는 vote-end 후 VOTE/RECOUNT 또는 night-end 후 DAY 에서만 set). 기존 `TestI8_PauseRejectedDuringNightIntro` 등 Iter8 회귀 테스트 PASS.

### 5.4 Pause/Resume 시퀀스 (Iter5 정책 확장)
- `handleResumeGame` 의 phase-deadline shift 로직은 변경 없음, PendingGameEnd shift 가 추가됨. Iter5 의 `TestI5_PauseShiftsNightDeadline`, `TestI5_PauseShiftsDayDeadline`, `TestI5_PauseShiftsIntroSpeaker` 모두 PASS.

---

## 6. 사용자 체감 흐름표

| 시점 | 호스트 화면 | 음성 (호스트 only) |
|---|---|---|
| T+0.0s (vote 종료 — 시민 승리 케이스) | SubtitleArea: "○○○이(가) 마피아였습니다." 노출 | `eliminated.mafia` cue 큐 enqueue |
| T+0.5s | 자막 그대로 | `eliminated.mafia` 재생 시작 |
| T+3.0s | 자막 그대로 (PlayersGrid: 처형된 마피아 표시) | cue 종료, 무음 |
| T+5.0s | EndScreen "CITIZENS WIN" 페이드인 | `end.citizen` cue enqueue → 재생 |
| T+0.0s (night 종료 — 마피아 승리 케이스) | PhaseChanged{DAY} → SubtitleArea: "○○○이(가) 사망했습니다." | `phase.day` + `death.announced` cue 순차 enqueue |
| T+5.0s | EndScreen "MAFIA WINS" | `end.mafia` cue 재생 |

Iter8 의 INTRO 5s + DayIntro 5s 패턴과 동일한 사용자 체감 일관성을 유지.

---

## 7. NFR 영향

| 영역 | 영향 |
|---|---|
| 메모리 | 방당 ~32 byte (PendingGameEnd struct + Team ptr + time.Time). 다중 동시 게임 미지원이라 무시 가능. |
| CPU | Tick 진입부 비교 1회. profile 측정 없음 — 영향 없음으로 판단. |
| 네트워크 | wire 변경 없음. `state.pendingGameEnd` 필드가 추가되긴 하나 결판 직후 5초간만 채워지고 클라이언트가 무시. |
| 운영 | binary 크기 17 MB 동일, dist 동일. 배포 영향 없음. |

---

## 8. 미해결 항목 / 후속 권장

- **Chrome DevTools MCP 다중 컨텍스트 회귀 (호스트 + 4 player)** — 사용자 트리거 권장. 4 시나리오:
  1. Vote-end CITIZEN_WIN: 자막 5초 노출 후 EndScreen
  2. Night-end MAFIA_WIN: 자막 5초 노출 후 EndScreen
  3. Pause/Resume mid-buffer: deadline shift 시각 검증
  4. HOST_FORCE_END mid-buffer: 즉시 EndScreen
- **변경 미커밋** — 본 iteration 의 모든 변경은 worktree `worktree-fix+final-result` 에 저장되어 있으나 commit/PR 미수행. 사용자 명시적 commit 지시 후 처리 예정.

---

## 9. DoD (Definition of Done) 체크리스트

- [x] 사용자 답변 (Q1~Q8 모두 A) 보존
- [x] FR-1~FR-8 모두 코드 + 테스트로 검증
- [x] NFR-1~NFR-6 충족
- [x] AC-1~AC-6 검증 (AC-3 은 코드 경로 분석)
- [x] go test 6 패키지 race PASS
- [x] game 커버리지 ≥ 91.8% (실측 92.5%)
- [x] go build 성공
- [x] npm test 66/66 PASS
- [x] npm run build 성공, gzip baseline 동일
- [x] aidlc-state.md / audit.md 동기화
- [x] U1 Code Generation Plan 모든 step [x]
- [x] **사용자 최종 승인** (2026-04-30T01:50:00Z)

---

## 10. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-30 | 최초 작성 |
