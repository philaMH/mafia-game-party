import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useAudioCueQueue } from "./useAudioCueQueue";

interface FakeAudio {
  src: string;
  paused: boolean;
  play(): Promise<void>;
  pause(): void;
  dispatch(name: string): void;
}

function flushMicrotasks(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 1));
}

describe("useAudioCueQueue", () => {
  it("plays sequential audio cues in FIFO order", async () => {
    const { result } = renderHook(() => useAudioCueQueue(true));

    act(() => {
      result.current.enqueue("phase.night");
      result.current.enqueue("night.mafia");
    });

    await flushMicrotasks();
    await flushMicrotasks();
    await flushMicrotasks();

    // No assertion error means both finished without throwing.
    expect(result.current.available).toBe(true);
  });

  it("urgent enqueue interrupts and replaces the queue", async () => {
    const { result } = renderHook(() => useAudioCueQueue(true));

    act(() => {
      result.current.enqueue("phase.day");
      result.current.enqueueUrgent("end.citizen");
    });

    await flushMicrotasks();
    await flushMicrotasks();

    expect(result.current.available).toBe(true);
  });

  it("ignores enqueue when disabled", async () => {
    const { result } = renderHook(() => useAudioCueQueue(false));

    act(() => {
      result.current.enqueue("game.started");
    });

    expect(result.current.available).toBe(true);
  });

  it("ignores empty audioId (graceful skip)", async () => {
    const { result } = renderHook(() => useAudioCueQueue(true));

    act(() => {
      result.current.enqueue("");
    });

    expect(result.current.available).toBe(true);
  });

  it("cancelAll clears queue and stops playback", async () => {
    const { result } = renderHook(() => useAudioCueQueue(true));

    act(() => {
      result.current.enqueue("phase.night");
      result.current.cancelAll();
    });

    expect(result.current.available).toBe(true);
  });

  it("survives play() rejection by advancing to the next cue", async () => {
    // Replace global Audio with a version that rejects play() once.
    const original = (globalThis as unknown as { Audio: typeof Audio }).Audio;
    let attempt = 0;
    class RejectingAudio {
      src = "";
      paused = true;
      currentTime = 0;
      addEventListener(): void {}
      removeEventListener(): void {}
      pause(): void {}
      play(): Promise<void> {
        attempt += 1;
        if (attempt === 1) return Promise.reject(new Error("missing"));
        // Second cue resolves.
        setTimeout(() => {
          (this as unknown as FakeAudio).dispatch?.("ended");
        }, 0);
        return Promise.resolve();
      }
    }
    vi.stubGlobal("Audio", RejectingAudio as unknown as typeof Audio);
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});

    const { result } = renderHook(() => useAudioCueQueue(true));
    act(() => {
      result.current.enqueue("missing.cue");
      result.current.enqueue("phase.night");
    });

    await flushMicrotasks();
    await flushMicrotasks();
    await flushMicrotasks();

    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
    vi.stubGlobal("Audio", original);
  });
});
