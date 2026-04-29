package session

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

// SessionManager is the public facade onto U2.
//
// All methods are safe for concurrent use. The struct holds a single mutex
// (P-U2-6); contention is fine because Engine and PersistenceStore calls
// are bounded by NFR-U2-P2 (p99 < 100 ms).
type SessionManager interface {
	CreateSession(ctx context.Context, hostName string) (JoinResult, error)
	JoinPlayer(ctx context.Context, name string) (JoinResult, error)
	ResumePlayer(ctx context.Context, token string) (JoinResult, error)
	StartGame(ctx context.Context, hostID game.PlayerID, opts game.Options) ([]EventOut, error)
	SubmitAction(ctx context.Context, action game.Action) ([]EventOut, error)
	Tick(now time.Time)
	Subscribe(handler EventHandler) (unsubscribe func())
	Close(ctx context.Context) error

	// Iteration 2 (FR-9 / FR-10 / FR-11 / FR-12): GM-only host flow.
	ClaimHost(ctx context.Context) (HostToken, error)
	ReleaseHost(token HostToken)
	OpenRoom(ctx context.Context, token HostToken, opts game.Options) (game.State, error)
	HostStartGame(ctx context.Context, token HostToken) ([]EventOut, error)
	HostForceTerminate(ctx context.Context, token HostToken) ([]EventOut, error)
	// HostCloseRoom resets the room and engine to a clean LOBBY-ready
	// state so the host can open a new room. Allowed only after the game
	// reached END (or before any game ran but after OpenRoom — though
	// the typical caller is the END screen). All player tokens become
	// invalid; clients receive a room:closed event and must rejoin.
	HostCloseRoom(ctx context.Context, token HostToken) error

	// Snapshot returns a deep copy of the engine's current state, taken
	// under the GM lock. Callers (e.g., U3 WSHub for VisRoleMafia routing)
	// may use it without their own synchronization. Returns the zero
	// State when no game is active.
	Snapshot() game.State

	// RoomSnapshot returns a frozen view of the room+game state captured
	// atomically under the GM lock (Iteration 3). Used by U3 to sync a
	// freshly registered WebSocket client to the current room state
	// without depending on broadcast replay.
	RoomSnapshot() RoomSnapshot
}

// session is the SessionManager implementation. Fields are partitioned
// into "lock-required" (mutated under mu) and "lock-free" (set once at
// construction or via atomic).
type session struct {
	mu          sync.Mutex
	persistence persistence.PersistenceStore
	catalog     announce.AnnouncementCatalog
	engine      game.Engine
	clock       game.Clock
	rand        io.Reader

	sess     Session // GameID, Members, HostID, Started
	handlers []handlerEntry
	nextHID  uint64

	opts SessionOpts

	stopCh    chan struct{}
	stopOnce  sync.Once
	tickerWG  sync.WaitGroup
	closed    atomic.Bool
	systemMsg []announce.Announcement // pending system toasts (delivered with next dispatch)

	hostAuth *hostAuthority // Iteration 2 — GM seat lock
}

// handlerEntry pairs a Subscribe id with its callback so that
// unsubscribe can locate its entry.
type handlerEntry struct {
	id uint64
	fn EventHandler
}

// New constructs a SessionManager. It performs the boot-time auto-restore
// (P-U2-9, BR-U2-RESTORE-*), then starts the background tick loop.
//
// If clock is nil, a wall clock is used. If rng is nil, crypto/rand is used.
func New(
	store persistence.PersistenceStore,
	catalog announce.AnnouncementCatalog,
	engine game.Engine,
	clock game.Clock,
	rng io.Reader,
	opts SessionOpts,
) (SessionManager, error) {
	if store == nil {
		return nil, errors.New("session.New: nil persistence store")
	}
	if catalog == nil {
		return nil, errors.New("session.New: nil announcement catalog")
	}
	if engine == nil {
		return nil, errors.New("session.New: nil engine")
	}
	if clock == nil {
		clock = wallClock{}
	}
	if rng == nil {
		rng = rand.Reader
	}

	s := &session{
		persistence: store,
		catalog:     catalog,
		engine:      engine,
		clock:       clock,
		rand:        rng,
		opts:        opts.withDefaults(),
		stopCh:      make(chan struct{}),
		hostAuth:    newHostAuthority(),
	}
	s.sess = Session{Members: make(map[game.PlayerID]*Member)}

	if err := s.bootRestore(context.Background()); err != nil {
		return nil, fmt.Errorf("boot restore: %w", err)
	}

	s.tickerWG.Add(1)
	go s.tickLoop()

	return s, nil
}

