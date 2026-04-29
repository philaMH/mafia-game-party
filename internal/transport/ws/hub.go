package ws

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/session"
)

// Hub is U3's public facade onto WebSocket transport. It owns the
// gorilla Upgrader, the client registry, and a single Subscribe handler
// on the SessionManager.
type Hub interface {
	// Register binds an already-upgraded *websocket.Conn to a fresh
	// Client and starts the per-client read/write goroutines. Most
	// callers will use UpgradeHandler instead.
	Register(conn *websocket.Conn) (ClientID, error)

	// Unregister cancels and removes the named client (idempotent).
	Unregister(id ClientID)

	// Run blocks until ctx is cancelled, then performs Close.
	Run(ctx context.Context) error

	// Close cancels every client and unsubscribes from the
	// SessionManager. Idempotent.
	Close() error

	// UpgradeHandler returns an http.HandlerFunc that performs the
	// WebSocket upgrade and registers the resulting connection.
	UpgradeHandler() http.HandlerFunc
}

// hub is the default Hub implementation.
type hub struct {
	mgr      session.SessionManager
	upgrader websocket.Upgrader
	log      *slog.Logger

	registry    *clientRegistry
	unsubscribe func()

	rootCtx    context.Context
	rootCancel context.CancelFunc

	closeOnce sync.Once
	closed    atomic.Bool
}

// New constructs a Hub. The Subscribe wiring is performed eagerly so
// the Hub never misses an event published between New and Run.
//
// If log is nil, slog.Default() is used.
func New(upgrader websocket.Upgrader, mgr session.SessionManager, log *slog.Logger) Hub {
	if log == nil {
		log = slog.Default()
	}
	rootCtx, rootCancel := context.WithCancel(context.Background())

	h := &hub{
		mgr:        mgr,
		upgrader:   upgrader,
		log:        log,
		registry:   newClientRegistry(),
		rootCtx:    rootCtx,
		rootCancel: rootCancel,
	}
	h.unsubscribe = mgr.Subscribe(h.onEvent)
	return h
}

// Register implements Hub.
func (h *hub) Register(conn *websocket.Conn) (ClientID, error) {
	if conn == nil {
		return "", errors.New("ws: nil conn")
	}
	if h.closed.Load() {
		_ = conn.Close()
		return "", errors.New("ws: hub closed")
	}

	c := newClient(h.rootCtx, conn)
	h.registry.add(c)

	// Welcome message — best-effort enqueue. If the channel is full at
	// startup something is very wrong, but we proceed regardless.
	h.enqueue(c, mustMarshal(welcomeMsg{
		Type:            TypeWelcome,
		ClientID:        c.ID,
		Kind:            c.Kind.String(),
		ProtocolVersion: protocolVersion,
	}))

	// Iteration 3 — late-joiner resync: push current room/game state so
	// clients that connect after OpenRoom or HostStartGame don't sit on
	// the LOBBY gate forever. Done before readLoop/writeLoop start so
	// the messages land before any client traffic interleaves.
	h.pushRoomState(c, h.mgr.RoomSnapshot())

	go h.readLoop(c)
	go h.writeLoop(c)

	h.log.Info("ws client registered", "client", c.ID, "kind", c.Kind.String())
	return c.ID, nil
}

// Unregister implements Hub.
func (h *hub) Unregister(id ClientID) {
	c := h.registry.remove(id)
	if c == nil {
		return
	}
	c.cancel()
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
	h.log.Info("ws client unregistered", "client", c.ID, "kind", c.Kind.String(), "playerId", c.PlayerID)
}

// Run implements Hub.
func (h *hub) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
	case <-h.rootCtx.Done():
	}
	return h.Close()
}

// Close implements Hub. Cancels every client and tears down the
// SessionManager subscription. Idempotent.
func (h *hub) Close() error {
	var firstErr error
	h.closeOnce.Do(func() {
		h.closed.Store(true)
		if h.unsubscribe != nil {
			h.unsubscribe()
			h.unsubscribe = nil
		}
		for _, c := range h.registry.all() {
			h.Unregister(c.ID)
		}
		h.rootCancel()
	})
	return firstErr
}

// UpgradeHandler implements Hub.
func (h *hub) UpgradeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			h.log.Warn("ws upgrade failed", "err", err)
			return
		}
		if _, err := h.Register(conn); err != nil {
			h.log.Warn("ws register failed", "err", err)
			_ = conn.Close()
			return
		}
	}
}
