# Functional Design Plan — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**대상 단위**: U4 / HTTP Bootstrap & Static (`cmd/mafia-game`, `internal/transport/http`)
**컴포넌트**: C7 HTTPServer + `cmd/mafia-game/main.go`
**참조**:
- `aidlc-docs/inception/application-design/unit-of-work.md` §4
- `aidlc-docs/inception/application-design/component-methods.md` C7
- `aidlc-docs/inception/application-design/component-dependency.md`
- U3 공개 API: `aidlc-docs/construction/u3-realtime-transport/code/u3-public-api.md`
- U2 공개 API: `aidlc-docs/construction/u2-session-persistence-announce/code/u2-public-api.md`
- U1 공개 API: `aidlc-docs/construction/u1-game-core/code/u1-public-api.md`

> 본 plan은 U4 Functional Design의 단일 진실 소스입니다.

---

## 0. 단위 컨텍스트 분석

**책임**:
1. **Composition Root**: 모든 단위(U1/U2/U3) 의존성 주입·와이어링 + graceful shutdown.
2. **HTTP 라우팅**: `/`, `/public`, `/play` (SPA), `/assets/*` (정적), `/ws` (Hub.UpgradeHandler), `/api/results` (FR-6.3), `/healthz`.
3. **정적 자산 동봉**: React SPA 빌드 산출물(`web/dist/`)을 `//go:embed`로 단일 바이너리에 포함 (NFR-7).
4. **LAN IP 콘솔 출력**: 시작 시 `http://<LAN-IP>:<port>` 출력 (FR-1.1 보조).

**비책임**: 비즈니스 로직, WebSocket 라우팅 본체, 프론트엔드 빌드.

**입력**:
- 환경변수 / CLI 플래그 (port, db path, log level)
- HTTP 요청
- OS 시그널 (SIGINT, SIGTERM)

**출력**:
- HTTP 응답 (정적 SPA, JSON results, WS 업그레이드)
- 콘솔 로그 (시작 메시지 + slog 구조화 로그)

**의존**:
- **U2 SessionManager** (orchestration target)
- **U3 WSHub** (UpgradeHandler 사용)
- **persistence.PersistenceStore** (`/api/results`용 ListResults 호출)
- **U5 web/dist** (정적 자산 — embed.FS)
- **외부**: `net/http`, `embed`, `os`, `os/signal`, `flag`, `net` — 모두 Go 표준 라이브러리. **신규 외부 lib 0개**.

**Primary FR/NFR**: FR-1.1 (LAN URL), FR-6.3 (`/api/results`), NFR-7 (단일 바이너리, 외부 서비스 0).

---

## 1. 단계 체크리스트

- [x] (1) 단위 컨텍스트·책임·비책임 정의
- [x] (2) 결정 질문 작성 (Q-FD-U4-1~15)
- [x] (3) plan 문서 작성 (본 파일)
- [x] (4) 사용자 답변 수집 — "권장 승인"
- [x] (5) 답변 일관성 검증·모호성 해결 — 모호성 없음
- [x] (6) `domain-entities.md` — Server 인터페이스 + Config + 라우팅 테이블 + Composition Root
- [x] (7) `business-logic-model.md` — 부팅 시퀀스, graceful shutdown, /api/results, LAN IP 출력
- [x] (8) `business-rules.md` — BR-U4-* 라우팅·부팅·종료·정적 자산·LAN 검색 규칙
- [x] (9) audit + aidlc-state 갱신, 사용자 승인 게이트

---

## 2. 결정 질문 (Q-FD-U4-1 ~ Q-FD-U4-15)

### Q-FD-U4-1. HTTP 라우터 선택

라우팅을 어떻게 구현?

- **A.** **표준 `net/http.ServeMux`** (Go 1.22+의 path-pattern 지원: `GET /api/results`, `GET /play/{$}`). 외부 lib 0. 본 PoC 권장.
- **B.** `gorilla/mux` 또는 `chi` 등 외부 라우터. NFR-7 의존 +1.
- **C.** 커스텀 핸들러 chain.

