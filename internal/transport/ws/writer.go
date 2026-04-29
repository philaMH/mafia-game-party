package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

// pingInterval is how often the server sends a ping frame on each open
// connection (BR-U3-HEARTBEAT-2). Combined with readDeadline below it
// keeps idle TCP connections alive and lets us detect silent peers.
const pingInterval = 25 * time.Second

// writeDeadline bounds a single conn write — both ping and message.
const writeDeadline = 10 * time.Second

// writeLoop drains Client.Out and pushes ping frames on a ticker. It is
// the only place gorilla.Conn.WriteMessage is called for a given client
// (single-writer requirement). Termination paths:
//   - client.ctx cancelled → return; defers close conn → readLoop exits.
//   - WriteMessage error → return; same teardown chain.
//
// We never close c.Out here — it is owned by the Hub and may receive
// late enqueues; instead, ctx cancellation is the canonical signal.
func (h *hub) writeLoop(c *Client) {
	defer func() {
		_ = c.Conn.Close()
	}()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.Out:
			if !ok {
				return
			}
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				h.log.Debug("ws write failed", "client", c.ID, "err", err)
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.log.Debug("ws ping failed", "client", c.ID, "err", err)
				return
			}
		}
	}
}

// enqueue submits a wire message to a client's send channel without
// blocking. If the client is already cancelled or its buffer is full,
// the message is dropped and the slow client is unregistered (in a
// separate goroutine to avoid recursive registry locking).
//
// Caller does NOT hold the registry lock.
func (h *hub) enqueue(c *Client, msg []byte) {
	select {
	case <-c.ctx.Done():
		return
	default:
	}
	select {
	case c.Out <- msg:
		// queued
	default:
		h.log.Warn("ws send buffer full; disconnecting", "client", c.ID)
		go h.Unregister(c.ID)
	}
}
