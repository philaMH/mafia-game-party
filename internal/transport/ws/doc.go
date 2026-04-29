// Package ws is U3's realtime transport layer. It wraps gorilla/websocket
// to manage multi-client WebSocket connections, routes domain events from
// the SessionManager (U2) to clients per the engine's Visibility policy
// (U1), and forwards client inputs back to the SessionManager.
//
// Concurrency model:
//   - The Hub holds a single ClientRegistry RWMutex for client bookkeeping.
//   - Each client has its own read goroutine and write goroutine. All
//     conn.WriteMessage calls happen in the write goroutine — gorilla
//     requires single-writer semantics.
//   - SessionManager event handlers run inside U2's GM lock. The Hub's
//     onEvent must therefore avoid blocking I/O — it only enqueues into
//     per-client send channels and returns immediately.
//   - Per-client cancellation is signaled via context.Context so that
//     close(c.Out) is never required (and never racy).
package ws
