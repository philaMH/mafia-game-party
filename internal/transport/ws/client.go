package ws

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

// ClientKind classifies a WebSocket connection. PUBLIC clients receive
// public-screen messages and announcements only; PLAYER clients also
// receive role-private events targeted at their PlayerID.
type ClientKind int

// ClientKind values.
const (
	ClientPublic ClientKind = iota
	ClientPlayer
)

// String returns the wire-format string for a ClientKind.
func (k ClientKind) String() string {
	switch k {
	case ClientPublic:
		return "PUBLIC"
	case ClientPlayer:
		return "PLAYER"
	default:
		return "UNKNOWN"
	}
}

// Client is the Hub's view of a single connected WebSocket peer.
//
// `Out` is buffered (16) — enqueue uses a non-blocking select with a
// default branch so a slow client only loses *its own* connection
// (BR-U3-QUEUE-2). `ctx`/`cancel` provide the per-client kill switch
// used by writeLoop instead of close(c.Out) (P-U3-4).
type Client struct {
	ID       ClientID
	Kind     ClientKind
	PlayerID game.PlayerID // empty until bindPlayer

	// HostToken is non-empty when this client successfully claimed the GM
	// seat via host:claim (Iteration 2). The hub releases it on read-loop
	// exit so the next /public connection can become the host.
	HostToken session.HostToken

	Conn *websocket.Conn
	Out  chan []byte

	ctx    context.Context
	cancel context.CancelFunc

	JoinedAt time.Time
}

// outBufferSize is the per-client send-channel buffer (NFR-U3-G1).
const outBufferSize = 16

// newClient constructs a Client with all derived fields wired. It does
// not register the client with the Hub or start any goroutines — the Hub
// is responsible for lifecycle management.
func newClient(parent context.Context, conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(parent)
	return &Client{
		ID:       newClientID(),
		Kind:     ClientPublic,
		Conn:     conn,
		Out:      make(chan []byte, outBufferSize),
		ctx:      ctx,
		cancel:   cancel,
		JoinedAt: time.Now(),
	}
}

// clientRegistry maintains three coordinated indices over the active
// client set. A single RWMutex synchronizes all updates; readers (route
// dispatching) take the read lock and copy slices so the caller can
// iterate without holding the registry lock (BR-U3-COMMON-4).
type clientRegistry struct {
	mu         sync.RWMutex
	byID       map[ClientID]*Client
	byPlayerID map[game.PlayerID]*Client
	publics    map[ClientID]*Client
}

// newClientRegistry constructs an empty registry.
func newClientRegistry() *clientRegistry {
	return &clientRegistry{
		byID:       make(map[ClientID]*Client),
		byPlayerID: make(map[game.PlayerID]*Client),
		publics:    make(map[ClientID]*Client),
	}
}

// add inserts a freshly minted Client. Initial Kind is PUBLIC; the
// publics index is therefore populated. PlayerID indexing happens later
// in bindPlayer.
func (r *clientRegistry) add(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[c.ID] = c
	if c.Kind == ClientPublic {
		r.publics[c.ID] = c
	} else if c.PlayerID != "" {
		r.byPlayerID[c.PlayerID] = c
	}
}

// remove drops the client from all indices and returns it (or nil if it
// was never registered or already removed). This is safe to call
// multiple times — it is the canonical idempotent unregister hook.
func (r *clientRegistry) remove(id ClientID) *Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byID[id]
	if !ok {
		return nil
	}
	delete(r.byID, id)
	delete(r.publics, id)
	if c.PlayerID != "" {
		// Only delete if it still points to this client (handles
		// last-connect-wins races where the index was already moved).
		if existing := r.byPlayerID[c.PlayerID]; existing == c {
			delete(r.byPlayerID, c.PlayerID)
		}
	}
	return c
}

// bindPlayer transitions a client from PUBLIC to PLAYER. If another
// client is already indexed under the same PlayerID, its ID is returned
// in `oldID` (with `hadOld=true`); the caller is then responsible for
// unregistering it (last-connect-wins, BR-U3-RECONNECT-1). The new
// client is *not* removed from publics if PlayerID is unchanged — but
// that case is impossible here because PUBLIC clients have no PlayerID.
func (r *clientRegistry) bindPlayer(c *Client, pid game.PlayerID) (oldID ClientID, hadOld bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.byPlayerID[pid]; exists && existing.ID != c.ID {
		oldID = existing.ID
		hadOld = true
		// Drop the prior client's PID index — caller will Unregister fully.
		delete(r.byPlayerID, pid)
	}

	delete(r.publics, c.ID)
	c.Kind = ClientPlayer
	c.PlayerID = pid
	r.byPlayerID[pid] = c
	return
}

// unbindPlayer demotes a PLAYER back to PUBLIC, dropping its byPlayerID
// index. Used by HostCloseRoom: the previous game's player tokens are
// invalidated, so the matching client should not remain indexed under
// the old PlayerID. Idempotent.
func (r *clientRegistry) unbindPlayer(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.PlayerID != "" {
		if existing := r.byPlayerID[c.PlayerID]; existing == c {
			delete(r.byPlayerID, c.PlayerID)
		}
		c.PlayerID = ""
	}
	c.Kind = ClientPublic
	r.publics[c.ID] = c
}

// byPlayerSafe returns the client currently indexed for pid (or nil).
// Safe under read lock — caller may use the returned *Client without
// further synchronization for read-only access.
func (r *clientRegistry) byPlayerSafe(pid game.PlayerID) *Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byPlayerID[pid]
}

// snapshotPublic returns a *new* slice containing every PUBLIC client.
// Safe to iterate without the registry lock.
func (r *clientRegistry) snapshotPublic() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Client, 0, len(r.publics))
	for _, c := range r.publics {
		out = append(out, c)
	}
	return out
}

// snapshotPlayers returns a *new* slice containing every PLAYER client.
func (r *clientRegistry) snapshotPlayers() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Client, 0, len(r.byPlayerID))
	for _, c := range r.byPlayerID {
		out = append(out, c)
	}
	return out
}

// all returns every registered client (PUBLIC + PLAYER). Used by Close
// to fan out cancellation.
func (r *clientRegistry) all() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Client, 0, len(r.byID))
	for _, c := range r.byID {
		out = append(out, c)
	}
	return out
}
