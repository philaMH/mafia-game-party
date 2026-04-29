# Code Generation Plan — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**대상 단위**: U4 (`cmd/mafia-game`, `internal/transport/http`)
**참조**:
- `application-design/unit-of-work.md` §4
- `construction/u4-http-bootstrap-static/functional-design/*.md`
- `construction/u4-http-bootstrap-static/nfr-requirements/*.md`
- `construction/u4-http-bootstrap-static/nfr-design/*.md`
- U3 공개 API: `construction/u3-realtime-transport/code/u3-public-api.md`
- U2 공개 API: `construction/u2-session-persistence-announce/code/u2-public-api.md`
- U1 공개 API: `construction/u1-game-core/code/u1-public-api.md`

> 본 plan은 U4 Code Generation의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트

**책임**: Composition Root + HTTP 라우팅 + 정적 자산 embed + LAN IP 출력 + graceful shutdown.

**구현 대상 요구사항**:
- FR-1.1 (LAN URL 출력)
- FR-6.3 (`/api/results` 결과 조회)
- NFR-1 (graceful shutdown)
- NFR-7 (단일 바이너리, 외부 의존 0)

**의존**:
- **U2 SessionManager** (와이어링 + Close)
- **U3 WSHub** (UpgradeHandler 사용)
- **persistence.PersistenceStore** (`/api/results` ListResults)
- **U1 game.Engine, KeywordPool, RoleAssigner** (NewDefault)
- **announce.NewDefaultCatalog**
- **외부**: 모두 표준 라이브러리. **신규 외부 의존 0개**.

**산출물**: `cmd/mafia-game/main.go` + `internal/transport/http/*` (7 Go 파일) + 단위 테스트 + `web/dist/index.html` placeholder.

---

## 1. 코드 위치 결정

| 항목 | 위치 |
|---|---|
| Workspace Root | `/Users/myunghoonkang/study/saltware-ai-dlc/mafia-game` |
| Composition Root | `cmd/mafia-game/main.go` |
| HTTP server lib | `internal/transport/http/` (alias `httpx`) |
| placeholder | `web/dist/index.html` |
| 문서 산출물 | `aidlc-docs/construction/u4-http-bootstrap-static/code/` |

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

### 3.1 placeholder 정적 자산 (P-U4-5)
- [x] (G1) `cmd/mafia-game/web/dist/index.html` placeholder 생성 — embed 상위 경로 불허로 main.go 형제로 이동

### 3.2 `internal/transport/http/` 패키지 (LC-U4-2~10)
- [x] (G2) `doc.go` — 패키지 godoc (alias 'httpx')
- [x] (G3) `server.go` — `Server` 인터페이스 + `Config` + `New` + http.Server 타임아웃 (P-U4-1)
- [x] (G4) `middleware.go` — `loggingMiddleware` + `statusRecorder` (P-U4-2)
- [x] (G5) `routes.go` — `buildMux` + `healthHandler` + `spaHandler` + `assetsHandler` (LC-U4-6/8/9)
- [x] (G6) `api_results.go` — `resultsHandler` + `buildResultsResponse` (Token 제외, P-U4-6)
- [x] (G7) `lan.go` — `PrintLANAddresses` (LC-U4-10)

### 3.3 Composition Root
- [x] (G8) `cmd/mafia-game/main.go` — flag/env 파싱 + slog + 단위 와이어링 + signal.NotifyContext + graceful shutdown 3단계

### 3.4 단위 테스트 — `internal/transport/http/`
- [x] (G9) `server_test.go` — New 인자 검증 + 모든 라우트 동작 + Token 미포함 (white-box)
- [x] (G10) `middleware_test.go` — statusRecorder + loggingMiddleware 4필드 + 페이로드 미기록 검증
- [x] (G11) `routes_test.go` — healthHandler/assetsHandler/spaHandler/buildMux 라우트 등록
- [x] (G12) `api_results_test.go` — buildResultsResponse Token 제거 + limit 경계값 7종 + DB 에러
- [x] (G13) `lan_test.go` — PrintLANAddresses 출력 형식 검증
- [x] (G14) `integration_test.go` — 실제 net.Listen + ListenAndServe + Shutdown < 5초

### 3.5 문서 산출물
- [x] (G15) `aidlc-docs/construction/u4-http-bootstrap-static/code/u4-code-summary.md`
- [x] (G16) `aidlc-docs/construction/u4-http-bootstrap-static/code/u4-public-api.md`

### 3.6 N/A 단계
- [x] (G17) Deployment Artifacts — N/A (단일 바이너리에 통합)
- [x] (G18) DB Migration Scripts — N/A
- [x] (G19) Frontend Components — N/A (U5에서 React SPA 작성)

---

## 4. Definition of Done

- [x] (V1) 모든 G1~G19 [x]
- [x] (V2) `go build ./cmd/mafia-game` 단일 바이너리 산출 (Mach-O 64-bit arm64)
- [x] (V3) `go vet ./...` 0 issue
- [x] (V4) `gofmt -l ./internal/transport/http/ ./cmd/mafia-game/` empty
- [x] (V5) `go test ./internal/transport/http/...` 모든 테스트 통과
- [x] (V6) `go test -race` 통과
- [x] (V7) 합산 커버리지 **87.6%** ≥ 85% (httpx 89.2% / ws 89.0% / session 88.2% / announce 93.3% / persistence 80.2%)
- [x] (V8) 직접 의존 추가 0 — `modernc.org/sqlite` + `gorilla/websocket` 그대로

---

## 5. 스토리/요구사항 추적성

| 요구사항 | 구현 단계 |
|---|---|
| FR-1.1 (LAN URL 출력) | G7 (PrintLANAddresses), G8 (main.go에서 호출) |
| FR-6.3 (결과 조회 API) | G6 (resultsHandler), G12 (테스트) |
| NFR-1 (graceful shutdown) | G8 (signal.NotifyContext + 7초 budget), G14 (테스트) |
| NFR-4 (비공개 정보) | G6 (Token 제외), G12 (검증) |
| NFR-7 (외부 의존 0) | G2~G7 모두 표준 lib만 사용 |
| NFR-U4-S2 (페이로드 미로그) | G4 (loggingMiddleware 4필드만) |
| NFR-U4-B3 (placeholder 보장) | G1 (placeholder commit) |

---

## 6. 산출물 요약 (예상)

| 종류 | 파일 수 | 위치 |
|---|---:|---|
| http 코드 | 6 | `internal/transport/http/{doc,server,middleware,routes,api_results,lan}.go` |
| http 테스트 | 6 | `internal/transport/http/*_test.go` |
| Composition Root | 1 | `cmd/mafia-game/main.go` |
| placeholder | 1 | `web/dist/index.html` |
| 문서 요약 | 2 | `aidlc-docs/construction/u4-.../code/*.md` |

---

## 7. 사용자 승인 게이트

본 plan에 동의하시면 **"승인"** 또는 **"continue"** 로 답변. 변경이 필요하면 구체적 항목을 알려주세요 (예: "G14에 동시 100 요청 테스트 추가").
