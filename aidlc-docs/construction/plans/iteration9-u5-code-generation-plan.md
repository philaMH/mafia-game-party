# U5 Web Frontend · Code Generation Plan — Iteration 9

**Status**: Draft v1.0 — 사용자 승인 대기
**Source**: `aidlc-docs/construction/u5-web-frontend/functional-design/iteration9-patch.md` v1.0 (사용자 승인 2026-04-30T00:05Z)
**Type**: Bug Fix Minimal Patch (단일 hook 라이프사이클 보강 + 신규 단위 테스트 5건)

---

## 1. Step 개요

```
Step A — useWebSocket.ts:    connect closure 안 abandoned flag 도입
Step B — useWebSocket.ts:    onclose race 가드 2건 (abandoned + wsRef === ws)
Step C — useWebSocket.ts:    pagehide 리스너 등록/해제
Step D — useWebSocket.ts:    pageshow 리스너 등록/해제 + persisted 분기
Step E — useWebSocket.test.ts (신규): I9-W1~W5 5 케이스
Step F — 검증: typecheck / test / build / go build
Step G — audit.md / aidlc-state.md 동기화
```

---

## 2. Step A — `useWebSocket.ts` abandoned flag

기존 `useEffect(() => { ... }, [url])` 본문 첫 줄에 지역 flag 추가:

```ts
useEffect(() => {
  closedRef.current = false;
  let abandoned = false;                              // (Step A)

  const connect = (): void => { ... };
  connect();

  // (Step C/D 의 리스너 등록은 connect() 호출 이후에 추가)

  return () => {
    abandoned = true;                                 // (Step A — 가장 먼저)
    closedRef.current = true;
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (wsRef.current) {
      try { wsRef.current.close(); } catch {}
      wsRef.current = null;
    }
    // (Step C/D 의 removeEventListener 도 여기에 추가)
  };
}, [url]);
```

### 체크리스트
- [ ] A.1 useEffect 본문 첫 줄 (closedRef 직후) `let abandoned = false` 추가
- [ ] A.2 cleanup 첫 줄 `abandoned = true` 추가 (closedRef 처리보다 앞)

---

## 3. Step B — onclose race 가드 2건

기존 `useWebSocket.ts:67-75` 의 onclose 를 다음과 같이 수정:

```ts
ws.onclose = () => {
  if (abandoned) return;                              // (Step B.1 — FR-4)
  if (wsRef.current === ws) {                         // (Step B.2 — FR-3)
    wsRef.current = null;
  }
  if (closedRef.current) return;
  const idx = Math.min(attemptRef.current, BACKOFF_MS.length - 1);
  const delay = BACKOFF_MS[idx] ?? 16000;
  attemptRef.current++;
  dispatch({ type: "ws_reconnecting" });
  timerRef.current = setTimeout(connect, delay);
};
```

### 체크리스트
- [ ] B.1 onclose 첫 줄 `if (abandoned) return;` 추가
- [ ] B.2 무조건 `wsRef.current = null` → `if (wsRef.current === ws) wsRef.current = null;` 변경
- [ ] B.3 기존 `if (closedRef.current) return;`, backoff 로직, dispatch ws_reconnecting, timer 설정 모두 보존

---

## 4. Step C — `pagehide` 리스너

`connect()` 호출 직후, return 이전 위치에 추가:

```ts
const onPageHide = () => {
  // iOS Safari 가 unload 전에 동기 close 호출 보장 — 직전 conn 이
  // 새 페이지의 WS open 을 직렬화로 막지 않도록 TCP FIN 강제 송출.
  const ws = wsRef.current;
  if (ws) {
    try { ws.close(1000, "pagehide"); } catch {
      // ignore — already closing/closed
    }
  }
};
window.addEventListener("pagehide", onPageHide);
```

cleanup 에 해제 추가:
```ts
window.removeEventListener("pagehide", onPageHide);
```

### 체크리스트
- [ ] C.1 `onPageHide` 함수 정의 + try/catch
- [ ] C.2 `window.addEventListener("pagehide", onPageHide)` 등록
- [ ] C.3 cleanup 에 `window.removeEventListener("pagehide", onPageHide)` 추가
- [ ] C.4 의도 코멘트 1줄 (한국어 또는 영어 — 기존 코멘트 톤 일치)

---

## 5. Step D — `pageshow` 리스너 + persisted 분기

`onPageHide` 인접 위치에 추가:

```ts
const onPageShow = (e: PageTransitionEvent) => {
  // BFCache 복원 감지 — iOS Safari 는 pageshow.persisted=true 로
  // 복원된 페이지의 좀비 WebSocket 참조를 연결됐다고 오인하므로,
  // 풀 리로드로 강제 재진입. token resume 으로 게임 상태 자동 복원.
  if (e.persisted) {
    window.location.reload();
  }
};
window.addEventListener("pageshow", onPageShow);
```

