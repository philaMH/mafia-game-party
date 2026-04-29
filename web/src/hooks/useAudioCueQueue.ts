import { useCallback, useEffect, useRef } from "react";

export interface AudioCueQueue {
  enqueue(audioId: string): void;
  enqueueUrgent(audioId: string): void;
  cancelAll(): void;
  available: boolean;
}

// useAudioCueQueue plays pre-recorded MP3 cues serially on the host's
// PublicView (Iter7 FR-8.1/8.2). Each queued audioId maps to
// /audio/<audioId>.mp3 and is played via a single HTMLAudioElement.
//
// Behaviour:
//  - enqueue: append to FIFO; play immediately if idle.
//  - enqueueUrgent: cancel current playback and queue, play immediately.
//    Used by GameContext for PhaseChanged / Eliminated / DeathAnnounced /
//    GameEnded so urgent transitions interrupt pending narration.
//  - missing files / decode errors: log a console warning and advance to
//    the next item (FR-8.8 graceful skip — game continues uninterrupted).
//  - disabled (enabled=false): cancel and refuse new enqueues.
export function useAudioCueQueue(enabled: boolean): AudioCueQueue {
  const available =
    typeof window !== "undefined" && typeof Audio !== "undefined";

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const queueRef = useRef<string[]>([]);
  const playingRef = useRef(false);
  const enabledRef = useRef(enabled);

  // Lazily build a single shared Audio element. Reusing one element keeps
  // browser autoplay accounting consistent with the user's original
  // priming gesture (e.g. clicking 방 개설).
  const getAudio = useCallback((): HTMLAudioElement | null => {
    if (!available) return null;
    if (audioRef.current) return audioRef.current;
    const el = new Audio();
    el.preload = "auto";
    el.addEventListener("ended", onEnded);
    el.addEventListener("error", onError);
    audioRef.current = el;
    return el;
    // onEnded / onError are stable via refs below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [available]);

  const playNext = useCallback((): void => {
    const el = audioRef.current;
    if (!el || !enabledRef.current) {
      playingRef.current = false;
      return;
    }
    const id = queueRef.current.shift();
    if (!id) {
      playingRef.current = false;
      return;
    }
    el.src = `/audio/${id}.mp3`;
    playingRef.current = true;
    el.play().catch((err) => {
      // Autoplay rejected, file missing, decode error — log and skip.
      // eslint-disable-next-line no-console
      console.warn(`[audio] failed to play /audio/${id}.mp3:`, err);
      playingRef.current = false;
      playNext();
    });
  }, []);

  const onEnded = useCallback((): void => {
    playingRef.current = false;
    playNext();
  }, [playNext]);

  const onError = useCallback((): void => {
    // eslint-disable-next-line no-console
    console.warn(`[audio] error event for ${audioRef.current?.src}`);
    playingRef.current = false;
    playNext();
  }, [playNext]);

  // Sync enabled flag and tear down current playback when toggled off.
  useEffect(() => {
    enabledRef.current = enabled;
    if (!enabled) {
      const el = audioRef.current;
      if (el) {
        el.pause();
        el.currentTime = 0;
      }
      queueRef.current = [];
      playingRef.current = false;
    }
  }, [enabled]);

  // Detach listeners on unmount so a refresh / route change doesn't keep
  // a phantom element alive.
  useEffect(() => {
    return () => {
      const el = audioRef.current;
      if (el) {
        el.pause();
        el.removeEventListener("ended", onEnded);
        el.removeEventListener("error", onError);
      }
      audioRef.current = null;
      queueRef.current = [];
      playingRef.current = false;
    };
  }, [onEnded, onError]);

  const enqueue = useCallback(
    (audioId: string) => {
      if (!available || !enabledRef.current || !audioId) return;
      // Touch the lazy audio element so the listeners are wired before
      // the first playback attempt.
      getAudio();
      queueRef.current.push(audioId);
      if (!playingRef.current) {
        playNext();
      }
    },
    [available, getAudio, playNext],
  );

  const enqueueUrgent = useCallback(
    (audioId: string) => {
      if (!available || !enabledRef.current || !audioId) return;
      const el = getAudio();
      if (el) {
        el.pause();
        el.currentTime = 0;
      }
      queueRef.current = [audioId];
      playingRef.current = false;
      playNext();
    },
    [available, getAudio, playNext],
  );

  const cancelAll = useCallback(() => {
    const el = audioRef.current;
    if (el) {
      el.pause();
      el.currentTime = 0;
    }
    queueRef.current = [];
    playingRef.current = false;
  }, []);

  return { enqueue, enqueueUrgent, cancelAll, available };
}