[Answer]: A

### Q-FD-U4-2. 정적 자산 동봉 방식

React `web/dist/`를 어떻게 바이너리에 동봉?

- **A.** **`//go:embed web/dist/*` + `embed.FS` 핸들러** — 단일 바이너리, 빌드 시 React 산출물 필수. (component-methods.md C7 그대로)
- **B.** 런타임에 `web/dist` 디렉터리에서 직접 서빙 — 배포 단순성↓.
- **C.** 별도 CDN/외부 호스팅 — NFR-7 위반.

[Answer]: A

### Q-FD-U4-3. SPA history fallback 처리

`/public`, `/play` 같은 SPA 경로는 React Router가 처리. 서버는 어떻게?

- **A.** **`/`, `/public`, `/play`, `/play/*`는 모두 `index.html` 반환** — React Router가 클라이언트에서 라우팅. `/assets/*`만 실제 파일.
- **B.** 프리픽스 매칭으로 `/api/*`, `/assets/*`, `/ws` 외 모든 경로에 대해 `index.html`.
- **C.** 명시적 라우트만 — 잘못된 경로는 404.

[Answer]: B

### Q-FD-U4-4. `/api/results` 응답 포맷

게임 결과 목록 JSON 응답은?

- **A.** **`{results: [{gameId, startedAt, endedAt, winner?, endReason, members[], reveal[]}]}`** — `persistence.GameResult`를 그대로 직렬화. `?limit=N` 쿼리 파라미터 (기본 50).
- **B.** 페이지네이션 cursor 기반 (`?cursor=...&limit=...`).
- **C.** GraphQL.

[Answer]: A

### Q-FD-U4-5. /healthz 응답

부팅·헬스 체크는 어떻게?

- **A.** **단순 `200 OK` + `"ok"` 텍스트**. 부담 없는 헬스 프로브.
- **B.** `{status: "ok", uptime, version}` JSON.
- **C.** 헬스 엔드포인트 미제공.

[Answer]: A

### Q-FD-U4-6. 포트 / 주소 설정

서버 포트는 어떻게 결정?

- **A.** **CLI 플래그 `--port=8080` (기본 8080) + 환경변수 `MAFIA_PORT`로 오버라이드**. listen은 `0.0.0.0:<port>` (모든 NIC).
- **B.** 0.0.0.0 자동 + 빈 포트 자동 할당.
- **C.** 설정 파일 YAML/TOML.

[Answer]: A

### Q-FD-U4-7. DB 경로 설정

SQLite 파일 경로는?

- **A.** **CLI 플래그 `--db=./data/mafia.db` + 환경변수 `MAFIA_DB_PATH` 오버라이드** (U2 BR-U2-PERSIST-8과 일치).
- **B.** OS-specific 디폴트 (~/.mafia-game/db).
- **C.** 메모리 SQLite (휘발성).

[Answer]: A

### Q-FD-U4-8. LAN IP 검색 방법

호스트 PC의 LAN IP를 어떻게 찾나?

- **A.** **`net.InterfaceAddrs()` 반복 → loopback 제외 + IPv4 only + private range(10/8, 172.16/12, 192.168/16) 필터** → 모든 매칭 IP를 콘솔에 `http://<ip>:<port>` 형식으로 출력.
- **B.** 첫 번째 non-loopback IP만.
- **C.** 외부 STUN/UPnP.

[Answer]: A

### Q-FD-U4-9. graceful shutdown 시그널 처리

SIGINT/SIGTERM 수신 시 종료 순서?

- **A.** **시그널 → http.Server.Shutdown(ctx 5초) → hub.Close() → mgr.Close(ctx 2초)** — 역순 라이프사이클. 5+2초 budget 안에 모든 자원 정리.
- **B.** 즉시 os.Exit(0).
- **C.** ctx 무한 대기.