cleanup 에 해제 추가:
```ts
window.removeEventListener("pageshow", onPageShow);
```

### 체크리스트
- [ ] D.1 `onPageShow(e: PageTransitionEvent)` 정의 + persisted 분기
- [ ] D.2 `window.addEventListener("pageshow", onPageShow)` 등록
- [ ] D.3 cleanup 에 `window.removeEventListener("pageshow", onPageShow)` 추가
- [ ] D.4 의도 코멘트

---

## 6. Step E — `useWebSocket.test.ts` (신규)

`web/src/hooks/useWebSocket.test.ts` 신규 파일.

### 6.1 setup 공통 부분

```ts
import { renderHook, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useWebSocket } from "./useWebSocket";

// Fake WebSocket — captures lifecycle callbacks and exposes manual fire helpers
class FakeWS {
  static instances: FakeWS[] = [];
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = FakeWS.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn<(data: string) => void>();
  close = vi.fn<(code?: number, reason?: string) => void>();

  constructor(public url: string) {
    FakeWS.instances.push(this);
  }

  fireOpen() {
    this.readyState = FakeWS.OPEN;
    this.onopen?.();
  }
  fireClose() {
    this.readyState = FakeWS.CLOSED;
    this.onclose?.();
  }
}

const dispatch = vi.fn();
const tokenIO = {
  get: vi.fn(() => null),
  set: vi.fn(),
  clear: vi.fn(),
};

let originalReload: () => void;

beforeEach(() => {
  FakeWS.instances = [];
  dispatch.mockReset();
  tokenIO.get.mockReset().mockReturnValue(null);
  tokenIO.set.mockReset();
  tokenIO.clear.mockReset();

  // Stub global WebSocket
  vi.stubGlobal("WebSocket", FakeWS as unknown as typeof WebSocket);

  // Mock window.location.reload safely
  originalReload = window.location.reload;
  Object.defineProperty(window.location, "reload", {
    configurable: true,
    value: vi.fn(),
  });

  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  Object.defineProperty(window.location, "reload", {
    configurable: true,
    value: originalReload,
  });
});
```

### 6.2 I9-W1 — pagehide 시 close(1000, "pagehide")

```ts
it("I9-W1: dispatches close(1000, 'pagehide') on window pagehide event", () => {
  renderHook(() =>
    useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
  );
  expect(FakeWS.instances).toHaveLength(1);
  const ws = FakeWS.instances[0];

  act(() => {
    window.dispatchEvent(new Event("pagehide"));
  });

  expect(ws.close).toHaveBeenCalledTimes(1);
  expect(ws.close).toHaveBeenCalledWith(1000, "pagehide");
});
```

### 6.3 I9-W2 — pageshow with persisted=true triggers reload

```ts
it("I9-W2: forces window.location.reload on BFCache restore (persisted=true)", () => {
  renderHook(() =>
    useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
  );

  act(() => {
    const ev = new Event("pageshow") as PageTransitionEvent;
    Object.defineProperty(ev, "persisted", { value: true });
    window.dispatchEvent(ev);
  });

  expect(window.location.reload).toHaveBeenCalledTimes(1);
});
```

### 6.4 I9-W3 — pageshow with persisted=false does NOT reload

```ts
it("I9-W3: skips reload on normal pageshow (persisted=false)", () => {
  renderHook(() =>
    useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
  );

  act(() => {
    const ev = new Event("pageshow") as PageTransitionEvent;
    Object.defineProperty(ev, "persisted", { value: false });
    window.dispatchEvent(ev);
  });

  expect(window.location.reload).not.toHaveBeenCalled();
});
```

### 6.5 I9-W4 — late onclose does not overwrite new wsRef

```ts
it("I9-W4: late onclose from prior conn does not null out new wsRef (race guard)", () => {
  const { result } = renderHook(() =>
    useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
  );

  expect(FakeWS.instances).toHaveLength(1);
  const wsA = FakeWS.instances[0];

  // First close — schedules backoff reconnect (1000ms)
  act(() => {
    wsA.fireClose();
  });

  // Advance backoff timer — connect() runs again, creating wsB
  act(() => {
    vi.advanceTimersByTime(1100);
  });

  expect(FakeWS.instances).toHaveLength(2);
  const wsB = FakeWS.instances[1];

  // Open wsB so send() routes through it
  act(() => {
    wsB.fireOpen();
  });

  // Late delayed close from wsA arrives AFTER wsB took over
  act(() => {
    wsA.fireClose();
  });

  // send() should still go to wsB, not be silently dropped
  result.current.send({ type: "host:claim" });
  expect(wsB.send).toHaveBeenCalledTimes(1);
  expect(wsA.send).not.toHaveBeenCalled();
});
```

