# Tech Stack Decisions — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-requirements.md`, 사용자 응답 Q-NFR-U2-1~10 모두 A

---

## 1. 결정 요약

| 영역 | 결정 | 출처 | 근거 |
|---|---|---|---|
| 언어 / 버전 | Go 1.22+ (U1과 동일) | 일관성 | 단일 모듈, 동일 toolchain |
| SQLite 드라이버 | **`modernc.org/sqlite`** (순수 Go, cgo X) | Q-NFR-U2-1=A | NFR-7 운영 단순성, 빌드 환경 비종속 |
| DB 인터페이스 | `database/sql` 표준 | 표준 | 드라이버 교체 가능 |
| 동시성 모델 | `sync.Mutex` 단일 락 | Q-FD-U2-1=A | 코드 명료성, BR-CONC-1 |
| 영속화 트리거 | PhaseChanged + DeathAnnounced + Eliminated + MafiaRepresentativeReassigned + GameEnded | Q-FD-U2-2=A | NFR-1 + 단일 락 부하 균형 |
| 자동 복원 | 부팅 시 즉시 | Q-FD-U2-3=A | NFR-1 |
| 토큰 발급 | 32바이트 `crypto/rand` → hex(64) | Q-NFR-U2-6=A | 256-bit 엔트로피 |
| 백그라운드 ticker | SessionManager 보유 1초 ticker | Q-FD-U2-5=A | 외부 와이어링 단순 |
| Tick 실행 | `tickLoop` 고루틴 + `stopCh` graceful stop | Q-FD-U2-5=A | Close에서 안전 종료 |
| 안내 카탈로그 | Go 함수 + `AnnouncementCatalog` 인터페이스 추상 | Q-FD-U2-7=A, NFR-U2-M4 | FR-7.2 외부화 가능 |
| 에러 매핑 | U2 백엔드에서 Korean 메시지 생성 | Q-FD-U2-6=A | 일관된 사용자 경험 |
| SQLite 모드 | `journal_mode=WAL`, `synchronous=NORMAL` | NFR-1·NFR-2 균형 | 문서 §6.2 |
| DB 파일 위치 | `./data/mafia.db` + `MAFIA_DB_PATH` env 오버라이드 | Q-FD-U2-10=A | 운영 단순 + 유연 |
| DB 파일 권한 | `0600` (소유자만 RW) | Q-NFR-U2-7=A | 비공개 정보 보호 |
| 손상 스냅샷 | rename `mafia.db.corrupt-{ts}` + 새 LOBBY | Q-NFR-U2-8=A | 데이터 보존 + 진행 보장 |
| 테스트 SQLite | 임시 파일 (`t.TempDir()`) | Q-NFR-U2-5=A | WAL/PRAGMA 실제 동작 검증 |
| 테스트 종류 | 테이블 드리븐 + 시나리오 + 동시성 | Q-NFR-U2-10=A | 다중 컴포넌트 검증 |
| 커버리지 목표 | 라인 ≥ 85% | Q-NFR-U2-4=A | 인프라 비중 고려 |

---

## 2. 의존성 트리 (예상)

```
internal/session/      → modernc.org/sqlite (database/sql 통해)
internal/announce/     → 표준 라이브러리만
internal/persistence/  → modernc.org/sqlite, database/sql, encoding/json

표준 라이브러리 사용:
- context
- crypto/rand
- database/sql
- encoding/hex, encoding/json
- errors, fmt, io
- log/slog (또는 log)
- os, path/filepath
- sync, sync/atomic
- time
```

> 외부 모듈은 `modernc.org/sqlite` 단 1개. `go.mod`의 `require` 블록에 추가 (transitive: `modernc.org/libc`, `modernc.org/mathutil` 등 — 모두 modernc 그룹).

---

## 3. 패키지 / 파일 레이아웃 (Code Generation 인풋)

### 3.1 `internal/session/`
```
internal/session/
├── doc.go                    # godoc
├── session.go                # SessionManager 인터페이스 + session struct + New
├── lifecycle.go              # CreateSession, JoinPlayer, ResumePlayer, StartGame, Close
├── action.go                 # SubmitAction + persistAndDispatch
├── tick.go                   # tickLoop, Tick
├── view.go                   # PrivateView 빌더 (마스킹 5종)
├── token.go                  # 32-byte crypto/rand 토큰 생성
├── types.go                  # Session, Member, JoinResult, EventOut 등
└── *_test.go
```

### 3.2 `internal/announce/`
```
internal/announce/
├── doc.go
├── catalog.go                # AnnouncementCatalog 인터페이스 + Announcement, Severity
├── catalog_default.go        # defaultCatalog: Render(env) 매핑 함수
├── catalog_data.go           # 한국어 메시지 상수/템플릿 (25개 + 에러 9개)
├── error_catalog.go          # ErrorAnnouncement 매핑
└── *_test.go
```

### 3.3 `internal/persistence/`
```
internal/persistence/
├── doc.go
├── store.go                  # PersistenceStore 인터페이스 + types (Snapshot, GameResult)
├── sqlite_store.go           # sqliteStore 구현체 + SQL 트랜잭션
├── schema.go                 # CREATE TABLE DDL + PRAGMA 적용 함수
├── recovery.go               # 손상 스냅샷 archive (rename) 처리
└── *_test.go
```

> 정확한 파일 분할은 Code Generation 단계에서 미세 조정. 본 문서는 의도된 책임 분리 명시.

---

## 4. 모듈 스코프 결정

### 4.1 Logging
- `log/slog` (Go 1.21+) 사용. 운영자가 환경변수 `MAFIA_LOG_LEVEL`로 조정.
- 단위 테스트는 `slog.New(slog.NewTextHandler(io.Discard, ...))` 또는 `*testing.T.Log` 어댑터로 묵음.

### 4.2 Context propagation
- 모든 PersistenceStore 메서드는 `ctx context.Context` 첫 인자로 받음 (cancellation 지원).
- SessionManager 공개 메서드도 `ctx context.Context` 첫 인자.

### 4.3 시간 / Clock
- U2도 U1과 동일한 `game.Clock` 인터페이스 사용 — Composition Root에서 동일 인스턴스 주입.
- 테스트는 `*game.FakeClock` 사용.

### 4.4 RNG (토큰 생성)
- 운영: `crypto/rand.Reader` 직접 사용 (256-bit 엔트로피)
- 테스트: 테스트용 토큰 발급 함수 별도 (랜덤 의존 테스트 회피)

---

## 5. 비-요구사항 / 향후 고려

| 항목 | 현재 | 향후 |
|---|---|---|
| 다중 게임 동시 진행 | 미지원 | session 인스턴스 분리 + DB schema 확장 (현재 Non-Goal) |
| 클라우드 백업 | 미지원 | active_snapshot/game_results 외부 export |
| 다국어 안내 | 한국어 only | AnnouncementCatalog 인터페이스로 교체 가능 (FR-7.2) |
| 암호화 | 평문 BLOB | 추가 라이브러리 필요 |

---

## 6. 결정 검증 체크리스트

- [x] 모든 결정이 Q-FD-U2 / Q-NFR-U2 사용자 응답과 일치
- [x] FR-6 (영속화)·FR-7.2 (외부화)·FR-8.4 (안내 풍부)와 충돌 없음
- [x] U1 의존(`game.PlayerID`, `game.State`, `game.Engine` 등) 정확히 명시
- [x] 외부 의존 1개(`modernc.org/sqlite`)만 추가 — NFR-7 충돌 없음
- [x] 테스트 환경(임시 파일) 결정과 단위 테스트 작성 가능성 일치
