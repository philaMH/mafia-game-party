# U5 Web Frontend · Functional Design Patch — Iteration 9

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/inception/requirements/iteration9-bug-safari-reload-requirements.md` v1.0
**Plan**: `aidlc-docs/construction/plans/iteration9-execution-plan.md` v1.0 — Phase A
**Type**: Minimal Patch (단일 hook 라이프사이클 보강 + 신규 단위 테스트)

---

## 1. 변경 의도

iOS Safari 휴대폰 환경에서 새로고침 시 alternation 패턴(1회 실패 → 2회 성공 → 3회 실패) 을 일으키는 두 가지 원인을 한 hook 안에서 동시에 차단:

1. **BFCache 좀비 WebSocket 참조 차단** — `pageshow` 에서 `event.persisted === true` 감지 시 `window.location.reload()` 로 풀 리로드. token resume 으로 게임 진행 상태 자동 복원.
2. **unload 전 동기 close 보장** — `pagehide` 에서 동기 `ws.close(1000, "pagehide")` 호출. iOS 네트워크 스택의 동일 origin WebSocket 직렬화 시 직전 conn 의 TCP FIN 을 unload 이전에 확실히 송출.

부수: onclose race 가드 2건(`abandoned` flag, `wsRef.current === ws` 조건부 nul) 동시에 도입 — 빠른 재연결 / dev StrictMode double-mount 시의 잠재 race 도 함께 차단.

서버측(U1/U2/U3/U4) 변경 없음. wire protocol / token resume 로직 / reconnect backoff 모두 보존.

---

## 2. `useWebSocket` 라이프사이클 변경 표

### 2.1 신설/변경 항목

| 항목 | 종류 | 위치 | 동작 |
|---|---|---|---|
| `abandoned` flag | 신설 | connect closure 지역 변수 | cleanup 후 늦은 close 무시 |
| `pagehide` 리스너 | 신설 | useEffect 본문 | `wsRef.current?.close(1000, "pagehide")` |
| `pageshow` 리스너 | 신설 | useEffect 본문 | `if (e.persisted) window.location.reload()` |
| onclose 첫 가드 | 신설 | `ws.onclose` 안 | `if (abandoned) return;` |
| onclose `wsRef.current = null` | 변경 | `ws.onclose` 안 (현 useWebSocket.ts:68) | `if (wsRef.current === ws) wsRef.current = null;` |
| cleanup 리스너 해제 | 신설 | useEffect return | `removeEventListener("pagehide", onPageHide)` / `removeEventListener("pageshow", onPageShow)` |
| cleanup `abandoned = true` | 신설 | useEffect return | 이전 closures 가 reconnect timer 다시 켜는 것 차단 |

### 2.2 보존 항목 (변경 없음)

| 항목 | 사유 |
|---|---|
| URL dial (`new WebSocket(url)`) | 본 결함 무관 |
| ws.onopen — token resume 자동 송신 | BR-U5-WS-3 — 풀 리로드 후에도 동일 동작으로 게임 복원 |
| ws.onmessage — JSON.parse + dispatch | 본 결함 무관 |
| ws.onclose — backoff (1s/2s/4s/8s/16s) + ws_reconnecting dispatch | NFR-U5-12=A — 본 결함 무관 |
| ws.onerror — no-op | 본 결함 무관 |
| send() — drop-on-not-open | 본 결함 무관 |
| `closedRef.current` 가드 | 기존 cleanup 시 reconnect 차단 메커니즘 — abandoned flag 와 보완적으로 함께 사용 |

---

## 3. 코드 구조 다이어그램

```
useWebSocket(url, dispatch, tokenIO)
  │
  └─ useEffect([url])
       │  closedRef.current = false
       │  let abandoned = false                      ← (신설 FR-4)
       │
       │  function connect() {
       │    if (closedRef.current) return;
       │    dispatch ws_connecting
       │    ws = new WebSocket(url)
       │    wsRef.current = ws
       │    ws.onopen   → ws_open + token? resume
       │    ws.onmessage → ws_message dispatch
       │    ws.onclose  → if (abandoned) return;     ← (신설 FR-4)
       │                  if (wsRef.current === ws)  ← (변경 FR-3)
       │                    wsRef.current = null
       │                  if (closedRef.current) return;
       │                  schedule backoff reconnect
       │    ws.onerror  → no-op
       │  }
       │  connect()
       │
       │  const onPageHide = () => {                 ← (신설 FR-1)
       │    try { wsRef.current?.close(1000,"pagehide") } catch {}
       │  }
       │  const onPageShow = (e) => {                ← (신설 FR-2)
       │    if (e.persisted) window.location.reload()
       │  }
       │  window.addEventListener("pagehide", onPageHide)
       │  window.addEventListener("pageshow", onPageShow)
       │
       └─ return cleanup() {
            abandoned = true                         ← (신설 FR-4)
            closedRef.current = true                 (기존)
            clearTimeout(timerRef.current)           (기존)
            try { wsRef.current?.close() } catch {}  (기존)
            wsRef.current = null                     (기존)
            window.removeEventListener("pagehide", onPageHide)
            window.removeEventListener("pageshow", onPageShow)
          }
