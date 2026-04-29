# NFR Design Plan — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**대상 단위**: U2
**참조**: `nfr-requirements/*.md`, `functional-design/*.md`

---

## 0. 컨텍스트

U2의 NFR Design 결정 영역:
- Resilience: 스냅샷 실패 처리, 손상 복구, 핸들러 panic 격리
- Performance: SQLite 연결/문장 캐싱
- Concurrency: 락 분할(필요 여부)
- Logical Components: 연결 관리, 트랜잭션 래퍼, JSON 직렬화 헬퍼

**N/A**: Scalability(단일 호스트), Availability SLA, 외부 인프라 (큐·캐시 없음)

---

## 1. 작업 체크리스트 (계획)

- [x] (1) NFR Req 분석
- [x] (2) 결정 질문지 (§3)
- [x] (3) plan에 [Answer]: 임베드
- [x] (4) 답변 수집
- [x] (5) 모순/모호성 점검
- [x] (6) 사용자 승인

## 2. 작업 체크리스트 (생성, 승인 후)

- [x] (7) `nfr-design/nfr-design-patterns.md`
- [x] (8) `nfr-design/logical-components.md`
- [x] (9) FD/NFR-Req 일관성 검증
- [x] (10) `aidlc-state.md` 갱신
- [x] (11) 사용자 승인 게이트

---

## 3. 결정 질문 (5건)

> 답변: `[Answer]: <기호>`. 일괄로 "권장안"이라고만 답하셔도 됩니다.

---

### Q-NFRD-U2-1. SQLite 연결 관리

- A. **`*sql.DB`(Go의 connection pool 추상) + `MaxOpenConns=1`** — 단일 라이터 보장 (SQLite WAL은 다중 reader 지원, 본 단위는 라이터 1개로 충분). 표준 패턴. (권장)
- B. 단일 `*sql.Conn`을 직접 보유 — 연결 풀 우회
- C. `MaxOpenConns=2` — 1 writer + 1 reader 분리 (ListResults 같은 read-only 동시성 활용)
- D. Other

[Answer]: A 

---

### Q-NFRD-U2-2. Prepared statement 캐싱

- A. **모든 자주 쓰는 SQL을 `*sql.Stmt`로 prepare 후 재사용** (SaveSnapshot/SaveResult/LoadActiveSnapshot 등). `Close()`에서 정리. (권장 — 성능 +α, p99 안정성)
- B. 매 호출마다 ad-hoc 실행 (단순)
- C. Other

[Answer]: A

---

### Q-NFRD-U2-3. SessionManager → PersistenceStore 실패 격리

SaveSnapshot 실패 시:

- A. **로그 + 호스트 안내만, 게임 진행 계속** (Q-NFR-U2-9=A 재시도 없음과 일치 — 다음 PhaseChanged에서 자동 재시도) (권장)
- B. SubmitAction에서 에러 반환 → 사용자에게 즉시 통지
- C. 회로 차단(N회 연속 실패 시 게임 중단)
- D. Other

[Answer]: A

---

### Q-NFRD-U2-4. Subscribe 핸들러 panic 격리

EventHandler가 panic하면:

- A. **`defer recover()`로 격리 + 로그** — 다른 핸들러는 정상 호출, 다음 SubmitAction 정상 진행 (권장)
- B. panic을 그대로 전파 (SessionManager 종료) — 단순하지만 위험
- C. Other

[Answer]: A

---

### Q-NFRD-U2-5. JSON 직렬화 헬퍼

`State`, `Members`, `Options`, `Reveal` 등을 SQLite BLOB으로 저장할 때:

- A. **`encoding/json`(표준 라이브러리)** + 결정적 직렬화(Map 키 정렬 자동) (권장)
- B. **`gob`** — 더 빠르지만 호환성 낮음
- C. Other

[Answer]: A

---

### 자유 의견란

[Answer]: 
