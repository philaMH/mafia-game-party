# U4 HTTP Bootstrap & Static — Code Summary

**작성일**: 2026-04-26
**대상 단위**: U4 (`cmd/mafia-game`, `internal/transport/http`)
**plan**: `aidlc-docs/construction/plans/u4-http-bootstrap-code-generation-plan.md`

---

## 1. 빌드 / 검증 결과

| 게이트 | 결과 |
|---|---|
| `go build ./cmd/mafia-game` | ✅ 단일 바이너리 산출 |
| `go vet ./...` | ✅ 0 issue |
| `gofmt -l ./internal/transport/http/ ./cmd/mafia-game/` | ✅ empty |
| `go test ./internal/transport/http/...` | ✅ 모든 테스트 통과 |
| `go test -race ./...` | ✅ 통과 |
| 라인 커버리지 (5 패키지 합산) | ✅ **87.6%** ≥ 85% (NFR-U4-M1) |
| · http (httpx) | 89.2% |
| · ws | 89.0% |
| · session | 88.2% |
| · announce | 93.3% |
| · persistence | 80.2% |
| 외부 직접 의존 신규 | ✅ **0개** (NFR-U4-M4) |

---

## 2. 산출 파일 인벤토리

### 2.1 `internal/transport/http/` (6 코드 + 6 테스트)

| 파일 | 책임 | LC |
|---|---|---|
| `doc.go` | 패키지 godoc (alias 'httpx') | — |
| `server.go` | `Server` 인터페이스 + `Config` + `New` + http.Server 타임아웃 (P-U4-1) | LC-U4-2/3/4 |
| `middleware.go` | `loggingMiddleware` + `statusRecorder` (4필드, 페이로드 미기록) | LC-U4-5 |
| `routes.go` | `buildMux` + `healthHandler` + `assetsHandler` + `spaHandler` | LC-U4-6/8/9 |
| `api_results.go` | `resultsHandler` + `buildResultsResponse` (Token 의도적 제외) | LC-U4-7 |
| `lan.go` | `PrintLANAddresses` (RFC1918 IPv4 only) | LC-U4-10 |
| `server_test.go` | New 검증, /healthz/assets/SPA/api 동작, Token 미포함 검증 | — |
| `middleware_test.go` | statusRecorder + loggingMiddleware 4필드 + 페이로드 미기록 검증 | — |
| `routes_test.go` | spaHandler/assetsHandler/healthHandler + buildMux 라우트 등록 | — |
| `api_results_test.go` | buildResultsResponse Token 제거 + limit 경계값 7종 + DB 에러 | — |
| `lan_test.go` | PrintLANAddresses 출력 형식 + 포트 표기 | — |
| `integration_test.go` | 실제 net.Listen + ListenAndServe + Shutdown < 5초 | — |

### 2.2 `cmd/mafia-game/` (1 코드 + placeholder)

| 파일 | 책임 |
|---|---|
| `main.go` | Composition Root: flag/env 파싱 + 단위 와이어링 + signal.NotifyContext + graceful shutdown 3단계 |
| `web/dist/index.html` | placeholder (embed 빌드 보장) |

> **위치 변경 메모**: 디자인 문서에서는 `web/dist`를 워크스페이스 루트에 두기로 했으나, Go의 `//go:embed`가 상위 경로(`..`)를 허용하지 않아 main.go의 형제 디렉터리(`cmd/mafia-game/web/dist/`)로 이동. U5에서 Vite 빌드 시 `outDir`을 `../cmd/mafia-game/web/dist`로 지정해 동일 위치를 유지하도록 함.

총 **6 http 코드** + **6 테스트** + **1 main.go** + **1 placeholder** + **2 문서**.

---

## 3. 스토리/요구사항 ↔ 구현 매핑

| 요구사항 | 구현 위치 |
|---|---|
| FR-1.1 (LAN URL 출력) | `lan.go:PrintLANAddresses` + `main.go` 시작 시 호출 |
| FR-6.3 (결과 조회 API) | `api_results.go:resultsHandler` + `buildResultsResponse` |
| NFR-1 (graceful shutdown) | `main.go:shutdown` 3단계 (HTTP 5s + Hub.Close + mgr 2s) |
| NFR-4 (비공개 정보) | `api_results.go:memberEntry` (Token 필드 부재) + `middleware.go` (4필드만 로그) |
| NFR-7 (외부 의존 0) | 모든 코드가 표준 lib만 사용 |
| NFR-U4-S1 (Token 미포함) | `TestServer_APIResultsOmitsMemberToken` + `TestBuildResultsResponse_StripsToken` |
| NFR-U4-R4 (graceful shutdown < 7초) | `TestIntegration_ListenAndShutdown` |
| NFR-U4-B3 (placeholder 보장) | `cmd/mafia-game/web/dist/index.html` commit |

---

## 4. 핵심 설계 결정 (재확인)

| 결정 | 위치 |
|---|---|
| http.Server 타임아웃 (Read 30s / Write 0 / Idle 60s) (P-U4-1) | `server.go` |
| signal.NotifyContext 통합 종료 (P-U4-4) | `main.go:run` |
| immutable Cache-Control (P-U4-3) | `routes.go:assetsHandler` |
| index.html no-cache (P-U4-3) | `routes.go:spaHandler` |
| Token 의도적 제거 (NFR-U4-S1) | `api_results.go:memberEntry` |
| RFC1918 IPv4 LAN 필터 | `lan.go:PrintLANAddresses` |
| **embed placeholder 위치 조정** | `cmd/mafia-game/web/dist/index.html` (디자인 문서의 `web/dist`에서 변경) |

---

## 5. 알려진 제한 / 후속 작업

| 항목 | 상태 |
|---|---|
| Vite outDir 설정 | U5 단계에서 `../cmd/mafia-game/web/dist`로 명시 필요 |
| TLS / wss:// | NFR 비-요구사항 — 사내 LAN 가정 |
| Prometheus / health detail | 비-요구사항 — `/healthz`는 단순 200 |
| `--config-file` 지원 | 비-요구사항 — flag/env로 충분 |

---

## 6. 변경된 모듈 메타데이터

`go.mod`: 신규 직접 의존 **0개**. 누계 직접 의존:
- `modernc.org/sqlite v1.50.0` (U2)
- `github.com/gorilla/websocket v1.5.3` (U3)
- (U4의 0개)

> 외부 직접 의존 누계 2개 — NFR-7 정책 만족.
