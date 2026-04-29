# U1 Game Core — Code Generation Summary

**작성일**: 2026-04-26
**상태**: Code Generation 완료
**워크스페이스 루트**: `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game`
**U1 코드 위치**: `internal/game/`

---

## 1. 생성된 파일 목록

### 워크스페이스 메타 (2)

| 파일 | 라인 | 역할 |
|---|---:|---|
| `go.mod` | 3 | Go 1.22 단일 모듈 — `module github.com/saltware/mafia-game`, **외부 의존 0** |
| `.gitignore` | 17 | 빌드 산출물·런타임 데이터·프론트엔드 빌드물·OS/IDE 메타 제외 |

### 도메인 코드 (`internal/game/`, 19 Go 파일)

| 파일 | 라인 | 역할 |
|---|---:|---|
| `doc.go` | 21 | 패키지 godoc 개요 |
| `types.go` | 237 | PlayerID/Role/Team/Phase/EndReason/Player/Options/State/PendingActions + State 헬퍼(LiveCount, FindPlayer 등) + JSON 태그 |
| `state_clone.go` | 50 | `State.Clone()` — 모든 슬라이스/맵/포인터 비공유 사본 (P2) |
| `action.go` | 96 | sealed `Action` interface + 10 액션 타입 (sealedAction 임베드) |
| `event.go` | 159 | sealed `Event` interface + 15 이벤트 타입 + Visibility/EventEnvelope + pub/priv/mafia 헬퍼 |
| `error.go` | 75 | ErrorCode 9종 + EngineError(`errors.Is/As` 호환) + sentinel errors 9개 |
| `validation.go` | 131 | ValidationErrors/FieldError + validateOptions(BR-OPT) + ensureHost/Phase/Role/Alive 헬퍼 (P5) |
| `clock.go` | 27 | Clock 인터페이스 + realClock + FakeClock |
| `rand.go` | 28 | extractSeed64, newInnerRand (crypto/rand → seed → math/rand) (P6) |
| `keyword.go` | 62 | KeywordPool 인터페이스 + mapKeywordPool 구현 + NewDefaultKeywordPool |
| `keyword_pool_data.go` | 36 | 한국어 기본 풀 140개 (Mafia 40 / Citizen 40 / Doctor 30 / Police 30) |
| `keyword_loader.go` | 47 | LoadKeywordPool(JSON) — FR-7.1 외부화 인터페이스 |
| `role.go` | 108 | Assignments + RoleAssigner + defaultAssigner (셔플·역할 부여·키워드 동일·대표자 무작위) |
| `engine.go` | 166 | Engine 인터페이스 + engine struct + New/NewDefault 생성자 + Snapshot/Restore + Start |
| `apply.go` | 49 | Apply 타입 스위치 dispatch (P1) + allNightActionsSubmitted 헬퍼 |
| `handlers_lifecycle.go` | 125 | handleStartGame, handleAdvanceIntro, handleToggleVoice, handleForceEnd, transitionIntroToNight |
| `handlers_night.go` | 126 | handleMafiaKill, handleDoctorHeal, handlePoliceCheck, handleEndNightEarly |
| `handlers_day_vote.go` | 69 | handleEndDiscussionEarly, handleVote, transitionDayToVote |
| `resolve_night.go` | 88 | resolveNight (살해/보호/대표자 재지정/Day++/PhaseChanged) + reassignMafiaRepresentative |
| `tally.go` | 116 | tally (VoteRound 1·2 + RECOUNT 동률 후보 한정) + applyElimination + transitionVoteToNight |
| `tick.go` | 94 | Tick 멱등 알고리즘 (P9) + tickIntro + tickDay (DiscussionTimerTick 임계 30/10/0) |
| `end.go` | 38 | checkEnd + endGame (FR-5.1/5.2 + HOST_FORCE_END) |

### 단위 테스트 (`internal/game/`, 16 `_test.go` 파일)

| 파일 | 라인 | 커버 영역 |
|---|---:|---|
| `fixtures_test.go` | 99 | 빌더 — newTestEngine, mustStart, deterministicRNG, allRoles, advanceToNight |
| `types_test.go` | 161 | State.Clone 비공유, JSON 결정성·라운드트립·32 KB 한도, Team/LiveCount 등 |
| `state_clone` 관련 | (in types_test) | NFR-U1-S1~S3 |
| `error_test.go` | 48 | EngineError errors.Is/As, ValidationErrors 누적, nil-safe |
| `validation_test.go` | 124 | validateOptions 8 케이스 + ensure 헬퍼 4 케이스 |
| `role_test.go` | 141 | 인원별 분배표(6~12), 같은 역할=같은 키워드, 대표자=마피아, 결정성 |
| `keyword_test.go` | 100 | 풀 크기·중복, 역할별 추출, JSON 로더 라운드트립·empty 거부 |
| `apply_test.go` | 121 | END 후 액션 거부, unknown action, 에러 시 state 불변, Start 검증 + 이벤트 스트림 + Snapshot/Restore |
| `handlers_lifecycle_test.go` | 80 | AdvanceIntro 호스트만/진행/NIGHT 전이, ForceEndGame, ToggleVoice |
| `handlers_lifecycle_apply_test.go` | 69 | Apply(StartGame) — LOBBY → INTRO 정상/거부 |
| `handlers_night_test.go` | 204 | Mafia 대표자 한정/마피아 못 죽임/의사 자가 보호/경찰 1회/자기 조사 금지/auto-resolve/protect |
| `handlers_day_vote_test.go` | 165 | EndDiscussionEarly 호스트만, 모두 투표 시 tally, 동률 RECOUNT, 후보 외 거부 |
| `handlers_errors_test.go` | 166 | 13개 에러 분기 — 단계 위반/역할 불일치/사망자/호스트 권한 |
| `resolve_night_test.go` | 56 | PeacefulNight, Day++, deadline 설정 |
| `tally_test.go` | 88 | 단일 최다, 동률 RECOUNT, 재투표 동률 → 무처형 |
| `tick_test.go` | 108 | 멱등성, INTRO 자동 진행, NIGHT 전이, DAY deadline 만료 → VOTE, DiscussionTimerTick(30) |
| `end_test.go` | 71 | 시민 승, 마피아 승 |
| `scenario_test.go` | 107 | 시나리오 1 (게임 시작 → NIGHT), 시나리오 4 (재투표 동률 무처형), 시나리오 3 (재시작 복원) |
| `property_test.go` | 68 | testing/quick — Tick 멱등, Snapshot/Restore 라운드트립, Clone 독립성 |
| `markers_test.go` | 108 | sealed interface 구현 검증, Pending/LivingMafiaIDs/NewDefault/realClock/engine.String |
| `reassign_test.go` | 85 | 대표자 처형 시 재지정 + MafiaRepresentativeReassigned 이벤트 |

