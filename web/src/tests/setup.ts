import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

// Iter7 — host PublicView plays pre-recorded /audio/<id>.mp3 via
// HTMLAudioElement. jsdom doesn't implement playback, so we stub the
// surface used by useAudioCueQueue: src setter, play(), pause(), and the
// lifecycle events ('ended', 'error').
class FakeAudio {
  src = "";
  currentTime = 0;
  paused = true;
  private listeners: Record<string, Array<() => void>> = {};

  play(): Promise<void> {
    this.paused = false;
    // Fire 'ended' on next tick so the queue advances deterministically.
    setTimeout(() => {
      this.paused = true;
      this.dispatch("ended");
    }, 0);
    return Promise.resolve();
  }
  pause(): void {
    this.paused = true;
  }
  load(): void {
    /* no-op */
  }
  addEventListener(name: string, cb: () => void): void {
    (this.listeners[name] ||= []).push(cb);
  }
  removeEventListener(name: string, cb: () => void): void {
    this.listeners[name] = (this.listeners[name] || []).filter((x) => x !== cb);
  }
  dispatch(name: string): void {
    (this.listeners[name] || []).slice().forEach((cb) => cb());
  }
}

beforeEach(() => {
  vi.stubGlobal("Audio", FakeAudio as unknown as typeof Audio);
  vi.stubGlobal("WebSocket", class FakeWS {
    static OPEN = 1;
    readyState = 0;
    onopen: (() => void) | null = null;
    onmessage: ((ev: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    onerror: (() => void) | null = null;
    constructor(public url: string) {}
    send(_: string): void {}
    close(): void {
      this.readyState = 3;
      this.onclose?.();
    }
  });
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  localStorage.clear();
});
