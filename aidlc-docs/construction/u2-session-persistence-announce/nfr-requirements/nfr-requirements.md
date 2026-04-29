# NFR Requirements — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 NFR-1/2/4/6/7, `construction/u2-session-persistence-announce/functional-design/*.md`, `plans/u2-session-nfr-requirements-plan.md`

---

## 1. NFR 영역별 요구사항

### 1.1 Reliability / Durability (NFR-1 — 최우선)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U2-R1 | 동기 SaveSnapshot이 PhaseChanged + 사망 이벤트 직후 반드시 실행 | 100% (단위 테스트로 모든 트리거 검증) | 테이블 드리븐 테스트 |
| NFR-U2-R2 | 자동 복원: 부팅 시 active_snapshot 발견 시 정확히 동일 단계로 복원 | Snapshot/Restore 의미 동등성 | 시나리오 테스트 (5종) |
| NFR-U2-R3 | 손상 스냅샷 처리: rename + 새 LOBBY (Q-NFR-U2-8=A) | 손상 시 게임 진행 보장 | 단위 테스트 |
| NFR-U2-R4 | SaveResult + DeleteActiveSnapshot 원자성 (트랜잭션) | GameEnded 이벤트 후 둘 중 하나만 적용된 상태 0 | 통합 테스트 |
| NFR-U2-R5 | 토큰 식별 정확성 | 잘못된 토큰 → ErrUnknownPlayer, 정확한 토큰 → 동일 PlayerID 복원 | 단위 테스트 |
| NFR-U2-R6 | tickLoop graceful shutdown | Close 호출 후 ticker 고루틴이 1초 이내 종료 | 단위 테스트 (timeout) |
| NFR-U2-R7 | SQLite WAL + synchronous=NORMAL 적용 | 부팅 시 PRAGMA 확인 가능 | DB 검증 테스트 |

### 1.2 Performance (NFR-2)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U2-P1 | SaveSnapshot 동기 호출 지연 (12명 기준, WAL) | **p99 < 50 ms** | Go benchmark |
| NFR-U2-P2 | SubmitAction 전체 처리 지연 (Lock~Unlock, 12명) | **p99 < 100 ms** | Go benchmark |
| NFR-U2-P3 | LoadActiveSnapshot 지연 | p99 < 50 ms | Go benchmark |
| NFR-U2-P4 | Tick 1초 ticker가 Lock 획득 못해 누락 | **누락 < 5%** (12명, 정상 경로) | 통합 테스트 |
| NFR-U2-P5 | tickLoop가 락을 1초 이상 점유하지 않음 | Tick 1회 처리 시간 < 100 ms | 측정 |

### 1.3 Maintainability (NFR-6)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U2-M1 | 단위 테스트 라인 커버리지 | **≥ 85%** (`internal/session`, `internal/announce`, `internal/persistence`) | `go test -cover` |
| NFR-U2-M2 | 정적 분석 통과 | `go vet`, `golangci-lint`(default), `gofmt -l → empty` | CI 게이트 |
| NFR-U2-M3 | godoc 주석 — 모든 공개 식별자 | SessionManager, PersistenceStore, AnnouncementCatalog, Member, Snapshot, GameResult 등 | revive 룰 |
| NFR-U2-M4 | AnnouncementCatalog 인터페이스 추상화 (FR-7.2) | 인터페이스 + default 구현체 분리 | 코드 리뷰 |
| NFR-U2-M5 | PersistenceStore 인터페이스 추상화 | 인터페이스 + sqliteStore 구현체. 테스트 가능한 mock 가능 | 코드 리뷰 |
| NFR-U2-M6 | 외부 의존성 한정 | `modernc.org/sqlite`만 (Q-NFR-U2-1=A) | `go list -m all` |

### 1.4 Security (NFR-4)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U2-S1 | 토큰 엔트로피 | 32바이트 (256-bit) `crypto/rand` (Q-NFR-U2-6=A) | 코드 리뷰 + 단위 테스트 |
| NFR-U2-S2 | 토큰 충돌 차단 | 같은 게임 내 중복 0 (재발급 정책) | 단위 테스트 (반복 발급 후 unique 검증) |
| NFR-U2-S3 | DB 파일 권한 | **0600** (소유자만 RW, Q-NFR-U2-7=A) | 단위 테스트 (생성 후 stat 확인) |
| NFR-U2-S4 | 비공개 정보 마스킹 (PrivateView) | PublicView 송신 시 Players[*].Role/Keyword == "" | 단위 테스트 5종 |

