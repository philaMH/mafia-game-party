# Logical Components — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-design-patterns.md`, `tech-stack-decisions.md`, `functional-design/*.md`

본 문서는 U4의 논리적 구성요소를 정의합니다. U4는 두 곳에 위치합니다:
- **Composition Root**: `cmd/mafia-game/main.go`
- **HTTP server library**: `internal/transport/http/` (alias `httpx`)

---

## 1. 구성요소 카탈로그

| ID | 구성요소 | 위치 | 책임 | 적용 패턴 |
|---|---|---|---|---|
| LC-U4-1 | `main` | cmd/mafia-game | CLI/env → 단위 와이어링 → ListenAndServe → graceful shutdown | P-U4-4, P-U4-5 |
| LC-U4-2 | `Server` | internal/transport/http | http.Server 래퍼 + ListenAndServe + Shutdown | P-U4-1 |
| LC-U4-3 | `Config` | internal/transport/http | Hub/Mgr/Store/Assets/Logger/Addr DTO | — |
| LC-U4-4 | `New` | internal/transport/http | ServeMux 구성 + 모든 핸들러 등록 + middleware 체인 | — |
| LC-U4-5 | `loggingMiddleware` + `statusRecorder` | internal/transport/http | 요청 로깅 (4필드) | P-U4-2 |
| LC-U4-6 | `healthHandler` | internal/transport/http | `GET /healthz` → 200 "ok" | — |
| LC-U4-7 | `resultsHandler` | internal/transport/http | `GET /api/results` JSON | P-U4-6 |
| LC-U4-8 | `assetsHandler` | internal/transport/http | `/assets/*` immutable cache + FileServerFS | P-U4-3 |
| LC-U4-9 | `spaHandler` | internal/transport/http | `/`, `/play`, `/public` SPA fallback | — |
| LC-U4-10 | `PrintLANAddresses` | internal/transport/http | RFC1918 IPv4 LAN IP 출력 | — |
| LC-U4-11 | `web/dist/index.html` placeholder | web/dist | embed 빌드 보장 | P-U4-5 |

---

## 2. 패키지 / 파일 레이아웃 (확정)

```
cmd/mafia-game/
├── main.go                    # Composition Root (LC-U4-1)
└── (web/dist embed 자산 참조)

internal/transport/http/
├── doc.go                     # 패키지 godoc
├── server.go                  # Server 인터페이스 + impl + New (LC-U4-2/3/4)
├── routes.go                  # mux 등록 + healthHandler + spaHandler (LC-U4-6/9)
├── api_results.go             # resultsHandler (LC-U4-7)
├── assets.go                  # assetsHandler + cache header (LC-U4-8)
├── lan.go                     # PrintLANAddresses (LC-U4-10)
├── middleware.go              # loggingMiddleware + statusRecorder (LC-U4-5)
└── *_test.go

web/dist/
└── index.html                 # placeholder (LC-U4-11) — git에 commit
```

---

## 3. 구성요소별 상세

### 3.1 LC-U4-2 Server

```go
// 패키지명은 'http' 충돌 회피를 위해 'httpx' alias로 import.
package httpx

type Server interface {
    ListenAndServe() error
    Shutdown(ctx context.Context) error
}

type server struct {
    cfg     Config
    httpSrv *http.Server
    log     *slog.Logger
}

func (s *server) ListenAndServe() error      { return s.httpSrv.ListenAndServe() }
func (s *server) Shutdown(ctx context.Context) error { return s.httpSrv.Shutdown(ctx) }
```

### 3.2 LC-U4-3 Config

```go
type Config struct {
    Addr   string
    Hub    ws.Hub
    Mgr    session.SessionManager
    Store  persistence.PersistenceStore
    Assets fs.FS
    Logger *slog.Logger
}
```

### 3.3 LC-U4-4 New

```go
func New(cfg Config) (Server, error) {
    if cfg.Hub == nil || cfg.Mgr == nil || cfg.Store == nil || cfg.Assets == nil {
        return nil, errors.New("httpx: missing required Config fields")
    }
    log := cfg.Logger
    if log == nil { log = slog.Default() }

    mux := http.NewServeMux()
    mux.HandleFunc("GET /healthz", healthHandler)
    mux.Handle("GET /ws", cfg.Hub.UpgradeHandler())
    mux.Handle("GET /api/results", resultsHandler(cfg.Store, log))
    mux.Handle("GET /assets/", assetsHandler(cfg.Assets))
    mux.Handle("GET /", spaHandler(cfg.Assets))

    handler := loggingMiddleware(log)(mux)

    return &server{
        cfg: cfg, log: log,
        httpSrv: &http.Server{
            Addr: cfg.Addr,
            Handler: handler,
            ReadHeaderTimeout: 10 * time.Second,
            ReadTimeout: 30 * time.Second,
            WriteTimeout: 0,           // WS 호환
            IdleTimeout: 60 * time.Second,
        },
    }, nil
}
```

### 3.4 LC-U4-5 loggingMiddleware + statusRecorder

```go
type statusRecorder struct {
    http.ResponseWriter
    status int
}
func (r *statusRecorder) WriteHeader(code int) {
    r.status = code
    r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler { ... }
```

