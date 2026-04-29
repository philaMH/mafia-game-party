# NFR Requirements Plan — U1 Game Core

**작성일**: 2026-04-26
**대상 단위**: U1 Game Core (`internal/game/*`)
**참조**:
- `requirements.md` v1.1 NFR-1, NFR-2, NFR-6 (특히 도메인 분리)
- `construction/u1-game-core/functional-design/*.md`
- `application-design/unit-of-work-story-map.md` U1 Primary

---

## 0. U1 NFR 컨텍스트 요약

U1은 **외부 I/O 0의 순수 도메인**입니다. 따라서 NFR 우선순위는 다음과 같이 자연스럽게 정해집니다:

| NFR 영역 | U1에서의 우선순위 | 근거 |
|---|---|---|
| **Reliability / Correctness** | 🔴 최우선 | NFR-1 안정성 최우선, 게임 규칙 정확성이 도구 가치의 핵심 |
| **Maintainability / Testability** | 🟠 높음 | NFR-6, 도메인 단위 테스트가 전체 안정성의 토대 |
| **Performance** | 🟢 낮음(여유) | 단일 게임 인스턴스, 최대 12명, Apply 호출 빈도 낮음 (인간 게임 페이스). 다만 Tick 멱등성·O(N) 보장 필요 |
| **Scalability** | 🟢 N/A | 단일 세션, 멀티 인스턴스 확장 요구 없음 |
| **Security** | 🟢 N/A | U1은 사용자 입력을 직접 받지 않음 (검증은 액션 사전조건). 공격 표면은 U3/U4 |
| **Availability** | 🟢 간접 | 직접 가용성 SLA 없음. NFR-1 복원 가능성을 Snapshot/Restore로 지원만 |

> **본 단계의 핵심 결정**: 정확성 보장(테스트 커버리지 목표·결정성), 코드 품질 게이트(린터·정적 분석), 성능 한계(상한 budget), Go 기본 환경.

---

## 1. 작업 체크리스트 (계획)

- [x] (1) FD 산출물 분석 + NFR 영역 우선순위 매핑
- [x] (2) 결정 질문지 작성 (§3)
- [x] (3) 본 plan에 [Answer]: 태그 임베드
- [x] (4) 사용자 답변 수집 — Q-NFR-U1-1~12 모두 A
- [x] (5) 모순/모호성 점검 — 일관성 확인
- [x] (6) 사용자 승인 — Generation 진입

## 2. 작업 체크리스트 (생성, 승인 후)

- [x] (7) `nfr-requirements.md` 작성 — 5개 영역 NFR 표(R/M/P/S/C) + 검증 게이트 8종 + 비-요구사항
- [x] (8) `tech-stack-decisions.md` 작성 — Go 1.22+, 외부 의존 0, RNG 정책, 에러 정책, 패키지 레이아웃
- [x] (9) FR/NFR 추적성 검증 (`nfr-requirements.md` §3)
- [x] (10) `aidlc-state.md` 갱신
- [x] (11) 사용자 승인 게이트 제시

---

## 3. 결정 질문 (사용자 답변 필요)

> 답변: `[Answer]: <기호>` 또는 자유 의견.

---

### Q-NFR-U1-1. Go 버전 및 표준 라이브러리 정책

- A. **Go 1.22+** (제네릭 활용 가능, 최신 표준 라이브러리 — 권장)
- B. Go 1.21
- C. Go 1.20
- D. Other (자유의견란에 명시)

[Answer]: 

---

### Q-NFR-U1-2. U1 외부 의존성 정책 (FR-7.1과 일치)

- A. **표준 라이브러리만** + U1 내부 인터페이스(KeywordPool, Clock, RoleAssigner)만 import. 외부 모듈 0. (도메인 순수성 — 권장)
- B. 일부 보조 라이브러리 허용 (예: `github.com/google/uuid` 등)
- C. Other

[Answer]: A 

---

### Q-NFR-U1-3. 단위 테스트 커버리지 목표 (NFR-1, NFR-6)

U1은 도메인 핵심이라 가장 강한 커버리지가 필요합니다.

- A. **라인 커버리지 ≥ 90%, 분기 커버리지 ≥ 85%** (권장 — 도메인은 정확성이 가치의 핵심)
- B. 라인 커버리지 ≥ 80% (일반적 목표)
- C. 라인 커버리지 ≥ 70% (최소 보증)
- D. Other (자유의견란에 수치 명시)

[Answer]: A

---

### Q-NFR-U1-4. 결정성 / 무작위성 테스트 정책

