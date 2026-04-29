# U2 Session, Persistence & Announce — Code Summary

**작성일**: 2026-04-26
**대상 단위**: U2 (`internal/session`, `internal/announce`, `internal/persistence`)
**plan**: `aidlc-docs/construction/plans/u2-session-code-generation-plan.md`

---

## 1. 빌드 / 검증 결과

| 게이트 | 결과 |
|---|---|
| `go build ./...` | ✅ 통과 |
| `go vet ./internal/...` | ✅ 0 issue |
| `gofmt -l ./internal/{session,announce,persistence}/` | ✅ empty |
| `go test ./internal/session/... ./internal/announce/... ./internal/persistence/...` | ✅ 모든 테스트 통과 |
| `go test -race ./...` | ✅ 통과 (NFR-U2-C2) |
| 라인 커버리지 (3 패키지 합산) | ✅ **86.5%** ≥ 85% (NFR-U2-M1) |
| · session | 88.1% |
| · announce | 93.3% |
| · persistence | 80.2% |
| 외부 직접 의존 | ✅ `modernc.org/sqlite v1.50.0` 1개 (NFR-U2-M6) |

> 주의: `modernc.org/sqlite v1.50.0`은 `go >= 1.25.0`을 요구하므로 본 모듈의 `go` 지시자가 1.22 → **1.25.0**으로 자동 갱신되었습니다. 다른 단위에 영향 없음 (U1 코드 호환).

---

## 2. 산출 파일 인벤토리

### 2.1 `internal/persistence/` (5 코드 + 5 테스트)
| 파일 | 책임 | LC |
|---|---|---|
| `doc.go` | 패키지 godoc | — |
| `store.go` | `PersistenceStore` 인터페이스 + `Snapshot`/`GameResult`/`PersistedMember` | LC-U2-10 |
| `schema.go` | 3 테이블 DDL + `applyPragmas` (WAL/synchronous=NORMAL) | LC-U2-12 |
| `sqlite_store.go` | `sqliteStore` 구현체 + Open/prepare/Save/Load/Close | LC-U2-11 |
| `recovery.go` | `ArchiveCorrupt` (rename + WAL/SHM 정리) | LC-U2-13 |
| `sqlite_store_test.go` | Open/0600/round-trip/원자 트랜잭션/순서 | — |
| `schema_test.go` | DDL · PRAGMA 적용 검증 | — |
| `recovery_test.go` | ArchiveCorrupt rename 동작 | — |
| `edge_test.go` | 빈 경로·다중 visibility·deep parent dir | — |
| `error_paths_test.go` | Close 후 모든 메서드 에러 surface | — |
| `json_error_test.go` | 손상 JSON 직접 주입 → unmarshal 에러 surface | — |

### 2.2 `internal/announce/` (5 코드 + 1 테스트)
| 파일 | 책임 | LC |
|---|---|---|
| `doc.go` | 패키지 godoc | — |
| `catalog.go` | `AnnouncementCatalog` 인터페이스 + `Announcement`, `Severity`, `CatalogContext` | LC-U2-7 |
| `catalog_default.go` | `defaultCatalog` Render — 모든 공개 이벤트 25종 매핑 | LC-U2-8 |
| `catalog_data.go` | 한국어 메시지 상수 + `roleKr` 헬퍼 + 시스템 토스트 (`SystemRestore`, `SystemPersistFailure`) | — |
| `error.go` | `RenderError` — EngineError 9종 + `ValidationErrors` 집계 | LC-U2-9 |
| `catalog_test.go` | 25 이벤트 + 9 에러 + 사적 이벤트 빈 결과 검증 | — |

### 2.3 `internal/session/` (8 코드 + 8 테스트)
| 파일 | 책임 | LC |
|---|---|---|
| `doc.go` | 패키지 godoc | — |
| `types.go` | `Member`, `JoinResult`, `EventOut`, `EventHandler`, `SessionOpts`, `Session` 내부 상태 | LC-U2-2/3 |
| `token.go` | 32바이트 hex 토큰 + 충돌 차단 5회 시도 | LC-U2-5 |
| `view.go` | `BuildPrivateView` 마스킹 5종 (Public/Self/Other/Mafia/Reveal) | LC-U2-6 |
| `session.go` | `SessionManager` 인터페이스 + `New` 생성자 + bootRestore + Subscribe + Close | LC-U2-1 |
| `lifecycle.go` | CreateSession / JoinPlayer / ResumePlayer / StartGame + ID 헬퍼 (`newID`) + `orderMembers` | LC-U2-1 |
| `action.go` | SubmitAction / persistAndDispatch / handleGameEnd / dispatchHandlers (panic recover) / catalogContext | LC-U2-1 |
| `tick.go` | tickLoop (1초 ticker, stopCh) + Tick | LC-U2-4 |
| `session_test.go` | fixtures (newTestManager, makeLobby, namesPool) | — |
| `lifecycle_test.go` | Create/Join/Resume 검증 + 토큰 unique + 12명 한도 + 닉네임 중복 거부 | — |
| `start_test.go` | 호스트 권한 / 인원 검증 / 이벤트 트리거 / 재시작 거부 | — |
| `action_test.go` | SubmitAction 정상·에러 + ForceEnd + Subscribe·panic 격리 | — |
| `action_more_test.go` | senderOf 모든 분기 + ForceEnd 후 finalize | — |
| `view_test.go` | PrivateView 마스킹 5종 검증 | — |
| `tick_test.go` | tickLoop graceful stop + 종료 후 Tick | — |
| `concurrency_test.go` | N=10 고루틴 동시 SubmitAction 직렬화 (NFR-U2-C1) | — |
| `restore_test.go` | reboot 후 ResumePlayer 정상 + 손상 DB archive | — |
| `restore_end_test.go` | END 단계 스냅샷 자동 finalize (BR-U2-RESTORE-6) | — |
| `edge_test.go` | New nil 가드 + 기본 SessionOpts + rng 실패 surface | — |

