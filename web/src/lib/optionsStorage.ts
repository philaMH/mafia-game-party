import { defaultOptions, type Options } from "../types/wire";

// Iteration 7 (FR-4) — host options localStorage cache.
// Key carries a version suffix so a future schema change can bump v1 → v2
// without colliding with stale entries.
const KEY = "mafia.options.v1";

function safeLocalStorage(): Storage | null {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

const NUMBER_FIELDS: Array<keyof Options> = [
  "mafiaCount",
  "maxPlayers",
  "introSecondsPerPlayer",
  "discussionSeconds",
  "nightMafiaSeconds",
  "nightPoliceSeconds",
  "nightDoctorSeconds",
];

const BOOLEAN_FIELDS: Array<keyof Options> = [
  "doctorSelfHealAllowed",
  "announcementVoiceOn",
];

function isPlainOptions(v: unknown): v is Options {
  if (!v || typeof v !== "object") return false;
  const o = v as Record<string, unknown>;
  for (const key of NUMBER_FIELDS) {
    const n = o[key];
    if (typeof n !== "number" || !Number.isFinite(n)) return false;
  }
  for (const key of BOOLEAN_FIELDS) {
    if (typeof o[key] !== "boolean") return false;
  }
  return true;
}

export function loadSavedOptions(): Options | null {
  const ls = safeLocalStorage();
  if (!ls) return null;
  const raw = ls.getItem(KEY);
  if (raw === null) return null;
  try {
    const parsed = JSON.parse(raw);
    if (!isPlainOptions(parsed)) {
      ls.removeItem(KEY);
      return null;
    }
    // Normalize via defaultOptions to fill any future-added field that
    // a v1 record may not carry yet.
    const fallback = defaultOptions(parsed.maxPlayers);
    return { ...fallback, ...parsed };
  } catch {
    ls.removeItem(KEY);
    return null;
  }
}

export function saveOptions(opts: Options): void {
  const ls = safeLocalStorage();
  if (!ls) return;
  try {
    ls.setItem(KEY, JSON.stringify(opts));
  } catch {
    // localStorage may throw on quota / private mode — fail silently.
  }
}

export function clearSavedOptions(): void {
  const ls = safeLocalStorage();
  if (!ls) return;
  try {
    ls.removeItem(KEY);
  } catch {
    // ignore
  }
}
