# Iteration 8 — Workflow Execution Plan v1.0

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration8-fix-vote-result-requirements.md` v1.0 (사용자 승인 2026-04-29T21:25Z)
**Branch**: `worktree-fix+vote-result`
**Type**: Bug Fix (UX · Domain Timing)
**Risk**: Low–Medium · 단일 NightStep enum 추가 + 도메인 상수 + 카탈로그 분기 + wire 유니온 갱신

---

## 1. 추천 실행 시퀀스 개요

```
INCEPTION (완료)
   ├─ Workspace Detection ✓
   ├─ Reverse Engineering — SKIP
   ├─ Requirements Analysis ✓ (사용자 승인 2026-04-29T21:25Z)
   ├─ User Stories — SKIP
   ├─ Workflow Planning ⟵ (현재)
   ├─ Application Design — SKIP (컴포넌트 추가 없음)
   └─ Units Generation — SKIP (5단위 구조 유지)

CONSTRUCTION (per-unit, 실행 순서: U1 → U2 → U3 → U5 → Build & Test)
   ├─ Phase A — U1 Game Core (FD Patch + Code Gen Plan + Code Gen)
   ├─ Phase B — U2 Session/Announce (FD Patch + Code Gen Plan + Code Gen)
   ├─ Phase C — U3 Realtime Transport (검증 only — FD Note + 회귀 테스트 추가)
   ├─ Phase D — U5 Web Frontend (FD Patch + Code Gen Plan + Code Gen)
   └─ Phase E — Build & Test (test-results.md 작성)

