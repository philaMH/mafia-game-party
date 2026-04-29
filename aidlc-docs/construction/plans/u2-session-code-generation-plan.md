# Code Generation Plan — U2 Session, Persistence & Announce

**작성일**: 2026-04-26
**대상 단위**: U2 (`internal/session/*`, `internal/announce/*`, `internal/persistence/*`)
**참조**:
- `application-design/unit-of-work.md` §2
- `construction/u2-session-persistence-announce/functional-design/*.md`
- `construction/u2-session-persistence-announce/nfr-requirements/*.md`
- `construction/u2-session-persistence-announce/nfr-design/*.md`
- `aidlc-state.md` (Workspace Root: `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game`)
- U1 공개 API: `construction/u1-game-core/code/u1-public-api.md`

> 본 plan은 U2 Code Generation의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트

**책임**: 단일 GM 락으로 게임 세션 직렬 처리, SQLite 영속화, 도메인 이벤트 → 한국어 안내 메시지 변환.

**구현 대상 요구사항** (story map §4 U2 Primary):
- FR-1.1, FR-1.2, FR-4.3, FR-6.1, FR-6.2, FR-6.3, FR-7.2, FR-8.4
- NFR-1, NFR-7

**의존**:
- **U1 Game Core** (`internal/game`) — 도메인 타입·Engine 인터페이스 import
- **외부**: `modernc.org/sqlite` (순수 Go SQLite 드라이버) — 신규 추가

**산출물**: Go 패키지 3개 (`internal/session`, `internal/announce`, `internal/persistence`) + 단위 테스트.

---

## 1. 코드 위치 결정

| 항목 | 위치 |
|---|---|
| Workspace Root | `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game` |
| `internal/session/` | SessionManager |
| `internal/announce/` | AnnouncementCatalog + 카탈로그 데이터 |
| `internal/persistence/` | SQLite store + 스키마 + recovery |
| 문서 산출물 | `aidlc-docs/construction/u2-session-persistence-announce/code/` (markdown 요약) |

---

## 2. Part 1 — Planning 체크리스트

- [x] (P1-1) 단위 컨텍스트 분석
- [x] (P1-2) 코드 위치·구조 결정
- [x] (P1-3) plan 문서 작성
- [x] (P1-4) 사용자에게 요약 제공
- [x] (P1-5) audit에 승인 게이트 로그
- [x] (P1-6) 사용자 승인
- [x] (P1-7) Part 2 진입

---

## 3. Part 2 — Generation 체크리스트

### 3.1 모듈 의존성 추가
- [x] (G1) `go get modernc.org/sqlite@latest` → `go.mod` / `go.sum` 갱신 (go 1.22 → 1.25.0 자동 갱신됨)

### 3.2 `internal/persistence/`
- [x] (G2) `internal/persistence/doc.go` — 패키지 godoc
- [x] (G3) `internal/persistence/store.go` — `PersistenceStore` 인터페이스 + `Snapshot`, `GameResult`, `PersistedMember` 타입 (LC-U2-10)
- [x] (G4) `internal/persistence/schema.go` — DDL 3 테이블 + `applyPragmas` (WAL, synchronous=NORMAL) (P-U2-8)
- [x] (G5) `internal/persistence/sqlite_store.go` — `sqliteStore` 구현체 + Open/prepare/Close + 모든 메서드 (P-U2-1, P-U2-2, P-U2-5, NFR-U2-S3 chmod 0600)
- [x] (G6) `internal/persistence/recovery.go` — `ArchiveCorrupt`(rename) (P-U2-9)

### 3.3 `internal/announce/`
- [x] (G7) `internal/announce/doc.go`
- [x] (G8) `internal/announce/catalog.go` — `AnnouncementCatalog` 인터페이스 + `Announcement`, `Severity`, `CatalogContext`
- [x] (G9) `internal/announce/catalog_default.go` — `defaultCatalog` Render 함수 (이벤트 25종 매핑)
- [x] (G10) `internal/announce/catalog_data.go` — 한국어 메시지 상수 + 보간 헬퍼 (`roleKr`, etc.)
- [x] (G11) `internal/announce/error.go` — `RenderError(err, sender)` 매핑 (EngineError 9종 → 한국어)

### 3.4 `internal/session/`
- [x] (G12) `internal/session/doc.go`
- [x] (G13) `internal/session/types.go` — `Session`, `Member`, `JoinResult`, `EventOut`, `EventHandler`, `SessionOpts`
- [x] (G14) `internal/session/token.go` — `newToken` (32-byte crypto/rand → hex64) + `issueUniqueToken` (충돌 차단 5회 시도)
- [x] (G15) `internal/session/view.go` — `BuildPrivateView` (마스킹 5종)
- [x] (G16) `internal/session/session.go` — `SessionManager` 인터페이스 + `session` struct + `New` 생성자 + `Subscribe`/`Close`
- [x] (G17) `internal/session/lifecycle.go` — `CreateSession`, `JoinPlayer`, `ResumePlayer`, `StartGame` + `newID` (crypto/rand 기반)
- [x] (G18) `internal/session/action.go` — `SubmitAction`, `persistAndDispatch`, `dispatchHandlers` (panic recover) + `handleGameEnd` + `buildResultFromState`
- [x] (G19) `internal/session/tick.go` — `tickLoop`, `Tick`

