# Code Generation Plan — U1 Game Core

**작성일**: 2026-04-26
**대상 단위**: U1 Game Core (`internal/game/*`)
**참조**:
- `application-design/unit-of-work.md` §1
- `application-design/unit-of-work-story-map.md` §4 U1 Primary
- `construction/u1-game-core/functional-design/*.md`
- `construction/u1-game-core/nfr-requirements/*.md`
- `construction/u1-game-core/nfr-design/*.md`
- `aidlc-state.md` (Workspace Root: `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game`)

> **본 plan은 U1 Code Generation의 단일 진실 소스(Single Source of Truth)** 입니다. Generation 단계는 본 체크리스트를 순서대로 실행합니다.

---

## 0. 단위 컨텍스트

### 0.1 단위 책임
U1 Game Core는 마피아 게임 한 판의 **상태 머신과 비즈니스 규칙**을 담당하는 도메인 단위입니다. 외부 I/O 0, 표준 라이브러리만 사용.

### 0.2 구현 대상 요구사항 (Primary, story map §4)
- FR-1.3 (인원 6~12 검증)
- FR-2.1 (역할 배분), FR-2.2 (무작위 키워드)
- FR-3.1 (키워드 풀), FR-3.3 (자기소개 시간 자동 진행)
- FR-4.1 (단계 전이), FR-4.2 (밤 행동), FR-4.3 (마피아 대표자), FR-4.4 (의사 자가 보호), FR-4.5 (토론 + 호스트 조기 종료), FR-4.6 (동률 처리)
- FR-5.1 (시민 승), FR-5.2 (마피아 승)
- FR-7.1 (외부화 가능 키워드 풀)
- NFR-1 (안정성·복원), NFR-2 (성능), NFR-6 (도메인 분리)

### 0.3 단위 의존성
- **외부 의존**: 없음 (Go 표준 라이브러리만)
- **다른 단위 의존**: 없음 (U1은 도메인 핵심, 가장 안쪽)
- **하위 단위가 의존**: U2 SessionManager가 U1의 도메인 타입을 import

### 0.4 인터페이스 / 계약
U1은 다음 공개 식별자를 다른 단위에 노출:
- 인터페이스: `Engine`, `RoleAssigner`, `KeywordPool`, `Clock`
- 도메인 타입: `PlayerID`, `Role`, `Team`, `Phase`, `EndReason`, `Player`, `State`, `Options`, `PendingActions`, `Assignments`
- Sum types: `Action` (10종), `Event` (14종), `EventEnvelope`, `Visibility`
- 에러: `EngineError`, `ErrorCode` (9종 상수), sentinel errors, `ValidationErrors`, `FieldError`
- 생성자: `New(assigner, clock, rng)`, `NewDefault(pool)`, `NewAssigner(pool)`, `NewDefaultKeywordPool()`, `LoadKeywordPool(reader)`, `realClock`

### 0.5 데이터 엔티티 소유
U1은 **DB 엔티티를 소유하지 않음**. 영속화는 U2가 책임. U1은 `State`만 정의하여 직렬화 가능하도록 설계.

---

## 1. 코드 위치 결정 (Step 2)

| 항목 | 위치 |
|---|---|
| Workspace Root | `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game` |
| 프로젝트 유형 | Greenfield, multi-unit (monolithic) |
| 적용 패턴 | Greenfield Go 모듈 — 모든 코드는 워크스페이스 루트 (NEVER `aidlc-docs/`) |
| U1 코드 위치 | `internal/game/` (워크스페이스 루트 기준) |
| 빌드 산출물 | Go 패키지 (다른 단위가 import) — 단독 바이너리 없음 |
| 문서 산출물 (요약) | `aidlc-docs/construction/u1-game-core/code/` (markdown 요약만) |

---

## 2. 작업 체크리스트 (Part 1 — Planning)

