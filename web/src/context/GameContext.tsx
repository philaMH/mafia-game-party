import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useState,
  type ReactNode,
} from "react";

import { useToken } from "../hooks/useToken";
import { useTTSQueue } from "../hooks/useTTSQueue";
import { useWebSocket } from "../hooks/useWebSocket";
import { loadSavedOptions, saveOptions } from "../lib/optionsStorage";
import { defaultOptions, type Options, type OutgoingMsg } from "../types/wire";

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
  // Iteration 7 — host main menu / settings route.
  hostOptions: Options;
  saveHostOptions(opts: Options): void;
}

export const GameContext = createContext<GameContextValue | null>(null);

const URGENT_KINDS = new Set([
  "PhaseChanged",
  "Eliminated",
  "DeathAnnounced",
  "GameEnded",
]);

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
  const tts = useTTSQueue(state.voiceOn);

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

  // Mark TTS unavailable once detected so PublicView can render the
  // subtitle-only fallback toast (FR-8.7).
  useEffect(() => {
    if (!tts.available && state.ttsAvailable) {
      dispatch({ type: "tts_unavailable" });
    }
  }, [tts.available, state.ttsAvailable]);

  // Speak each new announcement. Urgent transitions interrupt; everything
  // else queues behind whatever is currently playing.
  useEffect(() => {
    const ann = state.lastAnnounce;
    if (!ann) return;
    const kind = state.lastEventKind;
    if (kind && URGENT_KINDS.has(kind)) {
      tts.enqueueUrgent(ann.subtitle);
    } else {
      tts.enqueue(ann.subtitle);
    }
    // We deliberately depend only on the announcement reference so a
    // single message is spoken once; `tts` is stable across renders.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.lastAnnounce]);

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

  const [hostOptions, setHostOptions] = useState<Options>(
    () => loadSavedOptions() ?? defaultOptions(8),
  );
  const saveHostOptions = useCallback(
    (opts: Options) => {
      setHostOptions(opts);
      saveOptions(opts);
      send({ type: "host:save-options", options: opts });
    },
    [send],
  );

  const value: GameContextValue = useMemo(
    () => ({
      ...state,
      send,
      toggleVoice,
      ackError,
      logout,
      hostOptions,
      saveHostOptions,
    }),
    [state, send, toggleVoice, ackError, logout, hostOptions, saveHostOptions],
  );

  return <GameContext.Provider value={value}>{children}</GameContext.Provider>;
}

export function useGameContext(): GameContextValue {
  const ctx = useContext(GameContext);
  if (!ctx) throw new Error("useGameContext must be used inside GameProvider");
  return ctx;
}

export type { ConnectionStatus };
