# Business Rules — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `domain-entities.md`, `business-logic-model.md`

---

## 1. 공통 규칙 (BR-U4-COMMON)

| ID | 규칙 |
|---|---|
| BR-U4-COMMON-1 | Composition Root는 `cmd/mafia-game/main.go` 단일 — 모든 단위 와이어링은 여기서만. 다른 패키지에서 SessionManager / Hub 직접 생성 금지 |
| BR-U4-COMMON-2 | HTTPServer 코드(`internal/transport/http/`)는 표준 라이브러리만 사용. 외부 lib 추가 0 (NFR-7) |
| BR-U4-COMMON-3 | 모든 HTTP 응답은 UTF-8 인코딩 |
| BR-U4-COMMON-4 | 핸들러 panic은 `http.Server`의 기본 recover로 격리됨 — 추가 미들웨어 불필요 |

---

## 2. 라우팅 규칙 (BR-U4-ROUTE, Q-FD-U4-1=A, Q-FD-U4-3=B)

| ID | 규칙 |
|---|---|
| BR-U4-ROUTE-1 | `net/http.ServeMux` (Go 1.22+)을 사용 — `GET /api/results`, `GET /assets/...` 패턴 |
| BR-U4-ROUTE-2 | 모든 정의 핸들러는 GET method만 매칭. 다른 method는 405 (ServeMux 자동) |
| BR-U4-ROUTE-3 | 명시 핸들러 우선순위: `/healthz`, `/ws`, `/api/*`, `/assets/*`. 그 외 모든 경로(예: `/play`, `/public`, `/play/some/route`) → SPA index.html (history fallback) |
| BR-U4-ROUTE-4 | 패턴이 충돌하면 가장 specific한 패턴이 우선 (ServeMux 표준 동작) |
| BR-U4-ROUTE-5 | `/ws`는 Hub의 `UpgradeHandler()`에 위임 — U4는 매핑만 |

---

## 3. 정적 자산 규칙 (BR-U4-STATIC, Q-FD-U4-2=A, Q-FD-U4-12=A, Q-FD-U4-15=A)

| ID | 규칙 |
|---|---|
| BR-U4-STATIC-1 | React SPA 산출물은 `//go:embed all:web/dist`로 동봉. `embed.FS`를 `fs.Sub("web/dist")`로 잘라 `Server.Assets` 주입 |
| BR-U4-STATIC-2 | `/assets/*`는 `Cache-Control: public, max-age=31536000, immutable` (Vite hash 파일명 가정) |
| BR-U4-STATIC-3 | SPA fallback 응답(`index.html`)은 `Cache-Control: no-cache` — 항상 최신 bundle 로드 |
| BR-U4-STATIC-4 | `web/dist/index.html`은 placeholder("Frontend not built. Run npm run build.")를 함께 commit해 `go build`가 항상 성공하도록 보장 |
| BR-U4-STATIC-5 | embed에서 파일을 찾지 못하면 404 응답 (FileServerFS 표준 동작) |
| BR-U4-STATIC-6 | MIME 타입은 `mime` 패키지가 자동 판별 — `.js`, `.css`, `.html`, `.svg`, `.woff2` 등 모두 지원 |

---

## 4. /api/results 규칙 (BR-U4-API, Q-FD-U4-4=A)

| ID | 규칙 |
|---|---|
| BR-U4-API-1 | URL: `GET /api/results?limit=N`. 기본 limit=50, 허용 범위 1~500. 범위 위반 시 400 |
| BR-U4-API-2 | 응답 본문은 `{results: [...]}`. 빈 결과는 `{results: []}` |
| BR-U4-API-3 | `members` 항목에서 **Token 필드를 의도적으로 제외** (NFR-4 보안) |
| BR-U4-API-4 | `winner`는 `null` 가능 (`HOST_FORCE_END`). `endReason`은 항상 문자열 |
| BR-U4-API-5 | 시간은 RFC 3339 UTC (encoding/json 기본) |
| BR-U4-API-6 | 응답 헤더: `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store` |
| BR-U4-API-7 | DB 오류 시 500 + `"internal error"` 텍스트. 상세 에러는 slog ERROR 로그 |
| BR-U4-API-8 | rate limiting 없음 — 사내 LAN, 신뢰된 클라이언트 가정 |

---

## 5. /healthz 규칙 (BR-U4-HEALTH)

| ID | 규칙 |
|---|---|
| BR-U4-HEALTH-1 | `GET /healthz` → 200 + `"ok"` (text/plain). 단순 liveness 프로브 |
| BR-U4-HEALTH-2 | DB 상태 / SessionManager 상태 검사 안 함 (단순 부담 없는 ping) |

---

## 6. WebSocket 규칙 (BR-U4-WS)

| ID | 규칙 |
|---|---|
| BR-U4-WS-1 | `GET /ws` → `hub.UpgradeHandler()` 위임. U4는 라우팅만 |
| BR-U4-WS-2 | gorilla `Upgrader.CheckOrigin`은 항상 true (Q-FD-U4-11=A) — 사내 LAN 전제 |
| BR-U4-WS-3 | `WriteTimeout`은 0 (무제한) — long-lived WS connection 호환 |

---

## 7. graceful shutdown 규칙 (BR-U4-SHUTDOWN, Q-FD-U4-9=A)