- [x] (P1-1) 단위 컨텍스트 분석 완료
- [x] (P1-2) 코드 위치·구조 결정
- [x] (P1-3) 본 plan 문서 작성
- [x] (P1-4) 사용자에게 요약 제공
- [x] (P1-5) audit에 승인 게이트 로그
- [x] (P1-6) 사용자 승인 획득 — 2026-04-26
- [x] (P1-7) Part 2 진입

---

## 3. 작업 체크리스트 (Part 2 — Generation)

각 단계는 본 plan에 명시된 파일을 워크스페이스 루트에 생성·수정합니다. 단계 완료 즉시 [x] 처리.

### 3.1 Project Structure Setup (Greenfield)

- [x] (G1) `go.mod` 생성 — `module github.com/saltware/mafia-game`, Go 1.22
- [x] (G2) `.gitignore` 생성 — Go 표준 + `data/*.db`, `web/dist/`, `web/node_modules/`
- [x] (G3) `internal/game/` 디렉터리 생성

### 3.2 Domain Types & Errors

- [x] (G4) `internal/game/doc.go` — 패키지 godoc + 주요 사용 예
- [x] (G5) `internal/game/types.go` — `PlayerID`, `Role`, `Team`, `Phase`, `EndReason`, `Player`, `Options`, `State`, `PendingActions` (JSON 태그 포함)
- [x] (G6) `internal/game/state_clone.go` — `State.Clone()` 깊은 복사 (P2)
- [x] (G7) `internal/game/action.go` — `Action` 인터페이스 + 10 액션 타입 (sealed via private method)
- [x] (G8) `internal/game/event.go` — `Event` 인터페이스 + 14 이벤트 타입 + `Visibility` enum + `EventEnvelope`
- [x] (G9) `internal/game/error.go` — `ErrorCode` 9종 상수 + `EngineError` (Is/As 호환) + sentinel errors 9개
- [x] (G10) `internal/game/validation.go` — `ValidationErrors`, `FieldError`, `validateOptions`, `ensureHost/Phase/Role/Alive` 헬퍼 (P5)

### 3.3 Infrastructure Interfaces (Clock, RNG, KeywordPool, RoleAssigner)

- [x] (G11) `internal/game/clock.go` — `Clock` 인터페이스 + `realClock`
- [x] (G12) `internal/game/rand.go` — `extractSeed64`, `newInnerRand` 헬퍼 (P6)
- [x] (G13) `internal/game/keyword.go` — `KeywordPool` 인터페이스 + `mapKeywordPool` 구현 + `NewDefaultKeywordPool()`
- [x] (G14) `internal/game/keyword_pool_data.go` — 한국어 기본 풀 140개 슬라이스 (Mafia 40 / Citizen 40 / Doctor 30 / Police 30) (P4)
- [x] (G15) `internal/game/keyword_loader.go` — `LoadKeywordPool(io.Reader) (KeywordPool, error)` (FR-7.1)
- [x] (G16) `internal/game/role.go` — `Assignments` + `RoleAssigner` 인터페이스 + `defaultAssigner` 구현 (셔플·역할 부여·키워드 동일 부여·대표자 무작위) (LC-2)

### 3.4 Engine + Apply Dispatch + Handlers

- [x] (G17) `internal/game/engine.go` — `Engine` 인터페이스 + `engine` struct + `New`, `NewDefault` 생성자 (P3, P7) + `Snapshot`, `Restore` 메서드
- [x] (G18) `internal/game/apply.go` — `Apply(action) (State, []Event, error)` 타입 스위치 dispatch (P1)
- [x] (G19) `internal/game/handlers_lifecycle.go` — `handleStartGame`, `handleAdvanceIntro`, `handleForceEnd`, `handleToggleVoice`
- [x] (G20) `internal/game/handlers_night.go` — `handleMafiaKill`, `handleDoctorHeal`, `handlePoliceCheck`, `handleEndNightEarly`
- [x] (G21) `internal/game/handlers_day_vote.go` — `handleEndDiscussionEarly`, `handleVote`
- [x] (G22) `internal/game/resolve_night.go` — `resolveNight()` (살해/보호/대표자 재지정/Day++/PhaseChanged) (BR-RESOLVE)
- [x] (G23) `internal/game/tally.go` — `tally()` (VoteRound 1·2 + RECOUNT 동률 후보 한정) (BR-VOTE)
- [x] (G24) `internal/game/tick.go` — `Tick(now)` 멱등 알고리즘 (P9), INTRO/DAY 시간 진전, DiscussionTimerTick 임계 (30/10/0)
- [x] (G25) `internal/game/end.go` — `checkEnd()`, `transitionAfterElimination`, `transitionAfterNoElimination`