```

---

## 4. wire / 서버 변경 없음 명시

| 영역 | 변경 |
|---|---|
| `internal/transport/ws/protocol.go` | 없음 |
| `internal/transport/ws/handlers.go` (`TypeResume` 처리) | 없음 |
| `internal/transport/ws/hub.go` (Upgrader, ping/pong, ReadDeadline, last-connect-wins) | 없음 |
| `internal/transport/http/routes.go` (`spaHandler` Cache-Control) | 없음 |
| `internal/session/session.go` (`ResumePlayer` token 검증) | 없음 |
| `web/src/types/wire.ts` | 없음 |
| `web/src/context/reducer.ts` | 없음 |
| `web/src/hooks/useToken.ts` | 없음 |

서버는 본 패치 적용 후에도 동일 wire 프로토콜로 동작. iOS Safari 풀 리로드 후 클라이언트는 기존 token 으로 자동 resume → 서버는 last-connect-wins 으로 직전 PUBLIC/PLAYER 클라이언트 evict → 신규 conn 에 snapshot 송신.

---

## 5. 신규 / 변경 테스트

### 5.1 신규 — `web/src/hooks/useWebSocket.test.ts`

**공통 setup**
- vitest + jsdom
- `globalThis.WebSocket` 을 fake class 로 stub:
  ```ts
  class FakeWS {
    static instances: FakeWS[] = []
    readyState = 0 // CONNECTING
    static OPEN = 1; static CLOSING = 2; static CLOSED = 3
    onopen?: () => void
    onmessage?: (ev: MessageEvent) => void
    onclose?: () => void
    onerror?: () => void
    send = vi.fn()
    close = vi.fn()
    constructor(public url: string) { FakeWS.instances.push(this) }
    // helpers used by tests:
    fireOpen() { this.readyState = 1; this.onopen?.() }
    fireClose() { this.readyState = 3; this.onclose?.() }
  }
  ```
- `window.location.reload` 를 vi.fn() 으로 mock (기존 location 객체를 Object.defineProperty 로 교체).
- token mock: `tokenIO.get/set/clear` 모두 vi.fn(). 기본 get → null.

**케이스**

| ID | 시나리오 | 단언 |
|---|---|---|
| **I9-W1** | hook mount 후 `window.dispatchEvent(new Event("pagehide"))` 1회 | `FakeWS.instances[0].close` 가 `(1000, "pagehide")` 인자로 정확히 1회 호출됨 |
| **I9-W2** | hook mount 후 `pageshow` 이벤트 (`event.persisted = true`) dispatch | `window.location.reload` 가 정확히 1회 호출됨 |
| **I9-W3** | hook mount 후 `pageshow` 이벤트 (`event.persisted = false`) dispatch | `window.location.reload` 가 호출되지 않음 |
| **I9-W4** | mount → ws_A 생성 → fireClose(ws_A) 동기 호출 후 1.1s timer fast-forward → ws_B 생성됨 → fireClose(ws_A) 다시 호출 (지연된 close 시뮬레이션) | `wsRef` 는 ws_B 를 유지(send 호출 시 ws_B.send 가 호출됨), ws_A 의 추가 close 가 ws_B 를 덮어쓰지 않음 |
| **I9-W5** | mount → ws 생성 → unmount(=cleanup) → fireClose(ws) 호출 | 추가 reconnect timer 가 스케줄되지 않음 (vi.useFakeTimers 로 1.1s advance 후에도 새 FakeWS 인스턴스 생성되지 않음을 단언) |

**참고: I9-W4 의 send 호출 검증** — `useWebSocket` 의 `send` 가 외부로 노출되므로, hook 의 `result.current.send({ type: "join", name: "x" })` 호출 시 ws_B.send 가 1회 호출되고 ws_A.send 는 호출되지 않음.

### 5.2 회귀 — 기존 테스트

- `npm test` 66 케이스 → 71 케이스 (신규 5건 추가) PASS.
- `useWebSocket.ts` 직접 단위 테스트는 그동안 0건이었음 — 본 5건이 첫 단위 커버.
- reducer / Picker / view 레벨 테스트는 변경 없음.

---

## 6. 영향 받는 파일

| 파일 | 변경 종류 | 라인 추정 |
|---|---|---|
| `web/src/hooks/useWebSocket.ts` | 수정 (라이프사이클 보강) | +20 |
| `web/src/hooks/useWebSocket.test.ts` | **신규** (5 케이스) | +130 |

전체 변경 라인: ~150. 의존성 추가 없음.

---

## 7. 사용자 체감 흐름

### 7.1 정상 풀 리로드 (Safari 새로고침 — BFCache 미사용)
| 단계 | 동작 |
|---|---|
| 1 | 사용자가 새로고침 탭. 현재 페이지 `pagehide` 발화 → 본 패치가 `ws.close(1000, "pagehide")` 동기 호출 → TCP FIN 송출 |
| 2 | 서버 last-connect-wins 으로 직전 client evict 준비 |
| 3 | 새 페이지 로드 → JS bootstrap → `useWebSocket` 마운트 → 신규 WS 연결 |
| 4 | `ws.onopen` → token 있으면 `{ type: "resume", token }` 송신 → 서버 ResumePlayer → joined + snapshot 수신 |
| 5 | UI: 게임 진행 중이었다면 자동 복원, 아니면 join 폼 |

### 7.2 BFCache 복원 (Safari 가 BFCache 에서 페이지 가져옴)
| 단계 | 동작 |
|---|---|
| 1 | iOS Safari 가 페이지를 BFCache 에서 복원 → JS state 보존 → `pageshow` 발화 with `event.persisted === true` |
| 2 | 본 패치가 `window.location.reload()` 호출 → 페이지가 풀 리로드로 전환 |
| 3 | 풀 리로드 후 7.1 단계로 동일 진행 (token resume 자동 복원) |

### 7.3 일반 사용자 시점
- 새로고침 후 "연결 중" 표기가 빠르게 "연결됨" 으로 전환 (alternation 패턴 해소).
- BFCache 복원 시 약 0.5~1.5s 의 "다시 로드되는 듯한" 잠시 보임 — token resume 으로 게임 상태/입력 폼은 그대로 복원.

---

## 8. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-30 | 최초 작성, RA v1.0 / Plan v1.0 사용자 승인 후 |
