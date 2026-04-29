# Tech Stack Decisions — U1 Game Core

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-requirements.md`, 사용자 응답 Q-NFR-U1-1~12 모두 A (2026-04-26)

본 문서는 U1 Game Core의 기술 스택 결정과 그 근거를 기록합니다. 본 결정은 Code Generation 단계의 입력입니다.

---

## 1. 결정 요약 (의사결정 매트릭스)

| 영역 | 결정 | 출처 | 근거 |
|---|---|---|---|
| 프로그래밍 언어 | **Go** | Q-AD-2=A (Application Design) | 단일 바이너리 운영 단순성, gorilla/websocket 합치, NFR-7 |
| Go 버전 | **Go 1.22+** | Q-NFR-U1-1=A | 제네릭, range over int, 최신 표준 lib |
| 외부 의존성 | **0개 (표준 라이브러리만)** | Q-NFR-U1-2=A | 도메인 순수성, FR-7.1 인터페이스만 |
| 무작위성 (운영) | **`crypto/rand`** | Q-FD-U1-10=A, Q-NFR-U1-4=A | 비결정적 보안 강도 |
| 무작위성 (테스트) | **시드 가능 PRNG (`io.Reader` 주입)** | Q-NFR-U1-4=A | 시나리오 재현, 결정적 단위 테스트 |
| 동시성 모델 | **단일 스레드 (호출자 책임)** | Q-NFR-U1-9=A, BR-CONC-1 | U2가 단일 mutex/액터로 직렬화 |
| 에러 표현 | **타입드 에러 + `errors.Is/As`** | Q-NFR-U1-8=A | Go 1.13+ 표준 패턴 |
| 직렬화 형식 | **`encoding/json` (표준 lib)** | Q-NFR-U1-7=A, NFR-U1-S1 | U2 SQLite BLOB 호환, 외부 의존 0 |
| 정적 분석 | **`go vet` + `golangci-lint`(default)** + `gofmt` | Q-NFR-U1-5=A | 일반적 Go 품질 게이트 |
| 단위 테스트 도구 | **`testing`(표준)** + 테이블 드리븐 + 시나리오 + 속성 기반 | Q-NFR-U1-10=A | 표준 패턴, 외부 의존 0 |
| 속성 기반 테스트 | **`testing/quick`(표준)** 우선, 필요 시 `gopter` 추가 검토 | Q-NFR-U1-10=A | 표준 lib 우선 정책 |
| 벤치마크 | **`testing` 표준 벤치마크** | Q-NFR-U1-6=A | 표준 도구 |
| 빌드 재현성 | **`go.mod` + `go.sum` 커밋, vendor 미사용, `go mod verify`** | Q-NFR-U1-12=A | 일반적 모듈 운영 |
| 문서화 | **공개 식별자 godoc 의무** | Q-NFR-U1-11=A | Go 관용 |

---

## 2. 의존성 트리 (예상)

```
internal/game/
├── (외부 의존) — 없음
├── (표준 lib 사용)
│   ├── crypto/rand
│   ├── encoding/json
│   ├── errors
│   ├── fmt
│   ├── io
│   ├── math/rand           // 테스트 전용 (시드 PRNG)
│   ├── sort                // 결정적 직렬화 보조
│   ├── testing             // 단위 테스트
│   ├── testing/quick       // 속성 기반 테스트
│   └── time
└── (자체 인터페이스만 추상화)
    ├── KeywordPool
    ├── Clock (테스트 가능성)
    └── RoleAssigner (구현 분리)
```

> 주의: `math/rand`는 **테스트 전용**. 운영 코드는 `crypto/rand`를 통해 시드를 받는 inner PRNG 만 허용.

---

## 3. Go 1.22+ 채택 이유

- **`for i := range N`** 문법 활용 → 자기소개 발화자 인덱스 순회 등 가독성 향상
- **루프 변수 별도 캡처** (1.22 변경) — 클로저 버그 회피
- **표준 라이브러리 개선**: `slices`, `maps` (1.21+), `cmp` 등 — 자체 헬퍼 작성 최소화
- 호환성: 사내 도구 PoC → 최신 LTS 채택에 위험 적음

---

## 4. 무작위성 정책 상세

### 4.1 운영 (production)

```
// 의사 코드
package game