### 3.5 Business Logic Unit Testing

- [x] (G26) `internal/game/fixtures_test.go` — `newTestEngine`, `mustStartGame`, `playFirstNight`, `deterministicRNG` 빌더 (LC-11)
- [x] (G27) `internal/game/types_test.go` — `State.Clone` 비공유 라운드트립 + JSON Marshal 결정성 + 32 KB 한도 (NFR-U1-S1~S3)
- [x] (G28) `internal/game/error_test.go` — `errors.Is/As` 동작, `ValidationErrors` 누적
- [x] (G29) `internal/game/validation_test.go` — `validateOptions` 모든 BR-OPT-* 케이스 (테이블 드리븐)
- [x] (G30) `internal/game/role_test.go` — `RoleAssigner.Assign` 인원별 분배표, 키워드 동일 부여, 대표자 무작위 시드 결정성
- [x] (G31) `internal/game/keyword_test.go` — 풀 크기·중복 검증, `LoadKeywordPool` JSON 라운드트립
- [x] (G32) `internal/game/apply_test.go` — Apply의 모든 액션 × Phase 매트릭스 (정상/에러), 에러 시 state 불변 검증 (NFR-U1-R2)
- [x] (G33) `internal/game/handlers_lifecycle_test.go` — 호스트 권한, StartGame 검증 흐름, ForceEndGame 종착성 (NFR-U1-R4)
- [x] (G34) `internal/game/handlers_night_test.go` — 마피아 대표자 한정, 의사 자가 보호 허용/거부, 경찰 1회 제한, 자기 조사 금지
- [x] (G35) `internal/game/handlers_day_vote_test.go` — 투표 라운드 1/2, RECOUNT 후보 한정, 무처형 흐름
- [x] (G36) `internal/game/resolve_night_test.go` — 보호 성공/실패, 대표자 사망 → 재지정, PeacefulNight, Day++ 검증
- [x] (G37) `internal/game/tally_test.go` — 단일최다·동률·재투표 동률·무처형 시나리오
- [x] (G38) `internal/game/tick_test.go` — Tick 멱등성 (P9), INTRO 자동 진행, DAY 토론 타이머 임계
- [x] (G39) `internal/game/end_test.go` — 시민 승, 마피아 승, 호스트 강제 종료
- [x] (G40) `internal/game/scenario_test.go` — `requirements.md` §5 시나리오 1·4·5 통합 시나리오 (Snapshot/Restore 시나리오 3 포함)
- [x] (G41) `internal/game/property_test.go` — `testing/quick` 속성 기반: Tick 멱등, Snapshot/Restore 라운드트립, Day 단조성

### 3.6 Documentation Generation

- [x] (G42) `aidlc-docs/construction/u1-game-core/code/u1-code-summary.md` — 생성된 파일 목록·역할·라인 카운트 + Build & Test 단계 인풋 가이드
- [x] (G43) `aidlc-docs/construction/u1-game-core/code/u1-public-api.md` — U1이 노출하는 공개 API 카탈로그 (다른 단위 개발 시 참조)

### 3.7 Deployment Artifacts

- [x] (G44) U1 단독 배포 산출물 없음 (단일 바이너리에 통합) — N/A로 표기

### 3.8 Database Migration Scripts

- [x] (G45) U1은 DB 엔티티를 소유하지 않음 — N/A로 표기

### 3.9 Frontend Components

- [x] (G46) U1은 도메인 단위 — N/A로 표기

---

