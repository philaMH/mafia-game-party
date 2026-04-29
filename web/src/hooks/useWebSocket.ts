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
    // `abandoned` is connection-local: a late onclose that fires after this
    // effect's cleanup must not re-arm the reconnect timer or null out a ref
    // that already points to the next mount's socket.
    let abandoned = false;

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
        if (abandoned) return;
        // Only null the ref when *this* socket still owns it — a delayed
        // close from a prior conn must not erase the current one.
        if (wsRef.current === ws) {
          wsRef.current = null;
        }
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

    // iOS Safari serializes WebSocket connections per origin: a new socket
    // can hang waiting for the previous one's TCP FIN. React's useEffect
    // cleanup is not guaranteed to run synchronously before page unload, so
    // we close on `pagehide` (which IS synchronous) to send FIN ahead of
    // the next page's open.
    const onPageHide = () => {
      const ws = wsRef.current;
      if (ws) {
        try {
          ws.close(1000, "pagehide");
        } catch {
          // already closing/closed
        }
      }
    };
    // BFCache restores a page with its JS state intact, but the WebSocket
    // reference is a zombie — the OS already closed the underlying socket.
    // Force a full reload so the saved player token (if any) drives a clean
    // resume on a fresh connection.
    const onPageShow = (e: PageTransitionEvent) => {
      if (e.persisted) {
        window.location.reload();
      }
    };
    window.addEventListener("pagehide", onPageHide);
    window.addEventListener("pageshow", onPageShow);

    return () => {
      abandoned = true;
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
      window.removeEventListener("pagehide", onPageHide);
      window.removeEventListener("pageshow", onPageShow);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [url]);

  return { send };
}
