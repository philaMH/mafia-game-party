# NFR Requirements — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 NFR-1/2/4/6/7, `construction/u4-http-bootstrap-static/functional-design/*.md`, `plans/u4-http-bootstrap-nfr-requirements-plan.md`

---

## 1. NFR 영역별 요구사항

### 1.1 Reliability (NFR-1)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U4-R1 | SIGINT/SIGTERM 수신 시 graceful shutdown | **< 7초** (HTTP 5s + Hub 즉시 + mgr 2s) | 통합 테스트 |
| NFR-U4-R2 | 두 번째 시그널은 즉시 강제 종료 | os.Exit(1) | 코드 리뷰 |
| NFR-U4-R3 | http.Server.Shutdown 후 새 요청은 ErrServerClosed | 통합 테스트로 검증 | 단위 테스트 |
| NFR-U4-R4 | 단계별 종료 실패는 다음 단계 진행 (WARN 로그만) | 코드 리뷰 | — |
| NFR-U4-R5 | embed.FS는 항상 valid (placeholder 보장) | `go build` 통과 | CI 게이트 |

### 1.2 Performance (NFR-2)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U4-P1 | `/api/results?limit=50` 응답 지연 | **p99 < 50 ms** (LAN, 100 게임 누적 기준) | 단위 테스트 + benchmark |
| NFR-U4-P2 | 정적 자산(`/assets/main-abc.js`) 응답 지연 | **p99 < 20 ms** (cold + warm cache) | benchmark |
| NFR-U4-P3 | SPA fallback (`/play/route` → index.html) 응답 지연 | **p99 < 30 ms** | 단위 테스트 |
| NFR-U4-P4 | `/healthz` 응답 지연 | p99 < 5 ms | 단위 테스트 |
| NFR-U4-P5 | 동시 100 HTTP 요청 처리 (정적 + API 혼합) | 모두 성공 응답 | 통합 테스트 |

### 1.3 Maintainability (NFR-6)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U4-M1 | 단위 테스트 라인 커버리지 | **≥ 85%** (`internal/transport/http`, NFR-U4-M1) | `go test -cover` |
| NFR-U4-M2 | 정적 분석 통과 | `go vet` 0 issue + `gofmt -l` empty | CI 게이트 |
| NFR-U4-M3 | godoc 주석 — 모든 공개 식별자 (Server, Config, New, PrintLANAddresses) | 코드 리뷰 |
| NFR-U4-M4 | 외부 의존성 한정 | **0개** 추가 (Q-NFR-U4-1=A) — 표준 lib만 | `go list -m all` |
| NFR-U4-M5 | Composition Root 단일 (main.go) | 코드 리뷰 — 다른 패키지에서 단위 직접 생성 금지 |
| NFR-U4-M6 | 라우팅 테이블은 한 곳에 집중 (httpx.New) | 코드 리뷰 |

### 1.4 Security (NFR-4)

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U4-S1 | `/api/results` 응답에 Token 미포함 | JSON 파싱 후 `members[*].token` 키 부재 | 단위 테스트 (Q-NFR-U4-9=A) |
| NFR-U4-S2 | 요청 페이로드(query value, body) 로그 미기록 | 코드 리뷰 + 로그 grep |
| NFR-U4-S3 | TLS 미지원 — 평문 ws://, http:// (LAN 한정) | 코드 리뷰 |
| NFR-U4-S4 | CORS Origin 화이트리스트 없음 — 모든 Origin 허용 | gorilla CheckOrigin → true |

### 1.5 Concurrency

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U4-C1 | 추가 mutex 없음 — net/http가 자체 처리 | 코드 리뷰 |
| NFR-U4-C2 | 데이터 레이스 미발생 | `go test -race` 통과 | CI 게이트 |
| NFR-U4-C3 | 시그널 채널 1회 구독 후 Notify | 코드 리뷰 |

### 1.6 Build / Operability

