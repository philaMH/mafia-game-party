# U4 — Public API Catalog

**작성일**: 2026-04-26
**대상 패키지**: `github.com/saltware/mafia-game/internal/transport/http` (alias `httpx`) + `cmd/mafia-game`
**버전**: 1.0 (Code Generation 1차 산출물)

본 문서는 U4가 외부에 노출하는 **공개 API**와 **HTTP 라우트**를 요약합니다. godoc 주석은 소스 파일에서 직접 확인 가능합니다.

---

## 1. `httpx` 패키지 — Server 인터페이스

```go
type Server interface {
    ListenAndServe() error
    Shutdown(ctx context.Context) error
}

type Config struct {
    Addr   string                       // "0.0.0.0:8080" 등
    Hub    ws.Hub
    Store  persistence.PersistenceStore
    Assets fs.FS                        // embed.FS의 web/dist
    Logger *slog.Logger                 // nil → slog.Default()
}

func New(cfg Config) (Server, error)
func PrintLANAddresses(w io.Writer, port int)
```

**보장 사항**:
- `Addr`/`Hub`/`Store`/`Assets` 누락 시 `New`가 error 반환.
- 모든 핸들러는 `loggingMiddleware`로 래핑되어 method/path/status/duration_ms 4필드 INFO 로그.
- payload (query value, body, header) 미기록 — NFR-U4-S2.

---

## 2. HTTP 라우팅 카탈로그

| 패턴 | 메서드 | 응답 | 비고 |
|---|---|---|---|
| `/healthz` | GET | 200 + `"ok"` (text/plain) | 헬스 프로브 |
| `/ws` | GET | WebSocket Upgrade | U3 Hub.UpgradeHandler 위임 |
| `/api/results` | GET | `{results: [...]}` JSON | FR-6.3, `?limit=N` (1~500, 기본 50) |
| `/assets/{path...}` | GET | 정적 파일 + `Cache-Control: public, max-age=31536000, immutable` | Vite hash 파일명 가정 |
| `/`, `/play`, `/play/...`, `/public`, ... | GET | `index.html` + `Cache-Control: no-cache` | SPA fallback |

**모든 응답**:
- `Content-Type` 자동 판별 (text/plain, text/html, application/json, application/javascript 등)
- charset=utf-8

**HTTP method 매칭**: 모든 라우트는 GET만 허용. POST 등은 405.

---

## 3. `/api/results` 응답 스키마

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
      "options":   { "mafiaCount": 1, "introSecondsPerPlayer": 20, "discussionSeconds": 180, "doctorSelfHealAllowed": true, "announcementVoiceOn": true },
      "members":   [{"id": "...", "name": "철수", "joinedAt": "..."}],
      "reveal":    [{"id": "...", "name": "철수", "alive": false, "role": "MAFIA"}]
    }
  ]
}
```

> ⚠️ **`members[].token` 필드는 의도적으로 제외**됨 (NFR-U4-S1). 외부 API에 토큰을 노출하지 않음.

### 응답 헤더
```
Content-Type: application/json; charset=utf-8
Cache-Control: no-store
```

### 에러 코드
- `400 Bad Request`: `limit` 값이 1~500 범위 밖 또는 정수가 아님
- `500 Internal Server Error`: DB ListResults 실패

---

## 4. CLI / 환경변수

### 플래그 / 환경변수 우선순위

플래그 > 환경변수 > 기본값.

| 플래그 | 환경변수 | 기본 | 설명 |
|---|---|---|---|
| `--port` | `MAFIA_PORT` | `8080` | HTTP 리슨 포트 (1~65535) |
| `--db` | `MAFIA_DB_PATH` | `./data/mafia.db` | SQLite 파일 경로 |
| `--log-level` | `MAFIA_LOG_LEVEL` | `info` | slog 레벨 (debug/info/warn/error) |

### 실행 예시

```bash
# 기본
./mafia-game

# 커스텀 포트
./mafia-game --port 9000

# 환경변수
MAFIA_PORT=9000 MAFIA_DB_PATH=/tmp/mafia.db ./mafia-game

# 디버그 로그
./mafia-game --log-level debug
```

---

## 5. 부팅 시퀀스 (Composition Root)

```
1. flag.Parse + env 폴백
2. slog 로거 초기화
3. signal.NotifyContext(SIGINT, SIGTERM)
4. persistence.OpenSqlite (자동 PRAGMA + 0600 chmod)
5. game.NewDefault (engine + keyword pool)
6. announce.NewDefaultCatalog
7. session.New (자동 복원 포함)
8. websocket.Upgrader (CheckOrigin → true)
9. ws.New (Subscribe to mgr)
10. fs.Sub(webDist, "web/dist")
11. httpx.New (Config)
12. PrintLANAddresses → stdout
13. ListenAndServe (goroutine)
14. select { errCh | ctx.Done }
15. shutdown(srv 5s, hub, mgr 2s) → exit
```

---

## 6. graceful shutdown 보장

| 단계 | 한도 |
|---|---|
| http.Server.Shutdown | 5초 |
| hub.Close | 즉시 (cancel + 모든 client close) |
| mgr.Close (마지막 SaveSnapshot 포함) | 2초 |
| **전체 budget** | **~7초** |

두 번째 시그널은 Go runtime이 즉시 프로세스 종료 (signal handler가 `stop()` 후 자동 해제).

---

## 7. 변경 영향도

- **HTTP 라우트 추가**는 안전 (ServeMux 이용).
- **응답 스키마 필드 추가**는 안전 (클라이언트가 모르는 필드 무시).
- **`members[]`에 token 추가는 NFR-4 위반** — 절대 추가 금지.
- **wire 포맷 호환성**: U5 React SPA가 `members[*].token`을 기대하지 않도록 작성되어야 함.

---

## 8. 와이어링 — 외부 호출자 가정

본 단위는 **다른 단위의 호출 대상이 아님** — Composition Root 자체이므로 main.go에서 단방향으로 호출. 단위 테스트와 통합 테스트만 외부 진입점.