### 1.5 Storage / Growth

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U2-G1 | active_snapshot row 수 | 항상 ≤ 1 | DB invariant 테스트 |
| NFR-U2-G2 | game_results 누적 | game_id PRIMARY KEY 중복 차단 | 단위 테스트 |
| NFR-U2-G3 | events 테이블 (옵션) | EventLog OFF 기본값. 1게임 종료 시 (옵션) Truncate | 코드 리뷰 |
| NFR-U2-G4 | DB 파일 크기 한도 | 1년 운영 100게임 가정 < 10 MB (예상) | 운영 모니터링 (Build & Test 단계) |

### 1.6 Concurrency

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U2-C1 | 단일 GM 락 직렬화 | N개 고루틴 동시 SubmitAction → 결과가 직렬 적용된 것과 동등 | 동시성 단위 테스트 (Q-NFR-U2-10=A) |
| NFR-U2-C2 | 데이터 레이스 미발생 | `go test -race` 통과 | CI 게이트 |
| NFR-U2-C3 | tickLoop deadlock 차단 | Subscribe 핸들러 내 락 재획득 금지 (godoc 명시) | 코드 리뷰 |

---

## 2. 트레이드오프 결정

| 트레이드오프 | 본 단위의 결정 | 근거 |
|---|---|---|
| 동기 영속화 vs 비동기 큐 | **동기** (Q-FD-U2-2=A) | NFR-1 안정성 — 손실 위험 < 지연 감소 |
| 단순 mutex vs actor 채널 | **단순 mutex** (Q-FD-U2-1=A) | 코드 명료성 + PoC 규모 |
| modernc.org/sqlite vs cgo SQLite | **modernc** | 외부 빌드 환경 비종속, NFR-7 운영 단순성 |
| 메모리 vs 임시 파일 SQLite (테스트) | **임시 파일** | 실제 WAL/PRAGMA 동작 검증 |
| 토큰 16바이트 vs 32바이트 | **32바이트** | 보안 마진 (오버헤드 무시 가능) |

---

## 3. 추적성 (FR/NFR ↔ U2 NFR)

| 출처 | 본 문서 항목 |
|---|---|
| NFR-1 (안정성·복원) | NFR-U2-R1~R7 |
| NFR-2 (성능) | NFR-U2-P1~P5 |
| NFR-4 (비공개·LAN 한정) | NFR-U2-S3, NFR-U2-S4 |
| NFR-6 (도메인 분리·유지보수성) | NFR-U2-M1~M6 |
| NFR-7 (외부 서비스 0) | NFR-U2-M6 (외부 lib 1개), NFR-U2-G1~G2 (단일 SQLite 파일) |
| FR-6.1/6.2/6.3 (영속화·조회) | NFR-U2-R1, R4, G1, G2 |
| FR-7.2 (안내 외부화) | NFR-U2-M4 |
| FR-1.2 (재연결) | NFR-U2-R5, NFR-U2-S1, S2 |

---

## 4. 검증 게이트 (Build & Test 단계)

다음 모든 항목이 통과해야 U2가 출하 가능:
1. ✅ `go vet ./internal/session/... ./internal/announce/... ./internal/persistence/...` 0 issue
2. ✅ `gofmt -l ./internal/session/ ./internal/announce/ ./internal/persistence/` == empty
3. ✅ `golangci-lint run` (default set) 0 issue
4. ✅ `go test -race ./internal/session/... ./internal/announce/... ./internal/persistence/...` 통과
5. ✅ `go test -coverprofile=...` 라인 ≥ 85% (NFR-U2-M1)
6. ✅ `go test -bench=.` SaveSnapshot p99 < 50 ms, SubmitAction p99 < 100 ms (NFR-U2-P1, P2)
7. ✅ DB 파일 권한 0600 검증 단위 테스트 통과 (NFR-U2-S3)
8. ✅ 동시성 단위 테스트 (N=10 고루틴 동시 SubmitAction) 직렬화 통과 (NFR-U2-C1)

---

## 5. 명시적 비-요구사항 (Non-Goals)

- **Scalability**: 단일 호스트 PC, 단일 게임. 멀티 인스턴스 / sharding 미지원.
- **Availability SLA**: 직접 책임 없음. NFR-1로 정상 종료 시 재기동 안정성만 보장.
- **암호화 저장**: PoC 단순성 (Q-FD-U2-12=A). 사내 LAN + 호스트 PC 단일 사용자 가정.
- **다국어 안내**: 한국어 한정 (FR-8.3).
- **DB 백업/복구 도구**: PoC 범위 외. 운영자가 `data/mafia.db` 파일 복사로 수동 백업.
- **rate limiting**: 단일 LAN 환경, 신뢰된 클라이언트 가정. U3 전송 단위에서 ping/pong 하트비트만.
