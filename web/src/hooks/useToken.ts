import { useMemo } from "react";

const TOKEN_KEY = "mafia.token";

export interface TokenIO {
  get(): string | null;
  set(token: string): void;
  clear(): void;
}

// safeLocalStorage isolates the localStorage access so SSR / private-mode
// failures throw exactly once at the boundary instead of poisoning every
// caller (NFR-U5-S1: token IO is the only place that touches storage).
function safeLocalStorage(): Storage | null {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

export function useToken(): TokenIO {
  return useMemo<TokenIO>(
    () => ({
      get: () => safeLocalStorage()?.getItem(TOKEN_KEY) ?? null,
      set: (token) => {
        safeLocalStorage()?.setItem(TOKEN_KEY, token);
      },
      clear: () => {
        safeLocalStorage()?.removeItem(TOKEN_KEY);
      },
    }),
    [],
  );
}
