import { useCallback, useEffect, useRef, useState } from "react";

export interface TTSOpts {
  lang?: string;
  pitch?: number;
  rate?: number;
  volume?: number;
}

export interface TTSQueue {
  enqueue(text: string, opts?: TTSOpts): void;
  enqueueUrgent(text: string, opts?: TTSOpts): void;
  cancelAll(): void;
  available: boolean;
}

interface QueueItem {
  text: string;
  opts: Required<Pick<TTSOpts, "lang" | "pitch" | "rate" | "volume">>;
}

const DEFAULT_OPTS = {
  lang: "ko-KR",
  pitch: 0.9,
  rate: 0.95,
  volume: 1,
};

// useTTSQueue layers a serial queue on top of window.speechSynthesis so
// rapid `announce` messages are spoken in order rather than overlapping.
// `enqueueUrgent` clears the queue and starts immediately — used for
// phase transitions and player deaths (BR-U5-TTS-5).
export function useTTSQueue(enabled: boolean): TTSQueue {
  const synth = typeof window !== "undefined" ? window.speechSynthesis : undefined;
  const available = !!synth;

  const queueRef = useRef<QueueItem[]>([]);
  const speakingRef = useRef(false);
  const enabledRef = useRef(enabled);
  const [voices, setVoices] = useState<SpeechSynthesisVoice[]>([]);

  // Keep the ref in sync so callbacks always see the latest enabled flag.
  useEffect(() => {
    enabledRef.current = enabled;
    if (!enabled && synth) {
      synth.cancel();
      queueRef.current = [];
      speakingRef.current = false;
    }
  }, [enabled, synth]);

  // Voice loading: Safari returns a populated list synchronously; Chrome
  // populates it asynchronously and emits voiceschanged once ready
  // (P-U5-4).
  useEffect(() => {
    if (!synth) return;
    const load = (): void => setVoices(synth.getVoices());
    load();
    synth.addEventListener?.("voiceschanged", load);
    return () => {
      synth.removeEventListener?.("voiceschanged", load);
    };
  }, [synth]);

  const pickVoice = useCallback(
    (lang: string): SpeechSynthesisVoice | null => {
      if (voices.length === 0) return null;
      return voices.find((v) => v.lang.startsWith(lang.split("-")[0] ?? "ko")) ?? null;
    },
    [voices],
  );

  const speakNow = useCallback(
    (item: QueueItem): void => {
      if (!synth || !enabledRef.current) {
        speakingRef.current = false;
        return;
      }
      const utt = new SpeechSynthesisUtterance(item.text);
      utt.lang = item.opts.lang;
      utt.pitch = item.opts.pitch;
      utt.rate = item.opts.rate;
      utt.volume = item.opts.volume;
      const voice = pickVoice(item.opts.lang);
      if (voice) utt.voice = voice;
      utt.onend = () => {
        speakingRef.current = false;
        const next = queueRef.current.shift();
        if (next) speakNow(next);
      };
      utt.onerror = () => {
        speakingRef.current = false;
        const next = queueRef.current.shift();
        if (next) speakNow(next);
      };
      speakingRef.current = true;
      synth.speak(utt);
    },
    [synth, pickVoice],
  );

  const buildItem = (text: string, opts?: TTSOpts): QueueItem => ({
    text,
    opts: {
      lang: opts?.lang ?? DEFAULT_OPTS.lang,
      pitch: opts?.pitch ?? DEFAULT_OPTS.pitch,
      rate: opts?.rate ?? DEFAULT_OPTS.rate,
      volume: opts?.volume ?? DEFAULT_OPTS.volume,
    },
  });

  const enqueue = useCallback(
    (text: string, opts?: TTSOpts) => {
      if (!available || !enabledRef.current) return;
      const item = buildItem(text, opts);
      if (speakingRef.current) {
        queueRef.current.push(item);
      } else {
        speakNow(item);
      }
    },
    [available, speakNow],
  );

  const enqueueUrgent = useCallback(
    (text: string, opts?: TTSOpts) => {
      if (!available || !enabledRef.current) return;
      synth?.cancel();
      queueRef.current = [];
      speakingRef.current = false;
      speakNow(buildItem(text, opts));
    },
    [available, synth, speakNow],
  );

  const cancelAll = useCallback(() => {
    if (synth) synth.cancel();
    queueRef.current = [];
    speakingRef.current = false;
  }, [synth]);

  // Best-effort cleanup so refresh / route change doesn't leave a phantom
  // utterance reading half a sentence after unmount.
  useEffect(() => {
    return () => {
      if (synth) synth.cancel();
    };
  }, [synth]);

  return { enqueue, enqueueUrgent, cancelAll, available };
}
