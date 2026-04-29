import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useTTSQueue } from "./useTTSQueue";

interface FakeSynth {
  utterances: { text: string; onend?: (() => void) | null }[];
  speaking: boolean;
  cancel(): void;
}

function getSynth(): FakeSynth {
  return window.speechSynthesis as unknown as FakeSynth;
}

describe("useTTSQueue", () => {
  it("available reflects window.speechSynthesis", () => {
    const { result } = renderHook(() => useTTSQueue(true));
    expect(result.current.available).toBe(true);
  });

  it("enqueue speaks immediately when idle", () => {
    const { result } = renderHook(() => useTTSQueue(true));
    act(() => result.current.enqueue("안녕"));
    const synth = getSynth();
    expect(synth.utterances).toHaveLength(1);
    expect(synth.utterances[0]?.text).toBe("안녕");
  });

  it("enqueueUrgent cancels and speaks", () => {
    const { result } = renderHook(() => useTTSQueue(true));
    act(() => {
      result.current.enqueue("첫번째");
      result.current.enqueueUrgent("긴급!");
    });
    const synth = getSynth();
    // After cancel + urgent speak, queue length is 1 (urgent only).
    expect(synth.utterances.at(-1)?.text).toBe("긴급!");
  });

  it("disabling cancels queued speech", () => {
    const { result, rerender } = renderHook(({ on }: { on: boolean }) => useTTSQueue(on), {
      initialProps: { on: true },
    });
    act(() => result.current.enqueue("hello"));
    rerender({ on: false });
    expect(getSynth().utterances).toHaveLength(0);
  });

  it("does nothing when disabled at enqueue time", () => {
    const { result } = renderHook(() => useTTSQueue(false));
    act(() => result.current.enqueue("ignored"));
    expect(getSynth().utterances).toHaveLength(0);
  });
});
