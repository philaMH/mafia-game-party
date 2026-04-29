import { useCallback, useEffect, useRef } from "react";
import type { Dispatch } from "react";

import type { IncomingMsg, OutgoingMsg } from "../types/wire";
import type { TokenIO } from "./useToken";
import type { GameAction } from "../context/reducer";

export interface UseWebSocketParams {
  url: string;
  dispatch: Dispatch<GameAction>;
  tokenIO: TokenIO;
}

export interface UseWebSocketResult {
  send(msg: OutgoingMsg): void;
}

const BACKOFF_MS = [1000, 2000, 4000, 8000, 16000];

// useWebSocket owns the connection lifecycle: it dials the URL on mount,
// dispatches all parsed wire messages into the game reducer, and stays
// alive across reconnects with exponential backoff capped at 16s
// (Q-NFR-U5-12=A). When the socket reopens and a saved token exists, a
// `resume` message is sent automatically (BR-U5-WS-3).
export function useWebSocket({ url, dispatch, tokenIO }: UseWebSocketParams): UseWebSocketResult {
  const wsRef = useRef<WebSocket | null>(null);
  const closedRef = useRef(false);
  const attemptRef = useRef(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const send = useCallback((msg: OutgoingMsg) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return; // silent drop — caller will retry after reconnect+resume
    }
    ws.send(JSON.stringify(msg));
  }, []);

  useEffect(() => {
    closedRef.current = false;

    const connect = (): void => {
      if (closedRef.current) return;
      dispatch({ type: "ws_connecting" });

      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        attemptRef.current = 0;
        dispatch({ type: "ws_open" });
        const token = tokenIO.get();
        if (token) {
          ws.send(JSON.stringify({ type: "resume", token }));
        }
      };

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as IncomingMsg;
          dispatch({ type: "ws_message", msg });
        } catch {
          // malformed wire — ignored per BR-U5-COMMON-3
        }
      };

      ws.onclose = () => {
        wsRef.current = null;
        if (closedRef.current) return;
        const idx = Math.min(attemptRef.current, BACKOFF_MS.length - 1);
        const delay = BACKOFF_MS[idx] ?? 16000;
        attemptRef.current++;
        dispatch({ type: "ws_reconnecting" });
        timerRef.current = setTimeout(connect, delay);
      };

      ws.onerror = () => {
        // close handler will run; nothing to do here
      };
    };

    connect();

    return () => {
      closedRef.current = true;
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      if (wsRef.current) {
        try {
          wsRef.current.close();
        } catch {
          // ignore
        }
        wsRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [url]);

  return { send };
}
