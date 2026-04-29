package game

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

// Engine drives the game state machine. It is NOT safe for concurrent use:
// callers must serialize access (the SessionManager unit holds a single
// mutex / actor).
type Engine interface {
	// Start begins a fresh game with the given player IDs and options.
	// It assigns roles & keywords, sets Phase = INTRO, and emits the
	// initial event sequence (GameStarted, RoleRevealedToPlayer per player,
	// MafiaCohortRevealed, PhaseChanged{INTRO}, IntroSpeakerChanged).
	//
	// host should be a PlayerID present in players. The host plays as one
	// of the participants (Q-AD-6=B).
	Start(gameID string, host PlayerID, players []Player, opts Options) (State, []EventEnvelope, error)

	// Apply processes an external action and advances the state machine.
	// On error, the engine state is unchanged (NFR-U1-R2).
	Apply(action Action) (State, []EventEnvelope, error)

	// Tick advances time-driven phases (INTRO speaker rotation, DAY
	// discussion timer). It is idempotent for a given now value.
	Tick(now time.Time) (State, []EventEnvelope, error)

	// Snapshot returns a deep copy of the current state suitable for
	// persistence (NFR-U1-R5, NFR-U1-S1).
	Snapshot() State

	// Restore replaces the engine's state with a previously snapshotted
	// state (deep-copied internally to keep callers independent).
	Restore(s State) error
}

// engine is the default Engine implementation.
type engine struct {
	state    State
	clock    Clock
	rng      io.Reader
	assigner RoleAssigner
}

// New constructs an Engine with explicit dependencies. Production callers
// typically use NewDefault; tests inject their own clock and seeded reader.
func New(assigner RoleAssigner, clock Clock, rng io.Reader) Engine {
	return &engine{
		clock:    clock,
		rng:      rng,
		assigner: assigner,
	}
}

// NewDefault constructs an Engine wired with the real wall clock and
// crypto/rand reader. The supplied KeywordPool may be the built-in default
// (NewDefaultKeywordPool) or one loaded via LoadKeywordPool.
func NewDefault(pool KeywordPool) Engine {
	return New(NewAssigner(pool), realClock{}, rand.Reader)
}

// Snapshot implements Engine.
func (e *engine) Snapshot() State { return e.state.Clone() }

// Restore implements Engine.
func (e *engine) Restore(s State) error {
	if s.Phase == "" {
		return errf(CodeValidation, "restore: state has empty phase")
	}
	e.state = s.Clone()
	return nil
}

// Start implements Engine.
func (e *engine) Start(gameID string, host PlayerID, players []Player, opts Options) (State, []EventEnvelope, error) {
	if e.state.Phase != "" && e.state.Phase != PhaseLobby {
		return e.state, nil, errf(CodeWrongPhase, "Start called on phase=%s; expected empty or LOBBY", e.state.Phase)
	}
	if len(players) < 6 || len(players) > 12 {
		return e.state, nil, errf(CodeValidation, "player count must be 6..12; got %d", len(players))
	}
	if host != "" {
		hostFound := false
		for _, p := range players {
			if p.ID == host {
				hostFound = true
				break
			}
		}
		if !hostFound {
			return e.state, nil, errf(CodePermissionDenied, "host %q is not in players list", host)
		}
	}

	innerRand, err := newInnerRand(e.rng)
	if err != nil {
		return e.state, nil, errf(CodeValidation, "rng read: %v", err)
	}

	playerIDs := make([]PlayerID, len(players))
	for i, p := range players {
		playerIDs[i] = p.ID
	}
	asg, err := e.assigner.Assign(playerIDs, opts, innerRand)
	if err != nil {
		return e.state, nil, err
	}

	now := e.clock.Now()

	state := State{
		GameID:                 gameID,
		Phase:                  PhaseIntro,
		Day:                    1,
		Players:                make([]Player, len(players)),
		HostID:                 host,
		Settings:               opts,
		StartedAt:              now,
		IntroSpeakerIdx:        0,
		IntroSpeakerStartedAt:  now,
		MafiaRepresentativeID:  asg.RepresentativeID,
		Votes:                  map[PlayerID]PlayerID{},
		PoliceCheckedThisNight: false,
		LastTickAt:             now,
	}
	for i, p := range players {
		p.Alive = true
		p.Role = asg.PlayerRoles[p.ID]
		p.Keyword = asg.PlayerKeywords[p.ID]
		state.Players[i] = p
	}
	e.state = state

	events := make([]EventEnvelope, 0, len(players)+4)
	events = append(events, pub(GameStarted{State: state.Clone()}))
	for _, p := range state.Players {
		events = append(events, priv(RoleRevealedToPlayer{
			PlayerID: p.ID,
			Role:     p.Role,
			Keyword:  p.Keyword,
		}, p.ID))
	}
	if len(asg.MafiaIDs) > 0 {
		events = append(events, mafia(MafiaCohortRevealed{
			MafiaIDs:         asg.MafiaIDs,
			RepresentativeID: asg.RepresentativeID,
		}))
	}
	events = append(events, pub(PhaseChanged{
		Phase:    PhaseIntro,
		Day:      1,
		Deadline: time.Time{},
	}))
	events = append(events, pub(IntroSpeakerChanged{
		PlayerID:    state.Players[0].ID,
		SecondsLeft: opts.IntroSecondsPerPlayer,
	}))

	return e.state.Clone(), events, nil
}

// String returns a debug string for the engine state.
func (e *engine) String() string {
	return fmt.Sprintf("engine{phase=%s day=%d players=%d}", e.state.Phase, e.state.Day, len(e.state.Players))
}