### 3.5 LC-U4-7 resultsHandler

```go
func resultsHandler(store persistence.PersistenceStore, log *slog.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        limit := 50
        if v := r.URL.Query().Get("limit"); v != "" {
            n, err := strconv.Atoi(v)
            if err != nil || n < 1 || n > 500 {
                http.Error(w, "invalid limit", http.StatusBadRequest)
                return
            }
            limit = n
        }
        results, err := store.ListResults(r.Context(), limit)
        if err != nil {
            log.Error("ListResults", "err", err)
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        resp := buildResultsResponse(results)
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.Header().Set("Cache-Control", "no-store")
        _ = json.NewEncoder(w).Encode(resp)
    }
}
```

> `buildResultsResponse`는 `members[].Token`을 의도적으로 제외 (NFR-U4-S1).

### 3.6 LC-U4-8 assetsHandler

```go
func assetsHandler(assets fs.FS) http.Handler {
    fileServer := http.FileServerFS(assets)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
        fileServer.ServeHTTP(w, r)
    })
}
```

### 3.7 LC-U4-9 spaHandler

```go
func spaHandler(assets fs.FS) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        f, err := assets.Open("index.html")
        if err != nil {
            http.Error(w, "frontend not built", http.StatusServiceUnavailable)
            return
        }
        defer f.Close()
        info, err := f.Stat()
        if err != nil {
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        rs, ok := f.(io.ReadSeeker)
        if !ok {
            // embed.FS 파일은 ReadSeeker를 구현 — fallback 거의 발생 안 함
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Header().Set("Cache-Control", "no-cache")
        http.ServeContent(w, r, "index.html", info.ModTime(), rs)
    })
}
```

### 3.8 LC-U4-10 PrintLANAddresses

```go
func PrintLANAddresses(port int) {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        fmt.Printf("  (could not detect LAN: %v)\n", err)
        return
    }
    found := 0
    for _, addr := range addrs {
        ipNet, ok := addr.(*net.IPNet)
        if !ok { continue }
        ip := ipNet.IP.To4()
        if ip == nil { continue }
        if ip.IsLoopback() { continue }
        if !ip.IsPrivate() { continue }
        fmt.Printf("  http://%s:%d\n", ip.String(), port)
        found++
    }
    if found == 0 {
        fmt.Printf("  http://localhost:%d\n", port)
    }
}
```

---

## 4. 책임 매트릭스 (NFR ↔ LC)

| NFR Req | 책임 LC |
|---|---|
| NFR-U4-R1 (graceful shutdown) | LC-U4-1 (main의 NotifyContext) + LC-U4-2 (Server.Shutdown) |
| NFR-U4-R5 (embed valid) | LC-U4-11 |
| NFR-U4-P1 (/api/results) | LC-U4-7 |
| NFR-U4-P2 (/assets) | LC-U4-8 |
| NFR-U4-P3 (SPA fallback) | LC-U4-9 |
| NFR-U4-P4 (/healthz) | LC-U4-6 |
| NFR-U4-M1 (커버리지) | 모든 LC가 godoc + 테스트 |
| NFR-U4-M4 (외부 lib 0) | 모든 LC |
| NFR-U4-S1 (Token 미포함) | LC-U4-7 (buildResultsResponse) |
| NFR-U4-S2 (페이로드 미로그) | LC-U4-5 (4필드만) |
| NFR-U4-B1 (단일 바이너리) | LC-U4-1 (main + embed) |

---

## 5. Import Cycle 분석

```
cmd/mafia-game/main.go ──→ internal/game
                       ──→ internal/announce
                       ──→ internal/persistence
                       ──→ internal/session
                       ──→ internal/transport/ws
                       ──→ internal/transport/http (httpx)
                       ──→ github.com/gorilla/websocket (Upgrader 생성)

internal/transport/http ──→ internal/persistence (Store.ListResults)
                        ──→ internal/transport/ws  (Hub.UpgradeHandler)
                        ──→ internal/session       (SessionManager 인터페이스만)
                        ──→ internal/game          (Player, Options, Team, EndReason 타입)
                        ──→ (외부 lib 0)
```

- 어느 단위도 `cmd/mafia-game`을 import 하지 않음 (Composition Root 단방향).
- ws → http 의존 0 (반대만).
- import cycle 없음.

---

## 6. 외부 인프라 / 의존

| 외부 | 사용처 | 비고 |
|---|---|---|
| (없음 — 표준 lib만) | 모든 LC | NFR-U4-M4 |
| `gorilla/websocket` | LC-U4-1 (Upgrader 인스턴스 생성) | U3가 이미 의존 |

---

## 7. 검증 체크리스트

- [x] 모든 LC가 정확히 하나의 패키지에 위치
- [x] Server 인터페이스 2 메서드 (ListenAndServe/Shutdown)
- [x] Config 6 필드 (Addr/Hub/Mgr/Store/Assets/Logger)
- [x] import cycle 없음 (Composition Root 단방향)
- [x] 외부 lib 추가 0 (NFR-U4-M4)
- [x] NFR Req ↔ LC 매트릭스 모두 매핑
- [x] placeholder index.html 명시적 LC로 식별
