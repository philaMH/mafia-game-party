# NFR Design Plan — U1 Game Core

**작성일**: 2026-04-26
**대상 단위**: U1 Game Core (`internal/game/*`)
**참조**: `construction/u1-game-core/nfr-requirements/*.md`, `construction/u1-game-core/functional-design/*.md`

---

## 0. 컨텍스트

U1은 **외부 I/O 0의 순수 도메인 단위**라 일반적인 NFR Design 카테고리(Resilience/Scalability/Security)는 대부분 N/A입니다. 본 단계에서 결정해야 하는 패턴은 다음 영역에 한정됩니다:

| 카테고리 | U1에서 다룰 범위 |
|---|---|
| Resilience Patterns | Snapshot/Restore 구체 패턴 (FD에서 인터페이스만 정의) |
| Performance Patterns | 상태 머신 구현 형태 (분기 dispatch 방식) |
| Logical Components | Clock·RNG 주입, KeywordPool 로딩 방식 |
| Maintainability Patterns | 검증 함수 구성, 테스트 fixture 패턴 |

**N/A**: Scalability(단일 인스턴스), Security(네트워크 표면 X), Availability SLA(직접 책임 없음)

---

## 1. 작업 체크리스트 (계획)

- [x] (1) NFR Requirements 분석 + Design 결정 항목 추출
- [x] (2) 결정 질문지 작성 (§3) — 적용 가능한 카테고리만
- [x] (3) plan에 [Answer]: 임베드
- [x] (4) 사용자 답변 수집 — Q-NFRD-U1-1~5 모두 A
- [x] (5) 모순/모호성 점검 — 일관성 확인
- [x] (6) 사용자 승인 — Generation 진입

## 2. 작업 체크리스트 (생성, 승인 후)

- [x] (7) `nfr-design-patterns.md` 작성 — 패턴 P1~P11 + Mermaid + 안티패턴 6종
- [x] (8) `logical-components.md` 작성 — LC-1~11 + 패키지 레이아웃 + 책임 매트릭스
- [x] (9) FD/NFR-Req 일관성 검증 (`logical-components.md` §4·§6)
- [x] (10) `aidlc-state.md` 갱신
- [x] (11) 사용자 승인 게이트 제시

---

## 3. 결정 질문 (5건)

> 답변: `[Answer]: <기호>` 형식. 모두 권장안으로 가시려면 "권장안"이라고만 답하셔도 됩니다.

---

### Q-NFRD-U1-1. 액션 dispatch 패턴 (Performance / Maintainability)

`Apply(action Action)`에서 액션 종류별 핸들러 분기 방식:

- A. **타입 스위치(`switch a := action.(type)`)** — Go 관용, 컴파일러 정적 검사, 디버깅 용이 (권장)
- B. **`map[reflect.Type]Handler` dispatch table** — 런타임 등록, 확장 유연성, 단 reflect 비용
- C. **메서드 dispatch** (각 Action 타입이 `Apply(state) (events, err)` 메서드 구현) — Visitor 변형. 테스트 분리 좋음, 단 도메인 타입에 행위 결합
- D. Other

[Answer]: A 

---

### Q-NFRD-U1-2. State 깊은 복사(Snapshot) 패턴 (Reliability / Performance)

`Engine.Snapshot()`는 외부에 노출되어도 안전한 State 사본을 반환해야 함 (호출자가 변형해도 엔진 내부 영향 없음).

- A. **수동 깊은 복사 함수** (`func (s State) Clone() State`) — 명시적, 빠름, 필드 추가 시 갱신 필요 (권장)
- B. **`encoding/json` Marshal → Unmarshal** 라운드트립 — 단순, 자동, 단 GC 비용 + 시간 비용 (P1<1ms 한도 위협)
- C. **`encoding/gob`** 라운드트립 — JSON보다 빠름, 단 표준 외 직렬화
- D. Other

[Answer]: A

---

### Q-NFRD-U1-3. Clock / RNG 주입 패턴 (Testability)

- A. **생성자 주입 (`game.New(assigner, clock, rng)`)** — 테스트는 fake clock·시드 reader 주입. 명시적. (권장)
- B. **package-level 변수 + override** (`var Now = time.Now`) — 간단하지만 글로벌 상태, 병렬 테스트 위험
- C. **함수 옵션 패턴** (`game.New(WithClock(...), WithRNG(...))`) — 유연하지만 본 단위에는 과한 추상화
- D. Other

[Answer]: A

---

### Q-NFRD-U1-4. KeywordPool 기본 콘텐츠 로딩 (Maintainability)

기본 키워드 풀 140개를 어떻게 코드에 임베드할까:

- A. **Go 소스의 `[]string` 상수 슬라이스로 직접 임베드** (`keyword_pool.go` 내부에 한국어 문자열 배열) — 외부 의존 0, 단순 (권장)
- B. **`//go:embed default_keywords.json`** — 파일 분리, 운영자가 외부 파일로 교체 시 동일 포맷
- C. **`//go:embed default_keywords.yaml`** — 사람이 편집하기 좋음 (단 YAML 파서 의존)
- D. Other

> 참고: A 선택이라도 **외부 파일 로딩 인터페이스**(`KeywordPool.LoadFromFile`)는 별도 함수로 제공하여 FR-7.1 외부화 가능.

[Answer]: A

---

### Q-NFRD-U1-5. 검증 함수 구성 (Maintainability)

`StartGame Options 검증`(BR-OPT-1~8) 등 다중 규칙 검증의 구현 패턴:

- A. **에러 누적(append) 후 한 번에 반환** — 호출자가 한 번에 모든 위반 사항 표시 가능. 검증 함수 모음 패턴.
- B. **첫 위반 즉시 반환** (fail-fast) — 단순, 빠름. 호스트 UI에서는 한 가지씩 수정하며 다시 시도.
- C. Other

> 참고: 호스트 UI가 N개 위반을 한 번에 보여줄 가치가 있다면 A. PoC 단순함 우선이라면 B.

[Answer]: A

---

### 자유 의견란

[Answer]: 

---

## 4. 분석 / 산출물 미리보기

답변 수령 후 §5 산출물 작성:

- **`nfr-design-patterns.md`** — 영역별 패턴(Snapshot/Restore, dispatch, Clock·RNG 주입, 검증, 테스트 fixture, 에러 표현 패턴) + Mermaid (적용 영역 다이어그램)
- **`logical-components.md`** — 본 단위의 논리적 구성요소(Engine, Apply 핸들러 그룹, Tick, RoleAssigner, KeywordPool, Clock, RNG, ValidationFns) + 책임 매트릭스