- A. **모든 무작위 동작은 시드 가능 인터페이스(`io.Reader` 주입)로 테스트에서 결정적으로 검증**. 운영은 `crypto/rand`. (권장 — Q-FD-U1-10=A와 일치)
- B. 운영도 `math/rand` 사용으로 단순화 (시드 노출)
- C. Other

[Answer]: A

---

### Q-NFR-U1-5. 정적 분석 / 린터 게이트

- A. **`go vet` + `golangci-lint`(기본 enabled set)** + `gofmt` 강제 — CI 게이트로 동작 (권장)
- B. `go vet` + `gofmt`만 (가벼운 게이트)
- C. `golangci-lint` 엄격 모드 (govet + staticcheck + revive + unparam + errcheck + gocritic 등 강한 set)
- D. Other (자유의견란에 set 명시)

[Answer]: A

---

### Q-NFR-U1-6. Apply / Tick 성능 한도 (Performance Budget)

U1은 단일 게임에서 호출 빈도가 낮지만, Tick은 1초마다 호출됩니다. Apply는 사용자 입력마다.

- A. **Apply: p99 < 1ms, Tick: p99 < 1ms (12명 기준)**. (성능에 여유, 권장)
- B. Apply: p99 < 5ms, Tick: p99 < 5ms (느슨함)
- C. 측정만 하고 명시적 한도 없음 (NFR 미정의)
- D. Other

[Answer]: A

---

### Q-NFR-U1-7. 메모리 / 스냅샷 크기 (NFR-1)

12명 + 모든 게임 상태를 1개 State 구조에 직렬화. 직렬화 형식·크기 한도:

- A. **State는 JSON 직렬화 가능해야 함, 크기 < 32 KB (12명 기준)**. (권장 — U2 SQLite BLOB 저장 호환)
- B. 메모리 표현은 자유, 직렬화는 U2 책임
- C. Other

> 참고: 직렬화 형식(JSON vs Protobuf 등) 자체는 U2 PersistenceStore 책임이지만, U1의 State 타입이 그것을 지원하도록 설계되어야 함.

[Answer]: A

---

### Q-NFR-U1-8. 에러 표현 (Reliability)

`business-rules.md` §10에 9종 에러 코드가 분류되어 있습니다.

- A. **타입드 에러 (`type ValidationError struct{...}`) + `errors.Is/As` 호환** + 상수 에러 코드. (권장 — Go 표준 패턴)
- B. **단일 sentinel 에러 + 코드 enum** (`var ErrValidation = errors.New(...)`)
- C. Other

[Answer]: A

---

### Q-NFR-U1-9. 동시성 안전성 (NFR-1, BR-CONC)

- A. **GameEngine는 단일 스레드 가정** (동시 호출 안전성은 U2 SessionManager 단일 mutex/액터가 보장). 본 단위 자체에 락 없음. (권장 — `business-rules.md` BR-CONC-1과 일치)
- B. GameEngine 내부에 자체 mutex 도입 (이중 보호)
- C. Other

[Answer]: A

---

### Q-NFR-U1-10. 테스트 종류 (Maintainability)

- A. **테이블 드리븐 단위 테스트** (Go 관용 패턴) + **시나리오 단위 테스트**(시나리오 1~7 일부) + **속성 기반 테스트**(투표 집계·상태 머신 불변식 일부). (권장 — 도메인 단위 강 커버리지)
- B. 테이블 드리븐 단위 테스트 + 시나리오 단위 테스트 (속성 기반 제외)
- C. 단위 테스트만
- D. Other

[Answer]: A

---

### Q-NFR-U1-11. 문서화 / 코드 코멘트 정책 (Maintainability)

- A. **공개 식별자(Engine, RoleAssigner, Action 타입 등)에 godoc 주석 의무**. 비공개 식별자는 자율. (권장 — Go 관용)
- B. 모든 식별자에 주석 강제
- C. 주석 자율
- D. Other

[Answer]: A

---

### Q-NFR-U1-12. 빌드 재현성 (Maintainability / 운영)

- A. **`go.mod` + `go.sum` 커밋, GitHub-style vendoring 미사용**. CI에서 `go mod verify`. (권장)
- B. `vendor/` 디렉터리 커밋 (오프라인 빌드 보장)
- C. Other

[Answer]: A

---

### 자유 의견란

추가 NFR 또는 변경 제안.

[Answer]: 

---

## 4. 분석 단계 (Step 5, 답변 후)

답변 수령 후 모호성 점검.

## 5. 산출물 미리보기 (Generation 시 작성)

- **`nfr-requirements.md`** — U1 NFR 표 (영역별 요구사항·측정 가능한 한도·검증 방법)
- **`tech-stack-decisions.md`** — Go 버전, 표준 라이브러리 정책, 외부 의존, 린터, 테스트 도구 결정과 근거
