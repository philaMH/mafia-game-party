# Tech Stack Decisions — U3 Realtime Transport

**작성일**: 2026-04-26
**문서 버전**: 1.0
**참조**: `nfr-requirements.md`, `requirements.md` v1.1 NFR-7

---

## 1. 외부 직접 의존

| 라이브러리 | 버전 | 사용처 | 결정 근거 |
|---|---|---|---|
| `github.com/gorilla/websocket` | v1.x (latest) | 모든 WS 통신 | Q-AD-2=A: Go 생태계 표준, 풍부한 예제, 안정적 RFC 6455 지원, single-writer 패턴 명확. NFR-7의 외부 서비스 0 정책 만족 (외부 서버 없음, 단순 lib). |

> 다른 외부 lib 추가 금지 (NFR-U3-M4). transitive 의존이 추가되면 직접 의존 1개 정책 유지 여부 검토.

---

## 2. Go 표준 라이브러리

| 패키지 | 사용처 |
|---|---|
| `net/http` | WebSocket Upgrader (gorilla가 사용) |
| `encoding/json` | 와이어 메시지 직렬화/역직렬화 (NFR-U3-M6 결정성) |
| `log/slog` | 구조화 로깅 (Go 1.21+) |
| `sync` | Hub의 ClientRegistry mutex |
| `time` | Read/Write deadline, ping ticker |
| `context` | 메서드 cancellation propagation |
| `crypto/rand` + `encoding/hex` | ClientID 생성 (8-byte hex16) |

---

## 3. 패키지 레이아웃 (확정)

```
internal/transport/
└── ws/
    ├── doc.go              # 패키지 godoc
    ├── hub.go              # Hub 인터페이스 + impl, Run/Close
    ├── client.go           # Client struct, ClientRegistry
    ├── handlers.go         # readLoop, writeLoop, handleIncoming
    ├── dispatch.go         # onEvent, routeEvent, enqueue
    ├── protocol.go         # 와이어 메시지 타입(incoming + outgoing) — 단일 진실 소스 (NFR-U3-M5)
    ├── id.go               # newClientID
    └── *_test.go
```

**Composition Root** (U4 단계에서 작성):

```go
hub := ws.New(websocket.Upgrader{...}, mgr, slog.Default())
go hub.Run(ctx)
http.Handle("/ws", hub.UpgradeHandler())
```

---

## 4. 의존 그래프

```
internal/transport/ws ──→ internal/session   (SessionManager 호출 + Subscribe)
                      ──→ internal/announce  (Announcement 타입 사용)
                      ──→ internal/game      (Action/Event/State/EngineError 타입 사용)
                      ──→ github.com/gorilla/websocket (외부)
                      ──→ net/http, encoding/json, log/slog 등 (표준)

internal/transport/ws ✗ internal/persistence (직접 의존 안 함 — SessionManager 경유)
```

---

## 5. JSON 타입 정의 위치

- 모든 wire 메시지 struct는 `internal/transport/ws/protocol.go`에 집중.
- `internal/game.State` 등 도메인 타입은 그대로 import해 wire payload에 임베드 (별도 wire-only 타입 미정의).
- 시간 필드는 `time.Time` JSON 직렬화 → ISO 8601 (encoding/json default). `Deadline`은 별도 `int64` epoch ms 필드로 변환해 송신 (BR-U3-WIRE-4).

---

## 6. 빌드·실행 가정

| 항목 | 결정 |
|---|---|
| Go 버전 | 1.25.0 (U2가 갱신함) |
| OS | macOS / Linux / Windows (gorilla/websocket pure Go) |
| 컴파일 | `go build ./cmd/mafia-game` (U4 단계에서 main.go 작성) |
| 실행 | 단일 바이너리 + `data/mafia.db` (U2가 자동 생성) |

---

## 7. 미결정 / 후속 결정 사항

| 항목 | 결정 시점 |
|---|---|
| `Snapshot() game.State`를 SessionManager 인터페이스에 추가할지 | Code Generation (U3) — VisRoleMafia 라우팅에 필요. U2 인터페이스 확장. |
| net.Pipe 기반 in-memory WS 테스트 헬퍼 | Code Generation (U3) — `*websocket.Conn`을 pipe로 만드는 보조 함수 |
| `UpgradeHandler() http.Handler` API | NFR Design (U3) — Hub가 자체 핸들러 제공 vs U4가 직접 Upgrade. 후자 권장. |

---

## 8. 검증 체크리스트

- [x] 외부 직접 의존 1개 (gorilla/websocket)
- [x] 표준 lib만으로 메시지 인코딩·로깅·타이머·ID 생성
- [x] 패키지 레이아웃 정의 (`internal/transport/ws/*`)
- [x] 의존 그래프 — persistence는 직접 의존 안 함
- [x] wire 타입은 `protocol.go` 1개 파일에 집중 (NFR-U3-M5)
- [x] 결정적 직렬화 (encoding/json default)
- [x] 후속 결정 사항 명시 (SessionManager.Snapshot 추가 등)