### 문서 산출물 (`aidlc-docs/construction/u1-game-core/code/`, 2 파일)

- `u1-code-summary.md` (본 문서)
- `u1-public-api.md` — U1이 다른 단위(U2~U5)에 노출하는 공개 API 카탈로그

---

## 2. 검증 결과 (NFR-U1 게이트 8종)

| 게이트 | 결과 | 메모 |
|---|---|---|
| `go vet ./...` | ✅ 0 issue | (pre Build & Test) |
| `gofmt -l ./internal/game/` | ✅ 0 lines | |
| `go build ./...` | ✅ 통과 | |
| `go test ./internal/game/...` | ✅ 모든 테스트 통과 | |
| `go test -race ./internal/game/...` | ✅ 통과 | NFR-U1-C2 |
| `go test -cover ./internal/game/...` | ✅ **90.4%** | NFR-U1-M1 (≥ 90%) 충족 |
| `go list -deps` 외부 의존성 | ✅ **0개** | NFR-U1-M9 (요구사항: `go.mod`의 require 빈 상태) |
| State JSON < 32 KB (12명) | ✅ 단위 테스트 통과 | NFR-U1-S2 |

> `golangci-lint run` 및 분기 커버리지 측정은 Build & Test 단계에서 수행 예정.

---

## 3. NFR 패턴 적용 현황

| 패턴 | 적용 위치 |
|---|---|
| P1 타입 스위치 dispatch | `apply.go` |
| P2 수동 Clone 깊은 복사 | `state_clone.go` |
| P3 생성자 주입 | `engine.go::New`, `clock.go`, `rand.go` |
| P4 임베드 풀 + 외부화 인터페이스 | `keyword_pool_data.go`, `keyword_loader.go` |
| P5 누적 에러 검증 | `validation.go::ValidationErrors` |
| P6 시드 가능 inner PRNG | `rand.go::newInnerRand` |
| P7 단일 스레드 가정 | `engine.go` godoc 명시, mutex 미도입 |
| P8 타입드 에러 + sentinel | `error.go` |
| P9 Tick 멱등화 | `tick.go::Tick` (`LastTickAt` 가드) |
| P10 결정적 직렬화 | `types.go` JSON 태그 + `types_test.go::TestState_JSONDeterminism` |
| P11 테스트 분포 | 테이블/시나리오/속성 — `*_test.go` 16종 |

---

## 4. Build & Test 단계 인풋 가이드

다음 명령으로 U1을 검증할 수 있습니다 (Build & Test 단계 스크립트의 입력):

```bash
# 1) 빌드
go build ./internal/game/...

# 2) 정적 분석
go vet ./internal/game/...
gofmt -l ./internal/game/
golangci-lint run ./internal/game/...

# 3) 단위 테스트 + race + coverage
go test -race ./internal/game/...
go test -coverprofile=coverage.out ./internal/game/...
go tool cover -func=coverage.out | tail -1   # total ≥ 90%

# 4) 외부 의존성 감사
go list -m all   # 표준 lib + module 자체만 출력

# 5) 벤치마크 (Build & Test 단계에서 추가 작성)
# go test -bench=. -benchmem ./internal/game/...
```

---

## 5. 다음 단위 (U2)에서 사용할 의존성 시그니처

U2 SessionManager는 다음 U1 식별자를 import:
- `Engine` 인터페이스 + `New`, `NewDefault` 생성자
- `Action` 타입 10종, `Event` 타입 15종, `EventEnvelope`, `Visibility`
- 도메인 타입: `PlayerID`, `Role`, `Team`, `Phase`, `Player`, `State`, `Options`, `EndReason`
- 에러: `EngineError`, `ErrorCode 상수 9종`, sentinel `ErrValidation` 등
- 인프라 인터페이스: `Clock`, `RoleAssigner`, `KeywordPool`
- 헬퍼: `NewDefaultKeywordPool`, `LoadKeywordPool`, `DefaultOptions`
