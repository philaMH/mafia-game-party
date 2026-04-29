# NFR Requirements — U1 Game Core

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 NFR-1/2/6, `construction/u1-game-core/functional-design/*.md`, `plans/u1-game-core-nfr-requirements-plan.md`

본 문서는 U1 Game Core의 비기능 요구사항을 측정 가능한 형태로 정의합니다. NFR Design 단계에서 본 한도를 만족하는 패턴을 적용하고, Code Generation/Build & Test에서 실제 검증합니다.

> 우선순위: **Reliability > Maintainability > Performance**. Scalability·Security·Availability는 직접 책임 영역 아님(다른 단위로 위임).

---

## 1. NFR 영역별 요구사항 표

### 1.1 Reliability / Correctness (NFR-1 안정성 — 최우선)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U1-R1 | 게임 규칙 정확성 | `business-rules.md` 모든 BR-* 항목이 단위 테스트로 검증됨 (네거티브 테스트 포함) | `go test ./internal/game/...` 100% 통과 |
| NFR-U1-R2 | 액션 에러 시 상태 불변성 | Apply가 에러를 반환할 때 state는 호출 직전과 비트단위 동일 | 속성 기반 테스트 (BR-CONC-3 검증) |
| NFR-U1-R3 | Tick 멱등성 | 동일 `now`로 Tick을 N번 호출 시 첫 호출 외 모든 호출이 no-op (반환 events 비어 있음, state 동일) | 단위 테스트 |
| NFR-U1-R4 | 종료 단계 종착성 | `Phase=END` 진입 후 어떤 액션도 state를 변경하지 않음 (모두 에러 반환) | 단위 테스트 (모든 액션 종류 × END 단계) |
| NFR-U1-R5 | Snapshot ↔ Restore 라운드트립 | 임의 State `s`에 대해 `Restore(Snapshot(s))`가 의미 동등 | 속성 기반 테스트 |
| NFR-U1-R6 | 마피아 대표자 항상 유효 | `MafiaRepresentativeID`는 (a) 살아있는 MAFIA Player이거나 (b) 마피아 전멸 시 빈 값 — 두 경우만 존재 | 불변식 테스트, 사망 처리 후 즉시 검사 |
| NFR-U1-R7 | 단계 전이 무한 루프 방지 | Apply 1회 호출은 최대 1회 단계 전이만 트리거 (자동 연쇄 전이 시 안전 한도) | 코드 리뷰 + 시나리오 테스트 |

### 1.2 Maintainability / Testability (NFR-6)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U1-M1 | 단위 테스트 라인 커버리지 | **≥ 90%** (`internal/game` 패키지) | `go test -cover` |
| NFR-U1-M2 | 단위 테스트 분기 커버리지 | **≥ 85%** | `go test -cover -covermode=atomic` 또는 `go-test-coverage` 도구 |
| NFR-U1-M3 | 정적 분석 통과 | `go vet`, `golangci-lint`(default set), `gofmt -l ./... → empty` | CI 게이트 |
| NFR-U1-M4 | godoc 주석 완비 | `internal/game`의 모든 공개 식별자(`Engine`, `RoleAssigner`, `Action 종류`, `Event 종류`, `Role`, `Phase`, `State`, `Options`, `New*`)에 godoc 주석 존재 | `revive` 또는 `golint` 같은 export-comment 룰 활성 |
| NFR-U1-M5 | 도메인 순수성 | `internal/game/*`는 `net/`, `os/`, `database/`, `encoding/json`(가능하면) 등 외부 I/O 패키지 미import | `go list -deps`로 의존 그래프 검사 (CI) |
| NFR-U1-M6 | 결정성 / 시드 주입 가능성 | `Engine.New(...)` 시그니처에 `rng io.Reader` 주입 인자 존재. 단위 테스트는 결정적 PRNG로 시나리오를 재현 가능 | 단위 테스트 |
| NFR-U1-M7 | 테스트 종류 분포 | 테이블 드리븐 테스트(주력) + 시나리오 테스트(시나리오 1~7 일부) + 속성 기반 테스트(투표 집계, Tick 멱등성, Snapshot/Restore 라운드트립) | 코드 리뷰 |
| NFR-U1-M8 | 빌드 재현성 | `go.mod`+`go.sum` 커밋, vendor 미사용. CI `go mod verify` 통과 | CI 게이트 |
| NFR-U1-M9 | 외부 의존성 | 0개 (표준 라이브러리만, FR-7.1 인터페이스만) | `go list -m all`이 표준 lib만 포함 |

