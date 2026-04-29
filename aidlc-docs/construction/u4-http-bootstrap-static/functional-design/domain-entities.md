# Domain Entities — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `requirements.md` v1.1 FR-1.1/FR-6.3 + NFR-7, `application-design/component-methods.md` C7, `u3-public-api.md`, `u2-public-api.md`, `plans/u4-http-bootstrap-functional-design-plan.md`

본 문서는 U4의 도메인 엔티티(Server / Config / 라우팅 테이블 / Composition Root)를 정의합니다.

---

## 1. Server 인터페이스 (C7 HTTPServer)

```go
// 패키지명은 'http' 충돌 회피를 위해 'httpx'.
package httpx

type Server interface {
    // ListenAndServe blocks until the server stops or returns an error
    // other than http.ErrServerClosed.
    ListenAndServe() error

    // Shutdown initiates a graceful shutdown of the underlying http.Server.
    // The provided ctx bounds the wait for in-flight requests.
    Shutdown(ctx context.Context) error
}

type Config struct {
    Addr        string  // "0.0.0.0:8080" 등
    Hub         ws.Hub
    Mgr         session.SessionManager
    Store       persistence.PersistenceStore
    Assets      fs.FS    // web/dist embed (Q-FD-U4-2=A)
    Logger      *slog.Logger
}

func New(cfg Config) (Server, error)
```

> 본 단위는 직접 `http.Server`를 구성하고 라우팅을 등록합니다. SessionManager / Hub / PersistenceStore는 Composition Root(`cmd/mafia-game/main.go`)에서 생성·주입됩니다.

---

## 2. Composition Root — `cmd/mafia-game/main.go`

```go
package main

import (
    "context"
    "embed"
    "flag"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/websocket"

    "github.com/saltware/mafia-game/internal/announce"
    "github.com/saltware/mafia-game/internal/game"
    "github.com/saltware/mafia-game/internal/persistence"
    "github.com/saltware/mafia-game/internal/session"
    "github.com/saltware/mafia-game/internal/transport/http"  // alias httpx
    "github.com/saltware/mafia-game/internal/transport/ws"
)

//go:embed all:web/dist
var webDist embed.FS
```

> `//go:embed all:web/dist`의 `all:` 접두사는 `_`/`.`로 시작하는 파일도 포함시켜 Vite의 `_`-suffix 자산까지 보장.

---

## 3. CLI 플래그 / 환경변수 (Q-FD-U4-6=A, Q-FD-U4-7=A)

| 플래그 | 환경변수 | 기본값 | 설명 |
|---|---|---|---|
| `--port` | `MAFIA_PORT` | `8080` | HTTP 리슨 포트 |
| `--db` | `MAFIA_DB_PATH` | `./data/mafia.db` | SQLite 파일 경로 |
| `--log-level` | `MAFIA_LOG_LEVEL` | `info` | slog 레벨 (debug/info/warn/error) |

**우선순위**: CLI 플래그 > 환경변수 > 기본값.

---

## 4. 라우팅 테이블 (확정)

Q-FD-U4-1=A: 표준 `net/http.ServeMux` (Go 1.22+ patterns).

| 패턴 | 핸들러 | 응답 | 비고 |
|---|---|---|---|
| `GET /healthz` | healthHandler | 200 `"ok"` (text/plain) | Q-FD-U4-5=A |
| `GET /ws` | hub.UpgradeHandler() | WebSocket 업그레이드 | U3 위임 |
| `GET /api/results` | resultsHandler | JSON | FR-6.3, Q-FD-U4-4=A |
| `GET /assets/{path...}` | assetsHandler | embed.FS 파일 (immutable cache) | Q-FD-U4-2=A, Q-FD-U4-12=A |
| `GET /` | spaHandler | index.html | SPA |
| `GET /{path...}` (SPA fallback) | spaHandler | index.html | Q-FD-U4-3=B |

**SPA fallback 적용 범위** (Q-FD-U4-3=B):
- `/api/*`, `/assets/*`, `/ws`, `/healthz` → 위 테이블의 핸들러
- 나머지(예: `/public`, `/play`, `/play/some/route`) → `index.html` 반환 → React Router가 클라이언트에서 해석

**Method 매칭**: 모든 핸들러는 `GET`만 지원. 다른 method는 405.

---

## 5. /api/results 응답 형식 (Q-FD-U4-4=A)

### 요청
```
GET /api/results?limit=50
```

| 쿼리 | 타입 | 기본 | 한도 |
|---|---|---:|---:|
| `limit` | int | 50 | 1~500 (이외는 400) |

### 응답
```json
{
  "results": [
    {
      "gameId": "...",
      "startedAt": "2026-04-26T10:00:00Z",
      "endedAt":   "2026-04-26T10:30:00Z",
      "winner":    "CITIZEN",
      "endReason": "CITIZEN_WIN",
      "options":   { "mafiaCount": 1, "introSecondsPerPlayer": 20, ... },
      "members":   [{"id": "...", "name": "철수", "joinedAt": "..."}],
      "reveal":    [{"id": "...", "name": "철수", "alive": false, "role": "MAFIA"}]
    }
  ]
}
```