| ID | 규칙 |
|---|---|
| BR-U4-SHUTDOWN-1 | SIGINT / SIGTERM 1회 수신 시 종료 시작. 두 번째 시그널은 즉시 `os.Exit(1)` (강제) |
| BR-U4-SHUTDOWN-2 | 종료 순서: ① `http.Server.Shutdown(ctx 5초)` ② `hub.Close()` (즉시) ③ `mgr.Close(ctx 2초)` |
| BR-U4-SHUTDOWN-3 | 각 단계 실패는 WARN 로그만 — 다음 단계 강행 |
| BR-U4-SHUTDOWN-4 | 총 budget ~7초. 한도 내 마치지 못하면 ctx 만료 후 `os.Exit(0)` |
| BR-U4-SHUTDOWN-5 | shutdown 진행 중 새 요청은 `http.ErrServerClosed` 반환 |
| BR-U4-SHUTDOWN-6 | `srv.ListenAndServe()`이 `http.ErrServerClosed` 반환은 정상 종료 — error로 처리 안 함 |

---

## 8. LAN IP 출력 규칙 (BR-U4-LAN, Q-FD-U4-8=A)

| ID | 규칙 |
|---|---|
| BR-U4-LAN-1 | 시작 직후 `http://<ip>:<port>` 형식으로 stdout 출력 |
| BR-U4-LAN-2 | `net.InterfaceAddrs()` 사용. IPv4 only, RFC1918 private only, loopback 제외 |
| BR-U4-LAN-3 | 매칭 IP가 0개면 `http://localhost:<port>` fallback 1회 출력 |
| BR-U4-LAN-4 | 출력 순서는 InterfaceAddrs 반환 순서 유지 — 결정적 정렬 안 함 |
| BR-U4-LAN-5 | 일시적 net error는 한 줄 `"could not detect LAN: ..."` 출력 후 진행 |

---

## 9. 인증 / Origin 규칙 (BR-U4-AUTH, Q-FD-U4-10=A, Q-FD-U4-11=A)

| ID | 규칙 |
|---|---|
| BR-U4-AUTH-1 | HTTP / WS 요청에 인증 헤더 검사 없음 (사내 LAN, NFR-4 LAN 한정) |
| BR-U4-AUTH-2 | 호스트 식별은 SessionManager 단독 (BR-U2-AUTH 참조). U4는 알지 못함 |
| BR-U4-AUTH-3 | Origin 화이트리스트 없음 — 모든 Origin 허용 |
| BR-U4-AUTH-4 | TLS / wss:// 미지원 — 평문 ws:// (NFR-4 비-요구사항) |

---

## 10. 로깅 규칙 (BR-U4-LOG, Q-FD-U4-13=A)

| ID | 규칙 |
|---|---|
| BR-U4-LOG-1 | slog 로거 사용. 운영 기본 INFO. `--log-level debug`로 DEBUG 활성 |
| BR-U4-LOG-2 | 모든 HTTP 요청은 method, path, status, duration_ms 4 필드 INFO 기록 |
| BR-U4-LOG-3 | Query 값, body, 응답 본문 미기록 — NFR-4 토큰/PII 보호 |
| BR-U4-LOG-4 | 시작 시 `"mafia-game listening on:"` + LAN IP 줄들 stdout, slog 외부 출력 (사용자가 콘솔에서 즉시 확인) |
| BR-U4-LOG-5 | shutdown 단계별 INFO 로그 (`"signal received"`, `"goodbye"`) |
| BR-U4-LOG-6 | 핸들러 panic은 http.Server 표준 recover에 맡김. 추가 로그 없음 |

---

## 11. 설정 규칙 (BR-U4-CFG)

| ID | 규칙 |
|---|---|
| BR-U4-CFG-1 | CLI 플래그 / 환경변수 / 기본값 우선순위 — 플래그 > 환경 > 기본 |
| BR-U4-CFG-2 | 알 수 없는 플래그는 `flag.Parse()`가 실패시켜 `os.Exit(2)` |
| BR-U4-CFG-3 | `--log-level`이 알 수 없는 값이면 INFO로 fallback + WARN 로그 |
| BR-U4-CFG-4 | port 범위는 1~65535. 그 외는 startup 실패 |

---

## 12. FR/NFR 추적성

| 출처 | 본 문서 규칙 |
|---|---|
| FR-1.1 (LAN URL + 단일 호스트) | BR-U4-LAN-1~5, BR-U4-WS-2 |
| FR-6.3 (과거 결과 조회) | BR-U4-API-1~7 |
| NFR-4 (비공개 정보·LAN 한정) | BR-U4-API-3 (Token 제외), BR-U4-LOG-3, BR-U4-AUTH-1~4 |
| NFR-7 (외부 서비스 0) | BR-U4-COMMON-2 (표준 lib만), BR-U4-STATIC-1 (단일 바이너리) |
| NFR-1 (graceful shutdown) | BR-U4-SHUTDOWN-1~6 |

---

## 13. 검증 체크리스트

- [x] Composition Root 단일 파일 정책
- [x] 표준 lib만 외부 의존 0
- [x] 라우팅 5 패턴 + SPA fallback
- [x] /assets/* immutable 캐시 vs index.html no-cache
- [x] /api/results Token 제외 (NFR-4)
- [x] graceful shutdown 3단계 + 7초 budget
- [x] LAN IP RFC1918 필터 + IPv4 only
- [x] 로그 페이로드 미기록
- [x] 모든 Primary FR/NFR이 규칙으로 매핑됨 (§12)