## 4. 단계별 검증 (단위 테스트 실행은 Build & Test에서, 본 단계는 코드 생성만)

본 단계의 Definition of Done:
- [x] (V1) 모든 G1~G46 체크박스 [x] 처리
- [x] (V2) `internal/game`의 모든 파일이 `package game` 선언
- [x] (V3) 외부 의존성 0 (`go.mod`의 require 블록 비어 있음)
- [x] (V4) 모든 공개 식별자에 godoc 주석 작성됨
- [x] (V5) plan 체크리스트의 모든 항목 [x] 마감

### 추가 검증 (Definition of Done 보강)
- [x] (V6) `go build ./...` 통과
- [x] (V7) `go vet ./...` 0 issue
- [x] (V8) `gofmt -l ./internal/game/` 0 lines
- [x] (V9) `go test ./internal/game/...` 모든 테스트 통과
- [x] (V10) `go test -race ./internal/game/...` 통과 (NFR-U1-C2)
- [x] (V11) `go test -cover ./internal/game/...` = **90.4%** (NFR-U1-M1 ≥ 90% 충족)

> Build & Test 단계에서 `go test`, `go vet`, `golangci-lint`, 커버리지 측정을 수행. 본 단계는 **코드 생성만**.

---

## 5. 스토리 추적성 (FR/NFR ↔ 생성 단계)

| 요구사항 | 구현 단계 |
|---|---|
| FR-1.3 (인원 검증) | G10, G29 |
| FR-2.1 (역할 배분) | G16, G30 |
| FR-2.2 (무작위 키워드) | G16, G30 |
| FR-3.1 (키워드 풀) | G13, G14, G31 |
| FR-3.3 (자기소개 시간) | G24, G38 |
| FR-4.1 (단계 전이) | G17~G25, G32~G39 |
| FR-4.2 (밤 행동) | G20, G22, G34, G36 |
| FR-4.3 (마피아 대표자) | G16, G20, G22, G34 |
| FR-4.4 (의사 자가 보호) | G20, G34 |
| FR-4.5 (토론 + 조기 종료) | G21, G24, G35 |
| FR-4.6 (동률 처리) | G23, G35, G37 |
| FR-5.1·5.2 (종료) | G25, G39 |
| FR-7.1 (외부화) | G15, G31 |
| NFR-U1-R1 (정확성) | 모든 *_test.go |
| NFR-U1-R2 (에러 시 불변) | G32 (Apply 매트릭스) |
| NFR-U1-R3 (Tick 멱등) | G24, G38 |
| NFR-U1-R5 (Snapshot/Restore) | G6, G27, G41 |
| NFR-U1-M1~M2 (커버리지) | G26~G41 (모든 테스트) |
| NFR-U1-M5/M9 (외부 의존 0) | G1 |
| NFR-U1-P1~P2 (성능) | G18, G24, G6 (수동 Clone) |
| NFR-U1-S1~S3 (직렬화) | G5 (JSON 태그), G27 |

---

## 6. 단위 산출물 요약 (예상)

| 종류 | 파일 수 | 위치 |
|---|---:|---|
| 도메인 타입·에러 | 7 | `internal/game/*.go` (types/state_clone/action/event/error/validation/visibility) |
| 인프라 인터페이스 | 6 | `internal/game/*.go` (clock/rand/keyword/keyword_pool_data/keyword_loader/role) |
| Engine + 핸들러 | 9 | `internal/game/*.go` (engine/apply/handlers_*/resolve_night/tally/tick/end) |
| 단위 테스트 | 15+ | `internal/game/*_test.go` |
| 워크스페이스 메타 | 2 | `go.mod`, `.gitignore` |
| 문서 요약 | 2 | `aidlc-docs/construction/u1-game-core/code/*.md` |

---

## 7. Part 1 승인 게이트

본 plan에 동의하시면 **"승인"** 또는 **"continue"** 로 답변해 주세요. 변경이 필요하면 구체적 항목(예: "G19를 두 파일로 분리", "G14 풀 크기 변경" 등)을 알려주세요.
