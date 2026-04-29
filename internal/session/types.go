package session

import (
	"time"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
)

// Member is a single participant in the session.
//
// Token is the secret used by ResumePlayer; never log it. Connected is
// informational only — actual transport routing happens in U3.
type Member struct {
	ID        game.PlayerID `json:"id"`
	Name      string        `json:"name"`
	Token     string        `json:"token"`
	Connected bool          `json:"connected"`
	JoinedAt  time.Time     `json:"joinedAt"`
}

// JoinResult is returned by CreateSession / JoinPlayer / ResumePlayer.
//
// CurrentState is masked according to the viewer (PrivateView).
type JoinResult struct {
	PlayerID     game.PlayerID
	Token        string
	IsHost       bool
	CurrentState game.State
	YourRole     game.Role
	YourKeyword  string
	YourTeam     game.Team
	MafiaCohort  []game.PlayerID
}

// EventOut bundles a domain event with its rendered Korean announcement.
//
// Announcement may be nil when the event is private (RoleRevealedToPlayer,
// MafiaCohortRevealed, etc.) — the catalog returns an empty Announcement
// in that case and SessionManager drops the announcement field.
//
// State is the engine snapshot taken under the GM lock immediately after
// the action that produced this event. Subscribe handlers may consult
// it (for routing decisions) without acquiring further locks.
type EventOut struct {
	Envelope     game.EventEnvelope
	Announcement *announce.Announcement
	State        game.State
}

// EventHandler is the Subscribe callback. Handlers MUST be fast (run inside
// the GM lock) and MUST NOT call SessionManager methods (would deadlock).
type EventHandler func(EventOut)

// SessionOpts configures non-default knobs of the SessionManager.
//
// Zero-value SessionOpts is valid: TickInterval defaults to 1s, EventLog to
// off, MaxLobbySize to 12.
type SessionOpts struct {
	TickInterval time.Duration
	EventLog     bool
	MaxLobbySize int
	MinPlayers   int
}

// withDefaults fills in unset fields.
func (o SessionOpts) withDefaults() SessionOpts {
	if o.TickInterval <= 0 {
		o.TickInterval = time.Second
	}
	if o.MaxLobbySize <= 0 {
		o.MaxLobbySize = 12
	}
	if o.MinPlayers <= 0 {
		o.MinPlayers = 6
	}
	return o
}

// Session is the SessionManager's internal state. Only the manager itself
// reads/writes these fields and only under the GM mutex.
type Session struct {
	GameID  string
	Members map[game.PlayerID]*Member
	HostID  game.PlayerID
	Started bool

	StartedAt time.Time

	// Iteration 2 (FR-9 / FR-10 / FR-11). PendingOptions captures the host's
	// game configuration entered at OpenRoom time. RoomOpened gates JoinPlayer
	// under the v2 GM flow. The v1 CreateSession flow leaves these zero/false.
	PendingOptions game.Options
	RoomOpened     bool
}

// RoomSnapshot is a frozen view of the SessionManager's room and game
// state, captured atomically under the GM lock. Used by U3 to sync a
// freshly registered WebSocket client to the current room state
// (Iteration 3 — late-joiner resync).
//
// All fields are deep-copied at capture time; callers may mutate the
// returned State freely without affecting the engine.
type RoomSnapshot struct {
	// RoomOpened mirrors Session.RoomOpened.
	RoomOpened bool

	// Options is the host-configured game options captured at OpenRoom
	// time. Zero value when RoomOpened is false.
	Options game.Options

	// GameStarted reports whether the engine has progressed past LOBBY.
	// True when State.Phase is one of INTRO/NIGHT/DAY/VOTE/RECOUNT/END.
	GameStarted bool

	// State is a deep copy of the engine state. Always present; the engine
	// returns a zero-value State when no game is active.
	State game.State

	// HostOccupied reports whether the GM seat is currently held by a
	// live host token holder.
	HostOccupied bool
}
