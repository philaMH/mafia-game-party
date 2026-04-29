import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { defaultOptions } from "../types/wire";

import {
  clearSavedOptions,
  loadSavedOptions,
  saveOptions,
} from "./optionsStorage";

const KEY = "mafia.options.v1";

describe("optionsStorage", () => {
  afterEach(() => {
    localStorage.clear();
  });

  it("returns null when nothing is stored", () => {
    expect(loadSavedOptions()).toBeNull();
  });

  it("round-trips a valid Options value", () => {
    const opts = { ...defaultOptions(8), discussionSeconds: 240 };
    saveOptions(opts);
    expect(loadSavedOptions()).toEqual(opts);
  });

  it("clears on malformed JSON", () => {
    localStorage.setItem(KEY, "not-json");
    expect(loadSavedOptions()).toBeNull();
    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("clears on schema mismatch", () => {
    localStorage.setItem(KEY, JSON.stringify({ mafiaCount: "2" })); // wrong type
    expect(loadSavedOptions()).toBeNull();
    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("fills missing future-added fields with defaults", () => {
    // Simulate a v1 record that already has every required field.
    const v1 = defaultOptions(7);
    localStorage.setItem(KEY, JSON.stringify(v1));
    expect(loadSavedOptions()).toEqual(v1);
  });

  it("clearSavedOptions removes the key", () => {
    saveOptions(defaultOptions(8));
    clearSavedOptions();
    expect(localStorage.getItem(KEY)).toBeNull();
  });

  describe("safeLocalStorage fallback", () => {
    let original: PropertyDescriptor | undefined;
    beforeEach(() => {
      original = Object.getOwnPropertyDescriptor(window, "localStorage");
      Object.defineProperty(window, "localStorage", {
        configurable: true,
        get() {
          throw new Error("disabled");
        },
      });
    });
    afterEach(() => {
      if (original) {
        Object.defineProperty(window, "localStorage", original);
      }
    });

    it("loadSavedOptions returns null when storage throws", () => {
      expect(loadSavedOptions()).toBeNull();
    });

    it("saveOptions/clearSavedOptions are no-ops when storage throws", () => {
      const spy = vi.fn();
      // Ensure no exception bubbles up.
      saveOptions(defaultOptions(8));
      clearSavedOptions();
      expect(spy).not.toHaveBeenCalled();
    });
  });
});