OPERATIONS (placeholder)
```

User Stories / Application Design / Units Generation / NFR Requirements / NFR Design / Infrastructure Design 은 본 결함 범위에서 가치를 더하지 않으므로 모두 SKIP.

---

## 2. Phase 별 상세

### Phase A — U1 Game Core

**필요한 산출물**
- `aidlc-docs/construction/u1-game-core/functional-design/iteration8-patch.md` — Minimal patch
  - NightStep enum 변경 표 (Iteration 4 patch 와 동일 포맷)
  - State 필드 변경 없음
  - 도메인 상수 2건 (`defaultNightIntroSeconds`, `defaultDayIntroSeconds`)
  - `nightStepSeconds` 분기, `enterNight` 시작 step, `nextNightStep` 표, `resolveNight` Deadline 식, `handlePauseGame` INTRO 거부
  - 테스트 케이스 표 (I8-T1~T7)
- `aidlc-docs/construction/plans/iteration8-u1-code-generation-plan.md` — Step A~E 체크리스트

**코드 변경 파일** (예상 7건)
1. `internal/game/types.go` — NightStep enum 상수 1건, 도메인 상수 2건, `nightStepSeconds` switch 분기 1건
2. `internal/game/resolve_night.go`:
   - `enterNight()` 의 시작 step `NightStepMafia → NightStepIntro`
   - `nextNightStep()` switch 에 `NightStepIntro → NightStepMafia` 케이스
   - `resolveNight()` 의 Deadline 식: `now + (defaultDayIntroSeconds + DiscussionSeconds) * Second`
3. `internal/game/handlers_lifecycle.go` — `handlePauseGame` 의 사전조건 강화 (또는 `canPause` 가 NightStep 까지 보는 함수로 시그니처 변경)
4. `internal/game/iteration8_test.go` (신규) — I8-T1~T7 (7 테스트)
5. (회귀 보정) `internal/game/handlers_night_test.go`, `iteration5_test.go`, `iteration4_test.go`, `tick_test.go`, `resolve_night_test.go`, `fixtures_test.go` 의 `advanceToNight` 헬퍼 — 진입 시점에 NightStep 이 INTRO 임을 가정하고 적절히 step 진행 후 MAFIA 까지 도달시키거나, 헬퍼 자체가 INTRO 를 자동으로 통과하도록 갱신

**Step A~E 체크리스트** (Code Generation Plan 본문에 그대로 옮길 항목)
- [ ] Step A — `internal/game/types.go` enum/상수/`nightStepSeconds` 변경
- [ ] Step B — `internal/game/resolve_night.go` enter/next/resolve 변경
- [ ] Step C — `internal/game/handlers_lifecycle.go` Pause INTRO 거부
- [ ] Step D — 신규 `iteration8_test.go` 7 테스트 + 기존 헬퍼/테스트 회귀 보정
- [ ] Step E — `go vet ./...` PASS, `go test ./internal/game/... -count=1 -race` PASS, 6 패키지 회귀 PASS, 커버리지 측정 (목표 ≥ 91.0%)
- [ ] audit.md 갱신, aidlc-state.md U1 섹션 갱신

**완료 메시지** (FD/CG 각각 2-옵션 게이트):
- "Continue to Next Stage" → Phase B
- "Request Changes" → 동일 Phase 내 v1.1 보정

---

### Phase B — U2 Session/Announce

**필요한 산출물**
- `aidlc-docs/construction/u2-session-persistence-announce/functional-design/iteration8-patch.md` — Minimal patch
  - 카탈로그 `NightStepChanged{NightStepIntro}` silent 분기
  - 기존 `phase.night` cue 가 안내 담당이라는 점 명시
  - 테스트 케이스 표 (I8-A1)
- `aidlc-docs/construction/plans/iteration8-u2-code-generation-plan.md` — Step A~E 체크리스트

**코드 변경 파일** (예상 2건)
1. `internal/announce/catalog_default.go` — `NightStepChanged` switch 에 `NightStepIntro` 케이스 추가, `return Announcement{}` (간결한 의도 코멘트)
2. `internal/announce/catalog_test.go` — I8-A1 테스트 1건 (NightStepChanged{INTRO} → empty Announcement 검증), `TestRender_NightStepChanged` 의 케이스 표에 INTRO 행 추가

**Step A~E 체크리스트**
- [ ] Step A — `catalog_default.go` 분기 추가
- [ ] Step B — `catalog_test.go` I8-A1 + 매트릭스 갱신
- [ ] Step C — (해당 없음, 시그니처 변경 없음)
- [ ] Step D — `go test ./internal/announce/... -count=1 -race` PASS, 6 패키지 회귀 PASS
- [ ] Step E — audit.md 갱신, aidlc-state.md U2 섹션 갱신

---

### Phase C — U3 Realtime Transport (검증 only)

**필요한 산출물**
- `aidlc-docs/construction/u3-realtime-transport/functional-design/iteration8-patch.md` — Notes only
  - NightStep wire 직렬화는 string passthrough 이므로 enum 추가만으로 자동 호환 됨을 명시
  - protocol 상수에 NightStep 별도 등록이 없음 확인
- 별도 Code Generation Plan 작성하지 않음 — Phase C 가 그 자체로 검증 단계

**코드 변경 파일** (0~1건)
1. (선택) `internal/transport/ws/protocol_test.go` — NightStepChanged{INTRO} payload 직렬화 회귀 1건 (서버 측 통과 확인)

**Step A~E 체크리스트**
- [ ] Step A — `protocol.go` 의 NightStep 관련 상수 부재 확인 (영향 없음 명시)
- [ ] Step B — (선택) 회귀 테스트 1건 추가
- [ ] Step C — `go test ./internal/transport/ws/... -count=1 -race` PASS
- [ ] Step D — audit.md 갱신, aidlc-state.md U3 섹션 갱신

---

### Phase D — U5 Web Frontend

**필요한 산출물**
- `aidlc-docs/construction/u5-web-frontend/functional-design/iteration8-patch.md` — Minimal patch
  - `wire.ts` NightStep 유니온에 `"INTRO"` 추가
  - reducer 변경 없음 (string passthrough 자동 호환)
  - PlayerView Picker 들 — 현재 step 비교 로직이 정확히 `MAFIA`/`POLICE`/`DOCTOR` 등 명시 비교이므로 INTRO 단계에서 자동으로 모든 picker 비활성화 되어 호환됨
  - PublicView TimerBar — `nightStepDeadline` 으로 카운트다운만 표시
  - 테스트 케이스 표 (I8-W1, I8-W2)

**코드 변경 파일** (예상 2~3건)
1. `web/src/types/wire.ts` — `NightStep` 유니온에 `"INTRO"` 추가
2. `web/src/context/reducer.test.ts` — I8-W1 (NightStepChanged{INTRO} 보존 검증)
3. (필요 시) `web/src/views/PlayerView/PhaseInputs.test.tsx` 또는 NightInputs 회귀 — I8-W2 (INTRO 단계에서 모든 picker 비활성)

**Step A~E 체크리스트**
- [ ] Step A — `wire.ts` 유니온 갱신
- [ ] Step B — 신규 reducer 테스트 1건
- [ ] Step C — (필요 시) Picker 회귀 테스트 1건
- [ ] Step D — `npm run typecheck` PASS, `npm test` PASS (50+ 케이스), `npm run build` 성공
- [ ] Step E — audit.md 갱신, aidlc-state.md U5 섹션 갱신

---

### Phase E — Build & Test

**필요한 산출물**
- `aidlc-docs/construction/build-and-test/iteration8-test-results.md`
  - FR-1~FR-8 추적 매트릭스
  - 패키지별 커버리지 표 (announce / game / persistence / session / transport/http / transport/ws)
  - JS gzip 빌드 사이즈 표
  - 회귀 영향 분석 (특히 Iteration 5 의 NightStep 흐름 / Iteration 7 의 카탈로그 무결)
  - NFR 영향 정리
  - DoD 체크리스트
  - 후속 권장 사항 (예: Chrome DevTools MCP 골든패스 회귀)

**검증**
- [ ] `go test ./... -count=1` 6 패키지 PASS
- [ ] `go test ./... -race -count=1` PASS
- [ ] `go build -o /tmp/mafia-game-iter8 ./cmd/mafia-game` 성공
- [ ] `npm test` PASS, `npm run typecheck` PASS, `npm run build` 성공
- [ ] 커버리지 표 작성, 모든 단위 baseline 미만 없음 검증

**완료 메시지**: "**Build and test instructions complete. Ready to proceed to Operations stage?**" → 사용자 승인 후 Iteration 8 종료.

---

## 3. SKIP 단계 사유

| 단계 | 사유 |
|---|---|
| Reverse Engineering | 기존 Iteration 1~7 산출물 활용, 5단위 구조 변동 없음 |
| User Stories | 단일 결함 패치 — 페르소나 추가 없음, 기존 호스트/플레이어 인터랙션 재사용 |
| Application Design | 컴포넌트 추가/제거 없음 — NightStep enum 1건 + 카탈로그 분기 1건 + 도메인 상수 2건 + wire 유니온 1건 |
| Units Generation | 5단위 구조 유지 (U1/U2/U3/U4/U5) |
| NFR Requirements / NFR Design | 성능/보안/확장성 변경 없음 — 5초 인터벌은 사용자 정책. 커버리지/빌드 사이즈는 기존 NFR 게이트로 충분 |
| Infrastructure Design | 단일 바이너리, 인프라 변경 없음 |
| U4 HTTP Bootstrap | HTTP 라우팅/정적 자산/audio 핸들러 변경 없음 |

---

## 4. RISK 정리

| RISK | 완화책 |
|---|---|
| 기존 테스트 다수가 `advanceToNight` 헬퍼 + NightStep=MAFIA 시점을 가정 | 헬퍼 1곳을 INTRO→MAFIA 자동 진행하도록 교체. 회귀 테스트 6 패키지 PASS 로 검증 |
| 영속(SQLite) 스냅샷 호환 | I8-T7 회귀 테스트 1건 추가 — legacy NightStep=MAFIA 스냅샷이 정상 진행되는지 확인 |
| 첫째날 (transitionIntroToDay) 에 버퍼가 들어가는 회귀 | 명시적으로 `transitionIntroToDay` 코드를 변경하지 않음. 검증 테스트 I8-T5 추가 |
| Pause 거부 처리 누락 시 INTRO 중 Pause 가 deadline shift 에 잠재 결함 도입 | I8-T6 회귀로 명시적 거부 검증 |
| Iteration 7 Voice cue 중 `phase.night` 가 INTRO 단계 안내를 담당하는지 호스트 환경에서 실측 확인 | OPERATIONS 단계 권장 사항으로 Chrome DevTools MCP 골든패스 회귀 명시 |

---

## 5. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-29 | 최초 작성, RA v1.0 사용자 승인 후 |