func New(assigner RoleAssigner, clock Clock, rng io.Reader) Engine
```

- `rng`은 `crypto/rand.Reader`를 기본 주입.
- RoleAssigner와 Engine은 `rng`에서 64비트 시드를 추출 후 `math/rand.New(math/rand.NewSource(seed))`로 inner PRNG 생성. 매 게임마다 새 시드.

### 4.2 테스트

```
// 단위 테스트
seed := []byte{0x01, 0x02, ...}
rng := bytes.NewReader(seed)  // 결정적 io.Reader
engine := game.New(assigner, fakeClock, rng)
```

- 테스트는 시드 PRNG로 시나리오 재현. 동일 시드 → 동일 결과.

### 4.3 결정성 단위 (race-free)

- 단위 테스트는 한 번에 한 Engine만 다룸 (병렬 실행 시 각 Engine이 독립).
- `go test -race`로 데이터 레이스 미발생 검증.

---

## 5. 에러 표현 정책 (Q-NFR-U1-8=A)

### 5.1 에러 타입 (최종 시그니처는 Code Generation에서 확정)

```go
package game

type ErrorCode string

const (
    CodeValidation         ErrorCode = "VALIDATION_ERROR"
    CodeWrongPhase         ErrorCode = "WRONG_PHASE_ERROR"
    CodePermissionDenied   ErrorCode = "PERMISSION_DENIED_ERROR"
    CodeRoleMismatch       ErrorCode = "ROLE_MISMATCH_ERROR"
    CodeNotRepresentative  ErrorCode = "NOT_REPRESENTATIVE_ERROR"
    CodeDeadPlayer         ErrorCode = "DEAD_PLAYER_ERROR"
    CodeAlreadyDone        ErrorCode = "ALREADY_DONE_ERROR"
    CodeInvalidTarget      ErrorCode = "INVALID_TARGET_ERROR"
    CodeUnknownPlayer      ErrorCode = "UNKNOWN_PLAYER_ERROR"
)

type EngineError struct {
    Code    ErrorCode
    Message string
    // 선택 필드(검증 에러 등에서 활용)
    Field   string
    Want    any
    Got     any
}

func (e *EngineError) Error() string { ... }
func (e *EngineError) Is(target error) bool { ... }  // Code 매칭
```

- `errors.Is(err, &EngineError{Code: CodeValidation})` 패턴 지원
- `errors.As(err, &engErr)`로 상세 필드 추출
- 에러 메시지는 한국어 사용자 안내가 아니라 **개발자용 영문 메시지** — 사용자 안내는 U2 AnnouncementService 책임

---

## 6. 패키지 레이아웃 결정 (Code Generation 인풋)

```
internal/game/
├── doc.go                  // 패키지 godoc
├── types.go                // 도메인 타입 (PlayerID, Role, Phase, Player, State, Options 등)
├── action.go               // Action sealed type 그룹
├── event.go                // Event sealed type 그룹
├── error.go                // EngineError, ErrorCode
├── engine.go               // Engine 인터페이스 + impl
├── apply.go                // Apply 핸들러 8종 (handleStartGame 등)
├── tick.go                 // Tick 알고리즘
├── resolve_night.go        // resolveNight 알고리즘
├── tally.go                // 투표 집계
├── role.go                 // RoleAssigner 인터페이스 + impl
├── keyword.go              // KeywordPool 인터페이스 + default pool 임베드
├── clock.go                // Clock 인터페이스 (테스트 가능성)
├── rand.go                 // RNG 헬퍼 (crypto/rand → seed → math/rand inner PRNG)
└── *_test.go               // 단위 테스트 + 시나리오 + 속성 기반
```

> 정확한 분할은 Code Generation 단계에서 조정 가능. 본 문서는 의도된 책임 분리만 명시.

---

## 7. 외부 의존성 변경 영향 평가

| 변경 | 본 결정의 영향 |
|---|---|
| Go 버전 다운그레이드 (1.21 이하) | `slices`, `maps` 등 일부 호출 변경 필요. 큰 영향 없음 |
| 외부 lib 도입 (예: `go-cmp`) | NFR-U1-M5/M9 위반 — **금지**. 표준 `reflect.DeepEqual`로 대체 |
| `gopter`(속성 기반) 도입 | 테스트 의존 → 운영 코드와 분리되므로 NFR-M5 영향 없음. `testing/quick`으로 충분하면 미도입 |
| `golangci-lint` 엄격 모드로 강화 | 본 결정은 default set. 추후 강화 가능. |

---

## 8. 결정 검증 체크리스트

- [x] 모든 결정이 Q-NFR-U1-1~12 사용자 응답과 일치
- [x] FR-7.1 (외부화 가능 인터페이스)와 충돌 없음
- [x] NFR-U1-M5 (도메인 순수성)와 충돌 없음
- [x] NFR-U1-M8 (빌드 재현성)와 충돌 없음
- [x] U2~U5와의 인터페이스 계약 침해 없음 (도메인 타입 노출만, U1이 정의처)
- [x] 운영/테스트 환경 분리 명확 (무작위성, Clock 주입)