### 1.3 Performance (NFR-2)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U1-P1 | Apply 호출 지연 | p99 < **1 ms** (12명, JSON 직렬화 제외) | Go benchmark (`go test -bench=BenchmarkApply`) |
| NFR-U1-P2 | Tick 호출 지연 | p99 < **1 ms** (12명) | Go benchmark |
| NFR-U1-P3 | 시간 복잡도 | Apply, Tick 모두 O(N) 이내 (N=플레이어 수). 큐 정렬 등 N log N도 허용. | 코드 리뷰 |
| NFR-U1-P4 | 메모리 할당 | Tick은 변경이 없을 때 0 추가 할당(no-op 경로). | `go test -bench=. -benchmem` 검증 |

### 1.4 Storage / Serialization (NFR-1 보조)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U1-S1 | State JSON 직렬화 가능 | `json.Marshal(state)`가 에러 없이 통과. 모든 필드가 JSON 태그 또는 export 처리 | 단위 테스트 |
| NFR-U1-S2 | Snapshot 크기 한도 | 12명 활성 게임의 JSON 직렬화 크기 < **32 KB** | 단위 테스트 (size assertion) |
| NFR-U1-S3 | 직렬화 결정성 | 동일 state에서 두 번 Marshal 시 바이트 동일 (map 직렬화 안정성 보장) | `json.Marshal` 결과 비교 + 정렬 |

### 1.5 Concurrency

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U1-C1 | 단일 스레드 가정 | GameEngine 인터페이스에 동시 호출 안전 보장 명시 안 함. 호출자(U2)가 직렬화 책임 | 문서화 + godoc |
| NFR-U1-C2 | 데이터 레이스 미발생 (단위 테스트) | `go test -race ./internal/game/...` 통과 | CI 게이트 |

---

## 2. NFR 우선순위 / 트레이드오프

| 트레이드오프 | 본 단위의 결정 | 근거 |
|---|---|---|
| 정확성 vs 성능 | **정확성 우선** | 단일 게임, 12명 한정 — 성능 여유 큼. 코드 명료성 > 마이크로 최적화 |
| 결정성 vs 운영 보안 | **운영=비결정적(`crypto/rand`), 테스트=결정적 시드 주입** | NFR-U1-R5/M6 양립 (Q-FD-U1-10, Q-NFR-U1-4) |
| 외부 lib 사용 vs 표준 lib만 | **표준 lib만** | 도메인 순수성, Q-NFR-U1-2=A |
| 커버리지 강도 vs 작성 비용 | **라인 90% / 분기 85%** | 도메인 핵심 — 테스트가 가치의 토대 |

---

## 3. 추적성 (FR/NFR ↔ U1 NFR 항목)

| 출처 | 본 문서 항목 |
|---|---|
| NFR-1 (안정성·복원) | NFR-U1-R1~R7, NFR-U1-S1~S3 |
| NFR-2 (성능) | NFR-U1-P1~P4 |
| NFR-6 (도메인 분리·유지보수성·확장성) | NFR-U1-M1~M9 |
| NFR-7 (운영 단순성) | NFR-U1-M9 (외부 의존 0), NFR-U1-M8 (재현 빌드) |
| FR-7.1 (외부화 인터페이스) | NFR-U1-M5, NFR-U1-M9 |

---

## 4. 검증 게이트 (Build & Test 단계에서 강제)

다음 모든 항목이 통과해야 U1이 출하 가능:
1. ✅ `go vet ./internal/game/...` — 0 issue
2. ✅ `golangci-lint run ./internal/game/...` (default set) — 0 issue
3. ✅ `gofmt -l ./internal/game/... | wc -l` == 0
4. ✅ `go test -race ./internal/game/...` — 모든 테스트 통과
5. ✅ `go test -cover ./internal/game/...` — 라인 ≥ 90%, 분기 ≥ 85% (분기 커버리지는 별도 도구)
6. ✅ `go test -bench=. -benchmem ./internal/game/...` — Apply/Tick p99 < 1ms (12명)
7. ✅ `go list -deps ./internal/game/...` — 표준 lib + 자체 패키지만 (외부 모듈 0)
8. ✅ State JSON 직렬화 < 32 KB (12명 활성 게임 — 별도 단위 테스트)

---

## 5. 비고 / 명시적 비-요구사항 (Non-Goals)

- **Scalability**: 멀티 인스턴스/멀티 게임 동시 실행은 본 단위 책임 아님. SessionManager(U2)가 단일 세션을 가정.
- **Security**: 인증·권한 — U3/U4 (네트워크 표면). U1은 액션 사전조건 검증만.
- **Availability SLA**: 직접 책임 없음. NFR-1 복원성은 Snapshot/Restore 인터페이스만 제공.
- **국제화 (i18n)**: 키워드 풀은 한국어 한정 (FR-8 한국어 단방향). 다국어 풀은 본 단위 비책임 (외부화 인터페이스로 추후 확장).
- **하위 호환성**: 본 도구는 사내 PoC. 스냅샷 포맷 마이그레이션은 본 NFR 범위 외.
