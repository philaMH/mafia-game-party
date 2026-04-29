import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useBgm } from "./useBgm";

interface FakeAudio {
  src: string;
  loop: boolean;
  volume: number;
  preload: string;
  paused: boolean;
  currentTime: number;
  play: ReturnType<typeof vi.fn>;
  pause: ReturnType<typeof vi.fn>;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
}

let lastAudio: FakeAudio | null = null;
let playImpl: () => Promise<void> = () => Promise.resolve();
const originalAudio = globalThis.Audio;

beforeEach(() => {
  lastAudio = null;
  playImpl = () => Promise.resolve();
  globalThis.Audio = vi.fn().mockImplementation((src?: string) => {
    const fake: FakeAudio = {
      src: src ?? "",
      loop: false,
      volume: 1,
      preload: "",
      paused: true,
      currentTime: 0,
      play: vi.fn(() => {
        fake.paused = false;
        return playImpl();
      }),
      pause: vi.fn(() => {
        fake.paused = true;
      }),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    };
    lastAudio = fake;
    return fake;
  });
});

afterEach(() => {
  globalThis.Audio = originalAudio;
  vi.restoreAllMocks();
});

describe("useBgm", () => {
  it("creates a looping element at volume 0.15 and plays when enabled", async () => {
    renderHook(() => useBgm(true));
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastAudio).not.toBeNull();
    expect(lastAudio!.src).toContain("/audio/bgm.mp3");
    expect(lastAudio!.loop).toBe(true);
    expect(lastAudio!.volume).toBe(0.15);
    expect(lastAudio!.play).toHaveBeenCalledTimes(1);
  });

  it("pauses without resetting currentTime when toggled off", async () => {
    const { rerender } = renderHook(({ on }) => useBgm(on), {
      initialProps: { on: true },
    });
    await act(async () => {
      await Promise.resolve();
    });
    lastAudio!.currentTime = 12.34;
    rerender({ on: false });
    await act(async () => {
      await Promise.resolve();
    });
    expect(lastAudio!.pause).toHaveBeenCalled();
    expect(lastAudio!.currentTime).toBe(12.34);
  });

  it("logs a warning when play() rejects (graceful)", async () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    playImpl = () => Promise.reject(new Error("autoplay denied"));
    renderHook(() => useBgm(true));
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
    });
    expect(warn).toHaveBeenCalled();
    const arg = warn.mock.calls[0]?.[0];
    expect(String(arg)).toContain("[bgm] failed to play");
  });

  it("pauses and removes listener on unmount", async () => {
    const { unmount } = renderHook(() => useBgm(true));
    await act(async () => {
      await Promise.resolve();
    });
    const el = lastAudio!;
    unmount();
    expect(el.pause).toHaveBeenCalled();
    expect(el.removeEventListener).toHaveBeenCalledWith(
      "error",
      expect.any(Function),
    );
  });
});