총 **18 Go 코드** + **14 테스트** + **본 문서 2종**.

---

## 3. 스토리/요구사항 ↔ 구현 매핑

| 요구사항 | 구현 위치 |
|---|---|
| FR-1.1 (단일 1세션) | `session.CreateSession` (in-progress 거부) |
| FR-1.2 (재연결 + 닉네임 + 토큰) | `session.JoinPlayer`, `session.ResumePlayer`, `token.go` |
| FR-4.3 (마피아 대표자) | Engine 위임 — `action.go` SubmitAction 통과 |
| FR-6.1 (결과 누적) | `sqlite_store.SaveResultAndClearActive` (트랜잭션) |
| FR-6.2 (스냅샷 영속화) | `action.persistAndDispatch` 트리거 + `sqlite_store.SaveSnapshot` |
| FR-6.3 (결과 조회) | `sqlite_store.ListResults` (ended_at DESC) |
| FR-7.2 (안내 외부화 인터페이스) | `announce.AnnouncementCatalog` 인터페이스 |
| FR-8.4 (안내 풍부) | `announce.catalog_default.Render` 25종 |
| NFR-1 (안정성·복원) | bootRestore + ArchiveCorrupt + 동기 SaveSnapshot |
| NFR-7 (외부 서비스 0) | `modernc.org/sqlite` 단일 외부 의존 |
| NFR-U2-P1~P3 | prepared stmt 캐싱 + WAL + MaxOpenConns=1 |
| NFR-U2-S1~S3 | crypto/rand 32바이트 토큰 + 0600 chmod |
| NFR-U2-S4 (마스킹) | `view.BuildPrivateView` |
| NFR-U2-C1~C3 | sync.Mutex + panic recover + go test -race 통과 |

---

## 4. 코드 위치 결정 (루트 직하 — `aidlc-docs/` 외부)

```
mafia-game/                          # ← 워크스페이스 루트
├── internal/
│   ├── game/        (U1 Game Core, 21 파일)
│   ├── persistence/ (U2 신규, 10 파일)
│   ├── announce/    (U2 신규, 6 파일)
│   └── session/     (U2 신규, 16 파일)
├── go.mod
├── go.sum
└── aidlc-docs/      (문서만, 코드 없음)
```

---

## 5. 알려진 제한 / 후속 작업

| 항목 | 상태 |
|---|---|
| `recovery.go` ArchiveCorrupt 후 새 store 자동 재오픈 | 호출자(부팅 시퀀스)가 책임. 현 구현은 rename만 수행 후 호출자가 재 open 필요 |
| events 테이블 자동 truncate 옵션 | 미구현 (옵션 디폴트 OFF, 운영 단순) |
| benchmark (NFR-U2-P1/P2 측정) | Build & Test 단계에서 추가 예정 |
| Composition Root (NewSessionManager 와이어링) | U4 HTTP Bootstrap 단계에서 작성 예정 |

---

## 6. 변경된 모듈 메타데이터

`go.mod`:
- `go 1.22` → `go 1.25.0` (modernc.org/sqlite v1.50.0 요구사항)
- 신규 직접 의존: `modernc.org/sqlite v1.50.0`
- transitive 의존 11개 추가 (`golang.org/x/sys`, `modernc.org/libc`, `modernc.org/mathutil`, `modernc.org/memory`, `github.com/dustin/go-humanize`, `github.com/google/uuid`, `github.com/mattn/go-isatty`, `github.com/ncruces/go-strftime`, `github.com/remyoudompheng/bigfft`)

> 직접 의존은 `modernc.org/sqlite` 1개로 NFR-U2-M6 만족. transitive는 sqlite 드라이버의 내부 구성 요소.
