# NFR Requirements Plan — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**대상 단위**: U2 (`internal/session/*`, `internal/announce/*`, `internal/persistence/*`)
**참조**: `requirements.md` v1.1 NFR-1/2/4/6/7, `construction/u2-session-persistence-announce/functional-design/*.md`

---

## 0. U2 NFR 컨텍스트

U2는 **단일 GM 락 + 동기 SQLite 영속화**라서 NFR 우선순위는:

| NFR 영역 | U2 우선순위 | 근거 |
|---|---|---|
| **Reliability / Durability** | 🔴 최우선 | NFR-1, 스냅샷 안정성·자동 복원이 단위 가치의 핵심 |
| **Performance** | 🟠 중요 | 단일 락 + 동기 SaveSnapshot이 단계 전이 지연을 결정 (SLA 필요) |
| **Maintainability** | 🟠 중요 | NFR-6, 백엔드 단위 — 단위 테스트 필요 |
| **Security** | 🟡 보통 | 사내 LAN + 비공개 정보 라우팅. 토큰 엔트로피 / DB 파일 권한 정도 |
| **Storage / Growth** | 🟢 낮음 | SQLite 단일 파일 + 게임 결과 누적. PoC 규모 |
| **Scalability / Availability** | 🟢 N/A | 단일 호스트 PC, 단일 세션 |

본 단계 핵심 결정: 영속화 SLA, 락 contention 한도, 토큰 엔트로피, 파일 권한, 테스트 환경(메모리 SQLite vs 임시 파일), 커버리지 목표.

---

## 1. 작업 체크리스트 (계획)

- [x] (1) FD 산출물 분석 + NFR 영역 우선순위 매핑
- [x] (2) 결정 질문지 작성 (§3)
- [x] (3) plan에 [Answer]: 임베드
- [x] (4) 사용자 답변 수집
- [x] (5) 모순/모호성 점검
- [x] (6) 사용자 승인

## 2. 작업 체크리스트 (생성, 승인 후)

- [x] (7) `nfr-requirements/nfr-requirements.md` 작성
- [x] (8) `nfr-requirements/tech-stack-decisions.md` 작성
- [x] (9) FR/NFR 추적성 검증
- [x] (10) `aidlc-state.md` 갱신
- [x] (11) 사용자 승인 게이트

---

## 3. 결정 질문 (사용자 답변 필요)

> 답변: `[Answer]: <기호>` 또는 자유 의견. "권장안"으로 일괄 답변 가능.

---

### Q-NFR-U2-1. 외부 의존성 정책

- A. **`modernc.org/sqlite`만 추가** (순수 Go SQLite 드라이버, cgo 불필요) — `database/sql` 표준 인터페이스. 다른 외부 lib 0개 (권장)
- B. `mattn/go-sqlite3` 사용 (cgo 필요, 더 빠름, 다만 빌드 환경 복잡)
- C. Other (자유의견)

[Answer]: A 

---

### Q-NFR-U2-2. SaveSnapshot 동기 호출 지연 한도

PhaseChanged 직후 SaveSnapshot이 락을 잡고 동기 실행됨. 이 시간이 곧 단계 전이 지연.

- A. **p99 < 50 ms** (12명, WAL 모드 기준) (권장 — 인간 인지 임계 100ms 이내)
- B. p99 < 100 ms (느슨)
- C. 측정만 하고 명시적 한도 없음
- D. Other

[Answer]: A

---

### Q-NFR-U2-3. SubmitAction 전체 처리 지연

`mu.Lock → Engine.Apply → catalog.Render → SaveSnapshot(if trigger) → handlers → Unlock` 전체:

- A. **p99 < 100 ms** (12명, 일반 케이스) (권장)
- B. p99 < 200 ms
- C. Other

[Answer]: A

---

### Q-NFR-U2-4. 단위 테스트 커버리지 목표

U2는 다중 컴포넌트(C3+C4+C5)라 코드량 多. 합리적 목표:

- A. **라인 ≥ 85%** (도메인보다 인프라 비중 높음 — 외부 I/O 모킹 한계 고려) (권장)
- B. 라인 ≥ 90% (U1과 동일)
- C. 라인 ≥ 80% (현실적 하한)
- D. Other (자유의견에 수치)

[Answer]: A

---

### Q-NFR-U2-5. 테스트 SQLite 환경

- A. **임시 파일 SQLite** (`t.TempDir()` + `?cache=shared&mode=rwc`) — 실제 동작 검증 (권장)
- B. 메모리 SQLite (`:memory:`) — 빠르지만 WAL 모드 제약, 멀티 connection 이슈
- C. 둘 다 (병행 테스트)
- D. Other

[Answer]: A

---

### Q-NFR-U2-6. 토큰 엔트로피

- A. **32바이트(256비트) `crypto/rand` → 64 hex char** (권장 — 추측 불가)
- B. 16바이트(128비트)
- C. Other

[Answer]: A

---

### Q-NFR-U2-7. DB 파일 권한 (NFR-4 보조)

- A. **0600** (소유자만 읽기/쓰기) — 호스트 PC 단일 사용자 가정 (권장)
- B. 0644 (소유자 RW + 그룹/기타 R)
- C. Other

[Answer]: A

---

### Q-NFR-U2-8. 손상 스냅샷 처리 (BR-U2-RESTORE-3)

자동 복원 실패 시:

- A. **손상 파일을 `mafia.db.corrupt-{timestamp}`로 rename + 새 LOBBY로 시작** (권장 — 데이터 보존 + 게임 진행 보장)
- B. 손상 파일 즉시 삭제 + 새 LOBBY로 시작
- C. 호스트에게 수동 복구 요구 (게임 진행 차단)
- D. Other

[Answer]: A

---

### Q-NFR-U2-9. SaveSnapshot 실패 시 재시도

- A. **재시도 없음**. 다음 PhaseChanged에서 자동 재시도 (스냅샷이 누적되지 않으므로 매번 최신 상태 저장 — 권장 단순)
- B. 즉시 1회 재시도 후 실패 시 ERROR 안내
- C. 백오프 재시도 (예: 100ms, 500ms, 1s)
- D. Other

[Answer]: A

---

### Q-NFR-U2-10. 동시성 안전성 검증

- A. **`go test -race`로 race-free 검증** + 단위 테스트에서 N개 고루틴이 동시에 SubmitAction 호출 → 결과 직렬화 검증 (권장)
- B. race 검증만, 동시 호출 테스트 생략
- C. Other

[Answer]: A

---

### 자유 의견란

[Answer]: 