// bootRestore loads the active snapshot (if any) and asks the engine to
// restore. On any failure, the snapshot is archived and the new session
// starts empty (P-U2-9).
func (s *session) bootRestore(ctx context.Context) error {
	snap, found, err := s.persistence.LoadActiveSnapshot(ctx)
	if err != nil {
		slog.Error("load snapshot failed; archiving", "err", err)
		return s.persistence.ArchiveCorrupt(ctx)
	}
	if !found {
		return nil
	}
	if err := s.engine.Restore(snap.State); err != nil {
		slog.Error("engine restore failed; archiving", "err", err)
		return s.persistence.ArchiveCorrupt(ctx)
	}

	members := make(map[game.PlayerID]*Member, len(snap.Members))
	for _, pm := range snap.Members {
		m := Member{
			ID:        pm.ID,
			Name:      pm.Name,
			Token:     pm.Token,
			Connected: false, // resume must reconnect
			JoinedAt:  pm.JoinedAt,
		}
		members[pm.ID] = &m
	}
	s.sess = Session{
		GameID:    snap.GameID,
		Members:   members,
		HostID:    snap.HostID,
		Started:   isActivePhase(snap.State.Phase),
		StartedAt: snap.State.StartedAt,
	}

	// Auto-finalize a session that ended right before a crash
	// (BR-U2-RESTORE-6): immediately persist the result and clear active.
	if snap.State.Phase == game.PhaseEnd {
		if err := s.persistence.SaveResultAndClearActive(ctx, buildResultFromState(s.sess, snap.State)); err != nil {
			slog.Warn("finalize end-state during restore failed", "err", err)
		}
		s.sess.Started = false
	}

	// Queue the restore notice so the next dispatch surfaces it.
	s.systemMsg = append(s.systemMsg, announce.SystemRestore())
	return nil
}

// Subscribe registers a handler. Callbacks run inside the GM lock; keep
// them fast.
func (s *session) Subscribe(handler EventHandler) (unsubscribe func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextHID++
	id := s.nextHID
	s.handlers = append(s.handlers, handlerEntry{id: id, fn: handler})
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, h := range s.handlers {
			if h.id == id {
				s.handlers = append(s.handlers[:i], s.handlers[i+1:]...)
				return
			}
		}
	}
}

// Close stops the tick loop, flushes a final snapshot if a game is active,
// and closes the persistence handle (BR-U2-CLOSE-*).
func (s *session) Close(ctx context.Context) error {
	if s.closed.Swap(true) {
		return nil
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.tickerWG.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sess.Started {
		snap := persistence.Snapshot{
			GameID:  s.sess.GameID,
			State:   s.engine.Snapshot(),
			Members: persistedMembers(s.sess.Members),
			HostID:  s.sess.HostID,
		}
		if err := s.persistence.SaveSnapshot(ctx, snap); err != nil {
			slog.Error("close: final SaveSnapshot failed", "err", err)
		}
	}
	return s.persistence.Close()
}

// Snapshot implements SessionManager. Returns the engine state under the
// GM lock so callers can read role/alive info race-free for routing.
func (s *session) Snapshot() game.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.engine.Snapshot()
}

// RoomSnapshot implements SessionManager. Captures Session room flags,
// engine state, and host occupancy in a single GM-locked critical
// section so the returned view is atomic. The engine state is already a
// deep copy (engine.Snapshot semantics); the caller may mutate it.
func (s *session) RoomSnapshot() RoomSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.engine.Snapshot()
	// Before Engine.Start the engine has no game and Snapshot returns a
	// zero-value State. Reconstruct a LOBBY view from the member map so
	// late-joiner / refresh resync (BR-U3-RESYNC) carries the roster.
	if !isActivePhase(state.Phase) && state.Phase != game.PhaseEnd {
		state = lobbyStateFromMembers(s.sess.GameID, s.sess.HostID, s.sess.Members)
	}
	return RoomSnapshot{
		RoomOpened:   s.sess.RoomOpened,
		Options:      s.sess.PendingOptions,
		GameStarted:  isActivePhase(state.Phase),
		State:        state,
		HostOccupied: s.hostAuth.IsClaimed(),
	}
}

// wallClock is a default Clock when callers do not supply one.
type wallClock struct{}

// Now implements game.Clock.
func (wallClock) Now() time.Time { return time.Now() }

// isActivePhase reports whether a phase indicates an in-progress game.
func isActivePhase(p game.Phase) bool {
	return p != game.PhaseLobby && p != game.PhaseEnd && p != ""
}
