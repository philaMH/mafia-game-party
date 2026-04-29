# Tech Stack Decisions — U4 HTTP Bootstrap & Static

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-requirements.md`, `requirements.md` v1.1 NFR-7

---

## 1. 외부 직접 의존

| 라이브러리 | 결정 |
|---|---|
| (없음) | U4는 **외부 직접 의존 0개** (Q-NFR-U4-1=A). |

> 누계 외부 직접 의존: U2의 `modernc.org/sqlite` + U3의 `gorilla/websocket` + (U4의 0개) = **2개**. NFR-7 외부 서비스 0 정책 만족.

---

## 2. Go 표준 라이브러리

| 패키지 | 사용처 |
|---|---|
| `net/http` | http.Server + ServeMux + http.HandlerFunc |
| `embed` | `//go:embed all:web/dist` 정적 자산 동봉 |
| `io/fs` | `fs.FS` + `fs.Sub` (embed 자식 디렉터리) |
| `net` | `net.InterfaceAddrs` (LAN IP 검색) |
| `os` | os.Args, os.Exit, os.Stderr |
| `os/signal` | Notify(SIGINT, SIGTERM) |
| `syscall` | SIGINT, SIGTERM 상수 |
| `flag` | CLI 플래그 파싱 |
| `context` | shutdown / 핸들러 cancellation |
| `encoding/json` | /api/results 직렬화 |
| `errors` | errors.Is(http.ErrServerClosed) |
| `fmt` | LAN IP 출력 |
| `log/slog` | 구조화 로깅 |
| `mime` | 정적 자산 MIME 자동 판별 (FileServerFS 내부) |
| `strconv` | limit 파라미터 파싱 |
| `time` | 타임아웃, 로그 duration |

---

## 3. 패키지 / 파일 레이아웃 (확정)

```
cmd/mafia-game/
├── main.go              # Composition Root: 모든 단위 와이어링 + 시그널 + LAN 출력
└── (web/dist embed assets)

internal/transport/http/
├── doc.go               # 패키지 godoc (alias 'httpx' for stdlib 충돌 회피)
├── server.go            # Server 인터페이스 + impl + New + ListenAndServe + Shutdown
├── routes.go            # ServeMux 등록 + healthHandler + spaHandler + assetsHandler
├── api_results.go       # /api/results JSON 핸들러
├── lan.go               # PrintLANAddresses
├── middleware.go        # logging middleware + statusRecorder
└── *_test.go            # httptest 기반 단위 테스트

web/
└── dist/
    ├── index.html       # placeholder (commit) — npm run build 시 덮어씀
    └── (Vite 빌드 산출물 — gitignore)
```

**Composition Root**:

```go
// cmd/mafia-game/main.go (요약)
//go:embed all:web/dist
var webDist embed.FS

func main() {
    // ... flag.Parse(), slog 초기화, persistence/engine/announce/session/ws 생성 ...
    assets, _ := fs.Sub(webDist, "web/dist")
    srv, _ := httpx.New(httpx.Config{
        Addr: fmt.Sprintf("0.0.0.0:%d", port),
        Hub: hub, Mgr: mgr, Store: store, Assets: assets, Logger: log,
    })
    // ListenAndServe + signal handler + graceful shutdown
}
```

---

## 4. 의존 그래프

```
cmd/mafia-game/main.go
  ├── internal/game           (NewDefault, NewDefaultKeywordPool)
  ├── internal/announce       (NewDefaultCatalog)
  ├── internal/persistence    (OpenSqlite)
  ├── internal/session        (New)
  ├── internal/transport/ws   (New, Hub.UpgradeHandler)
  ├── internal/transport/http (New)
  └── github.com/gorilla/websocket  (Upgrader 생성)

internal/transport/http (httpx)
  ├── internal/persistence    (Store.ListResults — /api/results)
  ├── internal/transport/ws   (Hub.UpgradeHandler — /ws)
  ├── internal/session        (interface 참조 only — 미래 확장 hook)
  └── (외부 lib 0)
```

> 직접 의존: `cmd/mafia-game`이 모든 단위를 import. `internal/transport/http`는 `persistence`, `ws` import만 (mgr는 인터페이스 인자만 받음).

---

## 5. 빌드 / 실행 가정

| 항목 | 결정 |
|---|---|
| Go 버전 | 1.25.0 (U2가 갱신) |
| OS | macOS / Linux / Windows (모두 표준 lib만 사용) |
| 컴파일 | `go build ./cmd/mafia-game` → `mafia-game` 단일 바이너리 |
| 실행 | `./mafia-game --port 8080 --db ./data/mafia.db` |
| 프론트엔드 | `cd web && npm run build` (별도 단계, U5에서 작성) |
| placeholder | `web/dist/index.html`을 git에 commit하여 백엔드만 빌드해도 통과 |

---

## 6. 미결정 / 후속 결정 사항

| 항목 | 결정 시점 |
|---|---|
| `web/dist`의 정확한 file structure | U5 (Vite 설정 시 확정) |
| `/api/results` 페이지네이션 cursor | 운영 단계에서 필요 시 (현재 단순 limit) |
| 추가 API 엔드포인트 (예: `/api/healthz/db`) | 운영 단계 |
| Prometheus metrics | NFR-U4 비-요구사항 |

---

## 7. 검증 체크리스트

- [x] 외부 직접 의존 0개 (NFR-U4-M4)
- [x] 표준 lib 15종 사용 (모두 Go 표준)
- [x] 패키지 레이아웃 정의 (`cmd/mafia-game/` + `internal/transport/http/*`)
- [x] 의존 그래프 — Composition Root는 main.go 단일
- [x] placeholder index.html commit 정책 — 빌드 보장
- [x] 후속 결정 사항 명시
