import { renderHook, act } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useToken } from "./useToken";

describe("useToken", () => {
  it("get returns null when nothing is stored", () => {
    const { result } = renderHook(() => useToken());
    expect(result.current.get()).toBeNull();
  });

  it("set persists token and get reads it", () => {
    const { result } = renderHook(() => useToken());
    act(() => result.current.set("tok-123"));
    expect(result.current.get()).toBe("tok-123");
  });

  it("clear removes the token", () => {
    const { result } = renderHook(() => useToken());
    act(() => {
      result.current.set("tok-123");
      result.current.clear();
    });
    expect(result.current.get()).toBeNull();
  });
});
