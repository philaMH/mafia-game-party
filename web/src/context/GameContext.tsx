import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  type ReactNode,
} from "react";

import { useAudioCueQueue } from "../hooks/useAudioCueQueue";
import { useToken } from "../hooks/useToken";
import { useWebSocket } from "../hooks/useWebSocket";
import type { OutgoingMsg } from "../types/wire";

import {
  gameReducer,
  initialState,
  type ConnectionStatus,
  type GameState,
} from "./reducer";

export interface GameContextValue extends GameState {
  send(msg: OutgoingMsg): void;
  toggleVoice(on: boolean): void;
  ackError(addedAt: number): void;
  logout(): void;
}

const GameContext = createContext<GameContextValue | null>(null);

// Iter7-followup: URGENT interruption was removed because every typical
// transition (GameStarted → PhaseChanged → IntroSpeakerChanged, or
// PhaseChanged(NIGHT) → NightStepChanged) emits two announces back-to-
// back; treating PhaseChanged/Eliminated/DeathAnnounced/GameEnded as
// urgent caused the host to interrupt the very narration the previous
// frame had just started — game.started clipped, phase.day clipped,
// etc. With the FIFO log drain (see useEffect below), all announces
// queue and play in order, which matches the calm, sequential narration
// tone documented in voice-script.md §4.1. The enqueueUrgent API is
// preserved on the queue hook for callers that need it in the future.
function defaultUrl(): string {
  if (typeof window === "undefined") return "ws://localhost:8080/ws";
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws`;
}

export interface GameProviderProps {
  children: ReactNode;
  url?: string;
}

export function GameProvider({ children, url }: GameProviderProps) {
  const [state, dispatch] = useReducer(gameReducer, initialState);
  const tokenIO = useToken();
  const { send } = useWebSocket({
    url: url ?? defaultUrl(),
    dispatch,
    tokenIO,
  });
  // Iter7 — audio cues only fire on the host PublicView. Gating on
  // isHost here means a second PublicView tab opened by an observer
  // will be silent regardless of its voice toggle (FR-8.2/8.10).
  const audio = useAudioCueQueue(state.voiceOn && state.isHost);

  // Persist token whenever the server issues one; clear it on resume
  // failure so the next page load starts fresh (BR-U5-TOKEN-3).
  useEffect(() => {
    if (state.token) {
      tokenIO.set(state.token);
    }
  }, [state.token, tokenIO]);

  useEffect(() => {
    const last = state.errors[state.errors.length - 1];
    if (last && last.code === "UNKNOWN_PLAYER_ERROR") {
      tokenIO.clear();
      dispatch({ type: "logout" });
    }
  }, [state.errors, tokenIO]);

  // room:closed bumps a monotonic counter in the reducer; when it
  // changes we drop the saved player token from localStorage so the
  // next page load doesn't try to resume into an invalid session.
  useEffect(() => {
    if (state.roomClosedSeq > 0) {
      tokenIO.clear();
    }
  }, [state.roomClosedSeq, tokenIO]);

  // Mark audio unavailable once detected (HTMLAudioElement missing —
  // typically only true under SSR or a stripped jsdom).
  useEffect(() => {
    if (!audio.available && state.audioAvailable) {
      dispatch({ type: "audio_unavailable" });
    }
  }, [audio.available, state.audioAvailable]);

  // Drain every newly-appended audio cue from the reducer's FIFO log.
  // Tracking by monotonic seq (instead of `lastAnnounce` reference) makes
  // dispatch resilient to React batching multiple WS frames into a single
  // render — previously the effect saw only the last announcement and
  // silently dropped phase.night, game.started, and intro.speaker
  // whenever the server emitted them adjacent to a non-URGENT successor.
  const lastAudioSeqRef = useRef(0);
  useEffect(() => {
    const cues = state.audioCues;
    if (cues.length === 0) return;
    const watermark = lastAudioSeqRef.current;
    let advanced = watermark;
    for (const cue of cues) {
      if (cue.seq <= watermark) continue;
      audio.enqueue(cue.audioId);
      advanced = cue.seq;
    }
    lastAudioSeqRef.current = advanced;
    // `audio` is stable across renders; drain is keyed strictly on the
    // log array's reference change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.audioCues]);

  const toggleVoice = useCallback(
    (on: boolean) => {
      dispatch({ type: "set_voice", on });
      send({ type: "host:toggle-voice", on });
    },
    [send],
  );

  const ackError = useCallback((addedAt: number) => {
    dispatch({ type: "ack_error", addedAt });
  }, []);

  const logout = useCallback(() => {
    tokenIO.clear();
    dispatch({ type: "logout" });
  }, [tokenIO]);

  const value: GameContextValue = useMemo(
    () => ({ ...state, send, toggleVoice, ackError, logout }),
    [state, send, toggleVoice, ackError, logout],
  );

  return <GameContext.Provider value={value}>{children}</GameContext.Provider>;
}

export function useGameContext(): GameContextValue {
  const ctx = useContext(GameContext);
  if (!ctx) throw new Error("useGameContext must be used inside GameProvider");
  return ctx;
}

export type { ConnectionStatus };