> `winner`는 `null` 가능 (`HOST_FORCE_END`). 시간은 RFC3339 UTC. `persistence.GameResult` 그대로 직렬화.

### 응답 헤더
```
Content-Type: application/json; charset=utf-8
Cache-Control: no-store
```

---

## 6. 정적 자산 핸들러

### 6.1 `/assets/*` (Q-FD-U4-12=A)
- 응답 헤더: `Cache-Control: public, max-age=31536000, immutable` (Vite 빌드 hash 파일명 가정)
- 파일 없으면 → 404
- MIME type은 `mime` 패키지가 자동 판별

### 6.2 SPA fallback `/`, `/play`, `/public` 등
- 응답: `web/dist/index.html` 파일 그대로
- 응답 헤더: `Cache-Control: no-cache` (항상 최신 SPA bundle 로드)

### 6.3 placeholder `web/dist/index.html` (Q-FD-U4-15=A)
- U5 빌드 전이라도 바이너리가 빌드되도록, 본 단위 코드 작성 시 placeholder 파일을 함께 commit:
  ```html
  <!doctype html>
  <html><head><meta charset="utf-8"><title>mafia-game</title></head>
  <body><p>Frontend not built. Run <code>cd web && npm run build</code>.</p></body></html>
  ```

---

## 7. LAN IP 검색 정책 (Q-FD-U4-8=A)

```go
// PrintLANAddresses iterates net.InterfaceAddrs(), filtering for IPv4
// addresses in private RFC 1918 ranges (10/8, 172.16/12, 192.168/16),
// and writes "http://<ip>:<port>" lines to stdout (one per line).
//
// Loopback (127/8, ::1) and IPv6 are excluded by default.
func PrintLANAddresses(port int)
```

**필터 기준**:
- `addr.(*net.IPNet)` IPv4 only
- `ip.IsLoopback()` 제외
- `ip.IsPrivate()` (Go 1.17+) — 10/8, 172.16/12, 192.168/16
- 출력 순서: `net.InterfaceAddrs` 반환 순서 유지

**예시 출력**:
```
mafia-game listening on:
  http://192.168.1.42:8080
  http://10.0.0.5:8080
```

---

## 8. graceful shutdown 시퀀스 (Q-FD-U4-9=A)

```
SIGINT/SIGTERM 수신
  ↓
  ① http.Server.Shutdown(ctx 5초) — 진행 중 HTTP 요청 완료 대기
  ↓
  ② hub.Close() — 모든 WS 클라이언트 close + Subscribe 해제
  ↓
  ③ mgr.Close(ctx 2초) — 마지막 SaveSnapshot + persistence.Close
  ↓
  Exit(0)
```

총 budget: ~7초. 한도 내 마치지 못하면 `os.Exit(1)`.

---

## 9. 핵심 데이터 타입

```go
// internal/transport/http/server.go
type Config struct {
    Addr   string
    Hub    ws.Hub
    Mgr    session.SessionManager
    Store  persistence.PersistenceStore
    Assets fs.FS
    Logger *slog.Logger
}

type server struct {
    cfg     Config
    httpSrv *http.Server
    log     *slog.Logger
}

// 응답 타입 (api 핸들러)
type resultsResponse struct {
    Results []resultEntry `json:"results"`
}

type resultEntry struct {
    GameID    string             `json:"gameId"`
    StartedAt time.Time          `json:"startedAt"`
    EndedAt   time.Time          `json:"endedAt"`
    Winner    *string            `json:"winner"`
    EndReason string             `json:"endReason"`
    Options   game.Options       `json:"options"`
    Members   []memberEntry      `json:"members"`
    Reveal    []game.Player      `json:"reveal"`
}

type memberEntry struct {
    ID       game.PlayerID `json:"id"`
    Name     string        `json:"name"`
    JoinedAt time.Time     `json:"joinedAt"`
}
```

> `memberEntry`는 `Token`을 의도적으로 제외 (NFR-4 보안 — 결과 조회 API에 토큰 노출 금지).

---

## 10. 검증 체크리스트

- [x] Server 인터페이스 2 메서드
- [x] Config 6 필드 (Hub, Mgr, Store, Assets, Logger, Addr)
- [x] CLI 플래그 / 환경변수 매트릭스
- [x] 라우팅 테이블 6 패턴 + SPA fallback 정책
- [x] /api/results 응답 스키마 (members[].Token 제외)
- [x] LAN IP 검색 필터 기준 (private + IPv4 only)
- [x] graceful shutdown 3단계 시퀀스
- [x] placeholder index.html 정책 (embed 빌드 보장)
