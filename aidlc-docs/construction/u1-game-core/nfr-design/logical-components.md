# Logical Components — U1 Game Core

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-design-patterns.md`, `tech-stack-decisions.md`, `functional-design/*.md`

본 문서는 U1 Game Core의 **논리적 구성요소(Logical Components)** 를 정의합니다. 코드 단계의 패키지·파일 구조의 청사진입니다.

> 참고: U1은 외부 인프라 컴포넌트(큐·캐시·게이트웨이 등)를 갖지 않습니다. 본 단위의 "구성요소"는 모두 도메인 내부 단위(Go 타입·인터페이스·함수)입니다.

---

## 1. 구성요소 카탈로그

| ID | 구성요소 | 종류 | 책임 | 주입 여부 | 적용 패턴 |
|---|---|---|---|---|---|
| LC-1 | `Engine` | 인터페이스 + impl | 상태 머신·Apply·Tick·Snapshot·Restore | 호출자(U2)에게 주입됨 | P1, P2, P7, P9 |
| LC-2 | `RoleAssigner` | 인터페이스 + impl | 인원수→역할 분배, 키워드 부여, 마피아 대표자 선정 | Engine에 주입 | P3, P6 |
| LC-3 | `KeywordPool` | 인터페이스 + 기본 impl | 역할별 키워드 1개 추출 | RoleAssigner에 주입 | P4, P6 |
| LC-4 | `Clock` | 인터페이스 | 현재 시각 제공 | Engine에 주입 | P3 |
| LC-5 | `RNG (io.Reader)` | 표준 인터페이스 | 무작위 시드 source | Engine에 주입 | P3, P6 |
| LC-6 | `Action` 그룹 | sealed interface (sum type) | 외부 입력 식별 | 함수 인자 | P1 |
| LC-7 | `Event` 그룹 | sealed interface (sum type) + Visibility 메타 | 도메인 출력 | Apply 반환값 | — |
| LC-8 | `EngineError` + `ValidationErrors` | 타입드 에러 | 에러 분류 + 누적 | Apply 반환값 | P5, P8 |
| LC-9 | `Validator` 함수 그룹 | 함수 모음 | 사전조건 검증 (Options, Action별) | 핸들러에서 호출 | P5 |
| LC-10 | `State.Clone` | 메서드 | 깊은 복사 | Snapshot/Restore 내부 | P2 |
| LC-11 | `Test Fixtures` | 테스트 헬퍼 | engine·state 빌더, 시나리오 헬퍼 | `_test.go` 한정 | P11 |

---

## 2. 패키지 / 파일 레이아웃 (확정안)

```
internal/game/
├── doc.go                    # godoc 패키지 개요
│
├── types.go                  # PlayerID, Role, Team, Phase, EndReason, Player, Options, State, PendingActions
├── state_clone.go            # State.Clone (P2)
│
├── action.go                 # Action sealed interface + 10종 (LC-6)
├── event.go                  # Event sealed interface + 14종 + Visibility (LC-7)
│
├── error.go                  # ErrorCode, EngineError, sentinel errors (LC-8 P8)
├── validation.go             # ValidationErrors, FieldError, validateOptions, validate* helpers (LC-9 P5)
│
├── engine.go                 # Engine 인터페이스 + engine struct (LC-1, P3, P7)
├── apply.go                  # Apply의 type-switch dispatch (P1)
├── handlers_*.go             # handleStartGame.go, handleNight.go, handleVote.go, handleHost.go (LC-1 sub)
├── tick.go                   # Tick 멱등 알고리즘 (P9)
├── resolve_night.go          # NIGHT → DAY 전이 (kill/heal 적용)
├── tally.go                  # 투표 집계 + RECOUNT 처리
│
├── role.go                   # RoleAssigner 인터페이스 + 기본 impl (LC-2)
├── keyword.go                # KeywordPool 인터페이스 + mapKeywordPool (LC-3)
├── keyword_pool_data.go      # 기본 한국어 풀 140개 (P4)
├── keyword_loader.go         # LoadKeywordPool(io.Reader) (FR-7.1)
│
├── clock.go                  # Clock 인터페이스 + realClock (LC-4)
├── rand.go                   # extractSeed64, inner PRNG 헬퍼 (P6)
│
├── visibility.go             # Visibility enum + EventEnvelope (가시성 메타)
│
└── tests/                    # 단위·시나리오·속성 기반 테스트
    ├── apply_test.go
    ├── tick_test.go
    ├── resolve_night_test.go
    ├── tally_test.go
    ├── role_test.go
    ├── keyword_test.go
    ├── error_test.go
    ├── snapshot_test.go
    ├── scenario_test.go      # requirements §5 시나리오
    ├── property_test.go      # testing/quick 기반
    └── fixtures.go           # 빌더 헬퍼 (LC-11)
```

> 비고: Go 관용에 따라 `_test.go` 파일은 보통 같은 디렉터리에 둡니다. 위 `tests/` 분리는 가독성을 위한 잠정 제안 — Code Generation 단계에서 단일 디렉터리 평탄 구성으로 조정 가능.

---

## 3. 구성요소별 상세

### 3.1 LC-1 Engine

**의도**: 상태 머신·규칙·시간 진전·영속 직렬화 진입점.

**인터페이스**:
```go
type Engine interface {
    Start(players []PlayerID, opts Options) (State, []Event, error)
    Apply(action Action) (State, []Event, error)
    Tick(now time.Time) (State, []Event, error)
    Snapshot() State
    Restore(s State) error
}
```

**불변식**:
- 동시 호출 비안전 — 단일 스레드 가정 (P7)
- Apply 에러 → state 무변화 (NFR-U1-R2)
- Tick 멱등 (NFR-U1-R3, P9)

**의존**: Clock, RNG, RoleAssigner.

---

### 3.2 LC-2 RoleAssigner

**의도**: 인원수·옵션 기반 역할 분배 + 마피아 대표자 결정 (Q-FD-U1-4-FU=C).

**인터페이스**:
```go
type Assignments struct {
    PlayerRoles    map[PlayerID]Role
    PlayerKeywords map[PlayerID]string
    MafiaIDs       []PlayerID
    Representative PlayerID
}

type RoleAssigner interface {
    Assign(playerIDs []PlayerID, opts Options, rng *rand.Rand) (Assignments, error)
}
```

**기본 구현**:
- `defaultAssigner{pool KeywordPool}` — 셔플·역할 부여·키워드 동일 부여·대표자 무작위.

---

### 3.3 LC-3 KeywordPool

**의도**: 역할별 한국어 키워드 풀 (FR-7.1 외부화 인터페이스).

**인터페이스**:
```go
type KeywordPool interface {
    Pick(role Role, rng *rand.Rand) string
}

type mapKeywordPool struct {
    Mafia, Citizen, Doctor, Police []string
}

func (p mapKeywordPool) Pick(role Role, rng *rand.Rand) string { ... }
```

**기본 콘텐츠**: `keyword_pool_data.go`의 4 슬라이스 (P4).

**외부 로딩**:
```go
func LoadKeywordPool(r io.Reader) (KeywordPool, error)  // JSON
```

---

### 3.4 LC-4 Clock

**의도**: 시간 의존 부분(Tick, INTRO 발화자 시작 시각, 토론 deadline)의 테스트 가능성.

**인터페이스**:
```go
type Clock interface {
    Now() time.Time
}

type realClock struct{}
func (realClock) Now() time.Time { return time.Now() }

// 테스트
type FakeClock struct { t time.Time }
func (c *FakeClock) Now() time.Time { return c.t }
func (c *FakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }
```

> Engine.Tick(now time.Time)의 인자로 외부에서 `now`를 직접 전달하므로 Engine이 내부적으로 Clock을 호출하는 경우는 적음. 일부 핸들러(예: handleStartGame에서 `StartedAt` 기록)에서만 사용.

---

### 3.5 LC-5 RNG

**의도**: 무작위 시드 source. 운영 = `crypto/rand.Reader`, 테스트 = `bytes.NewReader([]byte{...})`.

**적용**:
```go
type engine struct {
    rng io.Reader
    ...
}

// 사용 시
seed, _ := extractSeed64(e.rng)
inner := rand.New(rand.NewSource(seed))
```

---

### 3.6 LC-6/7 Action / Event 그룹

**Action 10종** (`action.go`):
- StartGame, AdvanceIntro, SubmitMafiaKill, SubmitDoctorHeal, SubmitPoliceCheck, EndNightEarly, EndDiscussionEarly, SubmitVote, ToggleVoice, ForceEndGame

**Event 14종** (`event.go`):
- GameStarted, PhaseChanged, RoleRevealedToPlayer, MafiaCohortRevealed, IntroSpeakerChanged, MafiaTargetSelected, PoliceResult, DeathAnnounced, PeacefulNight, DiscussionTimerTick, VoteTallied, Eliminated, MafiaRepresentativeReassigned, GameEnded, VoiceToggled

**Visibility 메타** (`visibility.go`):
```go
type Visibility int
const (
    VisPublic Visibility = iota         // 모든 PublicView + 살아있는 모든 PlayerView
    VisPlayer                            // 특정 PlayerID 1인
    VisRoleMafia                         // 살아있는 모든 마피아
    VisRolePolice                        // 해당 경찰 1인
)

type EventEnvelope struct {
    Event      Event
    Visibility Visibility
    PlayerID   PlayerID  // VisPlayer/VisRolePolice일 때
}
```

> Apply/Tick 반환의 `[]Event`는 실제로는 `[]EventEnvelope`. U3가 envelope 메타를 보고 라우팅. (정확한 시그니처는 Code Generation에서 미세 조정.)

---

### 3.7 LC-8 EngineError / ValidationErrors

**구조**:
- `EngineError` (단일 위반) — `errors.Is/As` 호환 (P8)
- `ValidationErrors []FieldError` (다중 위반) — Options 검증 등 누적 결과 (P5)

```go
type ErrorCode string  // 9종 (NFR-Req §5)

type EngineError struct {
    Code    ErrorCode
    Message string
    Field   string
    Want    any
    Got     any
}
```

---

### 3.8 LC-9 Validator 함수 그룹 (`validation.go`)

```go
// 호스트 한정
func ensureHost(state *State, sender PlayerID) error
// 단계 체크
func ensurePhase(state *State, phases ...Phase) error
// 역할 체크
func ensureRole(state *State, sender PlayerID, role Role) error
// 살아있는 플레이어
func ensureAlive(state *State, ids ...PlayerID) error
// 누적 검증
func validateOptions(opts Options, playerCount int) ValidationErrors
```

**규칙**: 단일 위반 분기는 fail-fast (`*EngineError` 즉시 반환), 다중 규칙은 누적 (`ValidationErrors`).

---

### 3.9 LC-10 State.Clone

§3.2 P2 참조. 모든 슬라이스/맵/포인터를 비공유 사본으로.

---

### 3.10 LC-11 Test Fixtures (`fixtures.go`)

```go
// 빌더
func newTestEngine(t *testing.T, opts ...EngineOpt) Engine
func mustStartGame(t *testing.T, e Engine, n int) State

// 시나리오 헬퍼
func playFirstNight(t *testing.T, e Engine, mafiaTarget, doctorTarget, policeTarget PlayerID)

// 시드 PRNG
func deterministicRNG(seed int64) io.Reader
```

---

## 4. 책임 매트릭스 (NFR ↔ LC)

| NFR Req | 책임 LC |
|---|---|
| NFR-U1-R1 (규칙 정확성) | LC-1, LC-9 |
| NFR-U1-R2 (에러 시 불변) | LC-1, LC-8 |
| NFR-U1-R3 (Tick 멱등) | LC-1 (Tick), LC-4 |
| NFR-U1-R4 (END 종착) | LC-1 (Apply default 분기) |
| NFR-U1-R5 (Snapshot/Restore) | LC-1, LC-10 |
| NFR-U1-R6 (대표자 유효) | LC-2 (배정), LC-1 (handler에서 재지정) |
| NFR-U1-M1~M2 (커버리지) | LC-11 (fixtures가 테스트 작성 가속) |
| NFR-U1-M5 (도메인 순수성) | 모든 LC가 외부 lib 미사용 |
| NFR-U1-M6 (시드 주입) | LC-5, LC-2 |
| NFR-U1-P1~P2 (Apply/Tick p99 < 1ms) | LC-1, LC-10 (Clone 성능) |
| NFR-U1-S1~S3 (직렬화) | LC-1 + types.go의 JSON 태그 |
| NFR-U1-C1 (단일 스레드) | LC-1 godoc 명시 |

---

## 5. 인프라 / 외부 컴포넌트

**N/A** — U1은 큐·캐시·DB·게이트웨이 등 외부 인프라 컴포넌트를 사용하지 않음.

> 영속화·라우팅·HTTP는 모두 다른 단위(U2~U4)의 책임. 본 문서는 U1 내부 구성에 한정.

---

## 6. 검증 체크리스트

- [x] 모든 LC가 1개 패키지 내 정의 (`internal/game`)
- [x] 외부 의존 0 (NFR-U1-M5/M9)
- [x] 모든 LC가 nfr-design-patterns.md의 패턴 P1~P11 중 하나 이상 적용
- [x] 모든 NFR Req 항목이 책임 LC에 매핑됨 (§4)
- [x] 도메인 타입(types.go)·인터페이스 분리·테스트 fixture 분리 완비
- [x] FR-7.1 외부화(KeywordPool 인터페이스)가 LC-3·keyword_loader.go로 보장
