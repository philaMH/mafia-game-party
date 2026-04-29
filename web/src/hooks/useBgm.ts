import { useEffect, useRef } from "react";

const BGM_SRC = "/audio/bgm.mp3";
const BGM_VOLUME = 0.15;

export interface BgmHandle {
  available: boolean;
}

// useBgm renders a single looping HTMLAudioElement on the host's
// PublicView (Iter10 FR-1~FR-7). Independent from useAudioCueQueue —
// effect cues and BGM coexist without ducking (Q3=B). Graceful: any
// play() rejection or `error` event logs a warning and leaves the game
// untouched. enabled=false pauses without resetting currentTime so a
// subsequent re-enable resumes from the same position.
export function useBgm(enabled: boolean): BgmHandle {
  const available =
    typeof window !== "undefined" && typeof Audio !== "undefined";
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    if (!available) return;
    const el = new Audio(BGM_SRC);
    el.loop = true;
    el.volume = BGM_VOLUME;
    el.preload = "auto";
    const onError = (): void => {
      // eslint-disable-next-line no-console
      console.warn(`[bgm] error event for ${BGM_SRC}`);
    };
    el.addEventListener("error", onError);
    audioRef.current = el;
    return () => {
      el.pause();
      el.removeEventListener("error", onError);
      audioRef.current = null;
    };
  }, [available]);

  useEffect(() => {
    const el = audioRef.current;
    if (!el) return;
    if (enabled) {
      el.play().catch((err) => {
        // eslint-disable-next-line no-console
        console.warn(`[bgm] failed to play ${BGM_SRC}:`, err);
      });
    } else {
      el.pause();
    }
  }, [enabled]);

  return { available };
}