[Answer]: A

### Q-FD-U4-10. 호스트 인증

호스트(첫 입장자)는 어떻게 식별?

- **A.** **인증 없음** — 첫 `host:create-session`을 보낸 클라이언트가 호스트가 됨 (PoC + 사내 LAN). 한 번 생성되면 다음 LOBBY까지 다른 클라이언트의 `host:create-session`은 거부됨 (U2 이미 그렇게 동작).
- **B.** 시작 시 호스트 비밀번호 환경변수 발급.
- **C.** 호스트 PC 자체에서 자동 호스트 토큰 파일 생성.

[Answer]: A

### Q-FD-U4-11. CORS / Origin 정책

WebSocket / API 요청의 Origin 검증?

- **A.** **모든 Origin 허용** (사내 LAN 가정, NFR-4 LAN 한정). gorilla Upgrader.CheckOrigin → true.
- **B.** Same-Origin만.
- **C.** 환경변수로 화이트리스트.

[Answer]: A

### Q-FD-U4-12. 정적 자산 캐시 정책

브라우저 캐시 헤더는?

- **A.** **`/assets/*`는 `Cache-Control: public, max-age=31536000, immutable`** (Vite 빌드 시 hash 파일명). `/index.html`은 `Cache-Control: no-cache`.
- **B.** 모두 캐시 없음.
- **C.** 기본 (Last-Modified만).

[Answer]: A

### Q-FD-U4-13. 로깅

요청 로깅은?

- **A.** **slog INFO 레벨 — method, path, status, duration. WS 업그레이드는 path만 (페이로드 미기록)**.
- **B.** Apache Combined Log Format.
- **C.** 로깅 안 함.

[Answer]: A

### Q-FD-U4-14. 단위 테스트 패턴

`internal/transport/http/`의 테스트는?

- **A.** **`httptest.NewServer` + 표준 `http.Get` 클라이언트**. mock SessionManager / Hub 사용. 정적 자산은 `embed.FS` placeholder + 내장 fallback.
- **B.** 실제 React 빌드 산출물 필수 — CI에서 `npm run build` 선행.
- **C.** 테스트 안 함 (얇은 래퍼이므로).

[Answer]: A

### Q-FD-U4-15. embed.FS 비어있을 때 동작

`web/dist`가 비어있는 상태에서 `go build` 가능한가? (개발 초기 단계)

- **A.** **placeholder index.html(예: "frontend not built")을 함께 commit** → embed가 항상 성공. 운영 빌드 전 `npm run build`로 덮어씀.
- **B.** `web/dist`가 없으면 빌드 실패 — 명시적 에러.
- **C.** embed 옵션 `embed.FS`를 nil-safe하게 처리 + 런타임에서 디렉터리 부재 시 503.

[Answer]: A

---

## 3. 산출물 예상

| 파일 | 책임 |
|---|---|
| `domain-entities.md` | Server 인터페이스 + Config 타입 + 라우팅 테이블 + Composition Root 정의 |
| `business-logic-model.md` | main.go 부팅 시퀀스 + LAN IP 검색 + /api/results 핸들러 + graceful shutdown 흐름 + 시퀀스 다이어그램 |
| `business-rules.md` | BR-U4-COMMON/ROUTE/STATIC/API/SHUTDOWN/LAN/AUTH/LOG 약 30항목 + FR/NFR 추적성 |

---

## 4. 사용자 승인 게이트

본 plan과 답변을 검토해 주세요. 각 `[Answer]:` 줄을 변경하시려면 해당 문자를 수정하시고, 그 외 일관성·추가 질문이 있으시면 알려주세요. 모든 답변에 동의하시면 **"완료"** 또는 **"승인"** 으로 응답해 주시면 plan에 따라 산출물 3종을 생성하고 다음 단계인 NFR Requirements로 진입하기 위한 추가 승인 게이트를 제시합니다.