| ID | 요구사항 | 측정 가능 한도 | 검증 방법 |
|---|---|---|---|
| NFR-U4-B1 | `go build ./cmd/mafia-game` 단일 바이너리 산출 | 1 file output | CI 게이트 |
| NFR-U4-B2 | 빌드 시간 (재빌드 캐시 hit 기준) | < 5초 | 측정 |
| NFR-U4-B3 | `web/dist/index.html` placeholder 부재 시 빌드 실패 | embed 표준 | CI 게이트 (Q-NFR-U4-10=A) |
| NFR-U4-B4 | LAN IP 출력은 시작 시 1회 | 코드 리뷰 |

---

## 2. 트레이드오프 결정

| 트레이드오프 | 본 단위의 결정 | 근거 |
|---|---|---|
| 라우터 (std vs chi/mux) | **net/http.ServeMux** | NFR-7 외부 의존 0, Go 1.22+ pattern 충분 |
| 정적 자산 (embed vs 디렉터리) | **embed** | 단일 바이너리, 배포 단순 (NFR-7) |
| Auth (없음 vs 토큰) | **없음** | 사내 LAN, NFR-4 LAN 한정 |
| 캐시 (immutable vs no-cache) | **assets immutable + index.html no-cache** | Vite hash 파일명 가정 |
| 종료 budget (긴 vs 짧은) | **7초** | UX (호스트 재시작 시 빠른 응답) + 마지막 SaveSnapshot 안전 |
| placeholder index.html (있음 vs 없음) | **있음** | 백엔드만 빌드해도 `go build` 통과 (개발 워크플로우 단순) |

---

## 3. 추적성 (FR/NFR ↔ U4 NFR)

| 출처 | 본 문서 항목 |
|---|---|
| NFR-1 (안정성·복원) | NFR-U4-R1~R5 |
| NFR-2 (성능) | NFR-U4-P1~P5 |
| NFR-4 (비공개·LAN 한정) | NFR-U4-S1~S4 |
| NFR-6 (도메인 분리·유지보수성) | NFR-U4-M1~M6 |
| NFR-7 (외부 서비스 0) | NFR-U4-M4 (외부 lib 0), NFR-U4-B1 (단일 바이너리) |
| FR-1.1 (LAN URL) | NFR-U4-B4 |
| FR-6.3 (결과 조회) | NFR-U4-P1, NFR-U4-S1 |

---

## 4. 검증 게이트 (Build & Test 단계)

다음 모든 항목이 통과해야 U4가 출하 가능:

1. ✅ `go vet ./internal/transport/http/... ./cmd/mafia-game/...` 0 issue
2. ✅ `gofmt -l ./internal/transport/http/ ./cmd/mafia-game/` empty
3. ✅ `go test -race ./internal/transport/http/...` 통과 (NFR-U4-C2)
4. ✅ `go test -coverprofile=...` 라인 ≥ **85%** (NFR-U4-M1)
5. ✅ `go build ./cmd/mafia-game` 단일 바이너리 산출 (NFR-U4-B1)
6. ✅ /api/results JSON에 token 키 부재 검증 단위 테스트 통과 (NFR-U4-S1)
7. ✅ graceful shutdown 통합 테스트 < 7초 (NFR-U4-R1)
8. ✅ `go list -m all` 직접 의존 +0 (gorilla/websocket과 modernc.org/sqlite는 기존)

---

## 5. 명시적 비-요구사항 (Non-Goals)

- **Scalability**: 단일 호스트 PC, 단일 게임. 멀티 인스턴스 / horizontal scaling 미지원.
- **Availability SLA**: 직접 책임 없음. graceful shutdown만 보장.
- **TLS / HTTPS**: 사내 LAN 가정 (NFR-4 LAN 한정).
- **Authentication / Authorization**: 호스트 PC 단일 사용자 가정.
- **Rate limiting / DOS 방어**: 신뢰된 LAN 클라이언트 가정.
- **CDN / 외부 호스팅**: 단일 바이너리에 자산 동봉 (NFR-7).
- **Hot reload / file watcher**: 운영 단순성 우선.
- **Subprotocol 협상**: gorilla 기본 동작 사용.