### 6.6 I9-W5 — cleanup blocks late onclose from scheduling reconnect

```ts
it("I9-W5: late onclose after unmount does not schedule a reconnect timer", () => {
  const { unmount } = renderHook(() =>
    useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
  );

  expect(FakeWS.instances).toHaveLength(1);
  const ws = FakeWS.instances[0];

  unmount();

  // Late close arrives after cleanup — must NOT schedule a reconnect
  act(() => {
    ws.fireClose();
  });

  // Advance well past max backoff to confirm no new conn was created
  act(() => {
    vi.advanceTimersByTime(20_000);
  });

  expect(FakeWS.instances).toHaveLength(1);
});
```

### 체크리스트
- [ ] E.1 새 파일 `web/src/hooks/useWebSocket.test.ts` 생성
- [ ] E.2 setup 공통(FakeWS, dispatch/tokenIO mock, location.reload mock, fake timers) 작성
- [ ] E.3 I9-W1 (pagehide → close)
- [ ] E.4 I9-W2 (pageshow.persisted=true → reload)
- [ ] E.5 I9-W3 (pageshow.persisted=false → no reload)
- [ ] E.6 I9-W4 (late close 후 wsRef === ws 가드 동작)
- [ ] E.7 I9-W5 (cleanup 후 abandoned flag 동작)

---

## 7. Step F — 검증

- [ ] F.1 `cd web && npm run typecheck` PASS
- [ ] F.2 `cd web && npm test` PASS (66 → 71)
- [ ] F.3 `cd web && npm run build` 성공, JS gzip 측정 (baseline 65.62 KB ±1 KB)
- [ ] F.4 `go build -o /tmp/mafia-game-iter9 ./cmd/mafia-game` 성공 (정적 자산 임베드 갱신, baseline 17.97 MB ±0.05 MB)
- [ ] F.5 `go test ./... -count=1 -race` 6 패키지 PASS (서버 변경 없음, 회귀 확인)

---

## 8. Step G — 동기화

- [ ] G.1 audit.md — Code Generation 실행 결과 추가 (timestamp / 파일 / 변경 라인 / 검증 결과)
- [ ] G.2 aidlc-state.md — Iteration 9 U5 Code Generation 체크박스 [x]
- [ ] G.3 사용자 승인 게이트 (2-옵션): "Continue to Next Stage" → Phase D Build & Test / "Request Changes" → 동일 Phase 보정

---

## 9. 영향 받는 파일

| 파일 | 종류 | 라인 변동 |
|---|---|---|
| `web/src/hooks/useWebSocket.ts` | 수정 | +20 |
| `web/src/hooks/useWebSocket.test.ts` | 신규 | +130 |
| **합계** | | **+150** |

---

## 10. RISK

| RISK | 완화책 |
|---|---|
| jsdom 에 `WebSocket` 글로벌이 없음 | `vi.stubGlobal("WebSocket", FakeWS)` 로 주입 |
| `window.location.reload` 가 jsdom 에서 navigation 트리거 | `Object.defineProperty(window.location, "reload", { configurable: true, value: vi.fn() })` 로 교체, afterEach 에서 복원 |
| `PageTransitionEvent.persisted` 가 jsdom 에서 자동 false | 일반 `Event("pageshow")` 생성 후 `Object.defineProperty(ev, "persisted", { value: true })` 로 주입 |
| StrictMode dev double-mount 와 abandoned flag 의 의도치 않은 상호작용 | abandoned 는 useEffect closure scope — mount 별 독립. cleanup 후 다음 mount 의 connect 는 새 closure → 영향 없음. I9-W5 회귀로 검증 |
| `result.current.send` 호출 시 readyState 검증 | I9-W4 에서 wsB.fireOpen() 호출 후 send 하므로 readyState=OPEN 보장 |
| pagehide 시 close 가 send buffer 에 있는 message 를 끊는 부수효과 | code 1000 (Normal Closure) 사용 — 서버는 graceful close 처리, 토큰 무효화 없음 |
| BFCache 무한 루프 | reload 는 `event.persisted === true` 케이스에서만 호출. reload 후 fresh navigation 이므로 다음 pageshow 는 persisted=false → 무한 루프 없음 |

---

## 11. 변경 이력

| 버전 | 일자 | 변경 |
|---|---|---|
| v1.0 | 2026-04-30 | 최초 작성 |