### 3.5 단위 테스트 — `internal/persistence/`
- [x] (G20) `internal/persistence/sqlite_store_test.go` — Open/Close, 0600 권한 검증, Save/Load 라운드트립, SaveResult+Clear 트랜잭션, ListResults 정렬
- [x] (G21) `internal/persistence/recovery_test.go` — ArchiveCorrupt rename 동작
- [x] (G22) `internal/persistence/schema_test.go` — DDL 적용 + PRAGMA 검증
- [x] (추가) `edge_test.go`, `error_paths_test.go`, `json_error_test.go` — 커버리지 보강

### 3.6 단위 테스트 — `internal/announce/`
- [x] (G23) `internal/announce/catalog_test.go` — 25개 이벤트 + 9개 에러 매핑 검증, 비공개 이벤트는 빈 Announcement 반환

### 3.7 단위 테스트 — `internal/session/`
- [x] (G24) `internal/session/session_test.go` — fixtures (newTestManager, makeLobby, namesPool)
- [x] (G25) `internal/session/lifecycle_test.go` — CreateSession/JoinPlayer/ResumePlayer 검증, 토큰 unique, 12명 한도, 이름 중복 거부
- [x] (G26) `internal/session/start_test.go` — StartGame 호스트 권한/인원 검증, 이벤트 트리거
- [x] (G27) `internal/session/action_test.go` — SubmitAction 정상/에러, ForceEnd, Subscribe panic 격리
- [x] (G28) `internal/session/view_test.go` — PrivateView 마스킹 5종
- [x] (G29) `internal/session/tick_test.go` — tickLoop graceful stop, Tick 락 획득
- [x] (G30) `internal/session/concurrency_test.go` — N=10 고루틴 동시 SubmitAction (NFR-U2-C1)
- [x] (G31) `internal/session/restore_test.go` + `restore_end_test.go` — 자동 복원 + END 단계 finalize + 손상 archive
- [x] (추가) `edge_test.go`, `action_more_test.go` — 커버리지 보강

### 3.8 문서 산출물
- [x] (G32) `aidlc-docs/construction/u2-session-persistence-announce/code/u2-code-summary.md`
- [x] (G33) `aidlc-docs/construction/u2-session-persistence-announce/code/u2-public-api.md`

### 3.9 N/A 단계
- [x] (G34) Deployment Artifacts — N/A (단일 바이너리에 통합)
- [x] (G35) DB Migration Scripts — schema.go에 통합 (별도 마이그레이션 도구 없음)
- [x] (G36) Frontend Components — N/A (백엔드 단위)

---

## 4. Definition of Done

- [x] (V1) 모든 G1~G36 [x]
- [x] (V2) `go build ./...` 통과
- [x] (V3) `go vet ./internal/session/... ./internal/announce/... ./internal/persistence/...` 0 issue
- [x] (V4) `gofmt -l ./internal/session/ ./internal/announce/ ./internal/persistence/` empty
- [x] (V5) `go test ./internal/session/... ./internal/announce/... ./internal/persistence/...` 모든 테스트 통과
- [x] (V6) `go test -race` 통과 (NFR-U2-C2)
- [x] (V7) `go test -cover` 라인 **86.5%** ≥ 85% (NFR-U2-M1) — session 88.1% / announce 93.3% / persistence 80.2%
- [x] (V8) `go.mod` 직접 의존 1개(`modernc.org/sqlite v1.50.0`) — transitive는 sqlite 드라이버 내부 구성요소

---

## 5. 스토리 추적성

| 요구사항 | 구현 단계 |
|---|---|
| FR-1.1 (단일 1세션 + LAN URL) | G16, G17 (CreateSession) |
| FR-1.2 (재연결 + 닉네임) | G14 (token), G17 (Resume) |
| FR-4.3 (마피아 대표자 권한) | Engine 위임 — G18 (SubmitAction) |
| FR-6.1 (결과 누적) | G3, G5 (SaveResultAndClearActive) |
| FR-6.2 (스냅샷 영속화) | G5 (SaveSnapshot), G18 (트리거) |
| FR-6.3 (결과 조회) | G5 (ListResults) |
| FR-7.2 (안내 외부화 인터페이스) | G8 (AnnouncementCatalog) |
| FR-8.4 (안내 풍부 카탈로그) | G9, G10 |
| NFR-1 (안정성·복원) | G5, G6, G18, G31 |
| NFR-7 (외부 서비스 0) | G1 (단일 외부 lib) |
| NFR-U2-P1~P3 (성능) | G5 (prepared stmts) |
| NFR-U2-S1~S3 (보안) | G14 (토큰), G5 (chmod 0600) |
| NFR-U2-S4 (마스킹) | G15 (PrivateView) |
| NFR-U2-C1~C3 (동시성) | G16 (mutex), G18 (panic recover), G30 (race test) |

---

## 6. 산출물 요약 (예상)

| 종류 | 파일 수 | 위치 |
|---|---:|---|
| persistence 코드 | 5 | `internal/persistence/*.go` |
| announce 코드 | 5 | `internal/announce/*.go` |
| session 코드 | 8 | `internal/session/*.go` |
| 단위 테스트 | 12 | `internal/*/[file]_test.go` |
| 문서 요약 | 2 | `aidlc-docs/construction/u2-.../code/*.md` |

---

## 7. 사용자 승인 게이트

본 plan에 동의하시면 **"승인"** 또는 **"continue"** 로 답변. 변경이 필요하면 구체적 항목을 알려주세요 (예: "G31에 동시 ResumePlayer 테스트 추가").
