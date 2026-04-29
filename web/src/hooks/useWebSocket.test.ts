import { renderHook, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useWebSocket } from "./useWebSocket";

class FakeWS {
  static instances: FakeWS[] = [];
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = FakeWS.CONNECTING;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
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
  get: vi.fn<() => string | null>(() => null),
  set: vi.fn<(token: string) => void>(),
  clear: vi.fn<() => void>(),
};

describe("useWebSocket — iOS Safari reload lifecycle (Iteration 9)", () => {
  let reloadMock: ReturnType<typeof vi.fn>;
  let originalLocation: Location;

  beforeEach(() => {
    FakeWS.instances = [];
    dispatch.mockReset();
    tokenIO.get.mockReset().mockReturnValue(null);
    tokenIO.set.mockReset();
    tokenIO.clear.mockReset();

    vi.stubGlobal("WebSocket", FakeWS as unknown as typeof WebSocket);

    // jsdom marks `window.location` non-configurable, so swap the entire
    // object for one whose `reload` is a vi.fn(). Restored in afterEach.
    originalLocation = window.location;
    reloadMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, reload: reloadMock },
    });

    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
  });

  it("I9-W1: dispatches close(1000, 'pagehide') on window pagehide event", () => {
    renderHook(() => useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }));
    expect(FakeWS.instances).toHaveLength(1);
    const ws = FakeWS.instances[0]!;

    act(() => {
      window.dispatchEvent(new Event("pagehide"));
    });

    expect(ws.close).toHaveBeenCalledTimes(1);
    expect(ws.close).toHaveBeenCalledWith(1000, "pagehide");
  });

  it("I9-W2: forces window.location.reload on BFCache restore (persisted=true)", () => {
    renderHook(() => useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }));

    act(() => {
      const ev = new Event("pageshow");
      Object.defineProperty(ev, "persisted", { value: true });
      window.dispatchEvent(ev);
    });

    expect(reloadMock).toHaveBeenCalledTimes(1);
  });

  it("I9-W3: skips reload on normal pageshow (persisted=false)", () => {
    renderHook(() => useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }));

    act(() => {
      const ev = new Event("pageshow");
      Object.defineProperty(ev, "persisted", { value: false });
      window.dispatchEvent(ev);
    });

    expect(reloadMock).not.toHaveBeenCalled();
  });

  it("I9-W4: late onclose from prior conn does not redirect new wsRef (race guard)", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
    );

    expect(FakeWS.instances).toHaveLength(1);
    const wsA = FakeWS.instances[0]!;

    // First close — schedules backoff reconnect (1000 ms).
    act(() => {
      wsA.fireClose();
    });

    // Advance past the first backoff so connect() runs again, creating wsB.
    act(() => {
      vi.advanceTimersByTime(1100);
    });

    expect(FakeWS.instances).toHaveLength(2);
    const wsB = FakeWS.instances[1]!;

    act(() => {
      wsB.fireOpen();
    });

    // Late delayed close from wsA arrives AFTER wsB took over.
    act(() => {
      wsA.fireClose();
    });

    // send() must still route through wsB.
    result.current.send({ type: "host:claim" });
    expect(wsB.send).toHaveBeenCalledTimes(1);
    expect(wsA.send).not.toHaveBeenCalled();
  });

  it("I9-W5: late onclose after unmount does not schedule a reconnect timer", () => {
    const { unmount } = renderHook(() =>
      useWebSocket({ url: "ws://test/ws", dispatch, tokenIO }),
    );

    expect(FakeWS.instances).toHaveLength(1);
    const ws = FakeWS.instances[0]!;

    unmount();

    act(() => {
      ws.fireClose();
    });

    act(() => {
      vi.advanceTimersByTime(20_000);
    });

    expect(FakeWS.instances).toHaveLength(1);
  });
});
