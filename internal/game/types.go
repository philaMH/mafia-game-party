package game

import "time"

// PlayerID uniquely identifies a player within a single game. It is opaque
// (typically a UUID or stable token) and is allocated by the SessionManager
// (U2) before being passed into the engine.
type PlayerID string

// Role is a player's secret role assigned at game start.
type Role string

// Role constants.
const (
	RoleMafia   Role = "MAFIA"
	RoleCitizen Role = "CITIZEN"
	RoleDoctor  Role = "DOCTOR"
	RolePolice  Role = "POLICE"
)

// Team is the win-condition side a role belongs to. Doctor, Police, and
// Citizen are all on the CITIZEN team.
type Team string

// Team constants.
const (
	TeamMafia   Team = "MAFIA"
	TeamCitizen Team = "CITIZEN"
)

// TeamOf returns the team a role belongs to.
func TeamOf(r Role) Team {
	if r == RoleMafia {
		return TeamMafia
	}
	return TeamCitizen
}

// Phase is the current stage of the game state machine.
type Phase string

// Phase constants.
const (
	PhaseLobby   Phase = "LOBBY"
	PhaseIntro   Phase = "INTRO"
	PhaseNight   Phase = "NIGHT"
	PhaseDay     Phase = "DAY"
	PhaseVote    Phase = "VOTE"
	PhaseRecount Phase = "RECOUNT"
	PhaseEnd     Phase = "END"
)

// NightStep tracks which role is currently acting during PhaseNight. The
// engine forces strict ordering MAFIA -> POLICE -> DOCTOR; submissions for
// any other step are rejected. NightStepResolved is the terminal value used
// briefly while resolveNight() runs; outside PhaseNight the field is empty.
type NightStep string

// NightStep constants.
const (
	NightStepMafia    NightStep = "MAFIA"
	NightStepPolice   NightStep = "POLICE"
	NightStepDoctor   NightStep = "DOCTOR"
	NightStepResolved NightStep = "RESOLVED"
)

// EndReason explains why a game ended.
type EndReason string

// EndReason constants.
const (
	EndMafiaWin     EndReason = "MAFIA_WIN"
	EndCitizenWin   EndReason = "CITIZEN_WIN"
	EndHostForceEnd EndReason = "HOST_FORCE_END"
)

// Player is a single participant in a game. Role and Keyword are secret and
// must be masked when sending state to viewers other than the owner.
type Player struct {
	ID      PlayerID `json:"id"`
	Name    string   `json:"name"`
	Alive   bool     `json:"alive"`
	Role    Role     `json:"role,omitempty"`
	Keyword string   `json:"keyword,omitempty"`
}

// Options carries host-tunable game parameters.
type Options struct {
	MafiaCount            int  `json:"mafiaCount"`
	MaxPlayers            int  `json:"maxPlayers"`
	IntroSecondsPerPlayer int  `json:"introSecondsPerPlayer"`
	DiscussionSeconds     int  `json:"discussionSeconds"`
	// NightMafiaSeconds is the fixed duration of the MAFIA NightStep.
	// 0 or negative falls back to the package default (30s) at runtime.
	NightMafiaSeconds int `json:"nightMafiaSeconds"`
	// NightPoliceSeconds is the fixed duration of the POLICE NightStep
	// (10s default). Held even when no police is alive — Iteration 5 R1.
	NightPoliceSeconds int `json:"nightPoliceSeconds"`
	// NightDoctorSeconds is the fixed duration of the DOCTOR NightStep
	// (10s default). Held even when no doctor is alive — Iteration 5 R1.
	NightDoctorSeconds    int  `json:"nightDoctorSeconds"`
	DoctorSelfHealAllowed bool `json:"doctorSelfHealAllowed"`
	AnnouncementVoiceOn   bool `json:"announcementVoiceOn"`
}

// Default night step durations applied when Options leaves them at zero.
// Iteration 5: the engine treats every NightStep as a fixed wall-clock
// window so role-death cannot leak through accelerated transitions.
const (
	defaultNightMafiaSeconds  = 30
	defaultNightPoliceSeconds = 10
	defaultNightDoctorSeconds = 10
)

// DefaultOptions returns the recommended defaults derived from
// Functional Design Q-FD-U1-2/3/7 and FR-8.5.
func DefaultOptions(playerCount int) Options {
	return Options{
		MafiaCount:            recommendedMafiaCount(playerCount),
		IntroSecondsPerPlayer: 20,
		DiscussionSeconds:     180,
		NightMafiaSeconds:     defaultNightMafiaSeconds,
		NightPoliceSeconds:    defaultNightPoliceSeconds,
		NightDoctorSeconds:    defaultNightDoctorSeconds,
		DoctorSelfHealAllowed: true,
		AnnouncementVoiceOn:   true,
	}
}

// nightStepSeconds returns the configured wall-clock duration for the given
// NightStep. Zero / negative fields fall back to the package default so old
// snapshots and bare Options literals keep working.
func nightStepSeconds(opts Options, step NightStep) int {
	var v, def int
	switch step {
	case NightStepMafia:
		v, def = opts.NightMafiaSeconds, defaultNightMafiaSeconds
	case NightStepPolice:
		v, def = opts.NightPoliceSeconds, defaultNightPoliceSeconds
	case NightStepDoctor:
		v, def = opts.NightDoctorSeconds, defaultNightDoctorSeconds
	default:
		return 0
	}
	if v <= 0 {
		return def
	}
	return v
}

// recommendedMafiaCount mirrors the "표준안" baseline from
// domain-entities.md §3.
func recommendedMafiaCount(playerCount int) int {
	switch {
	case playerCount <= 6:
		return 1
	case playerCount <= 9:
		return 2
	default:
		return 3
	}
}

// PendingActions summarizes the un-applied night actions of the current
// NIGHT phase. The fields mirror State.PendingMafiaTarget /
// PendingDoctorTarget / PendingPoliceTarget for read-only views.
type PendingActions struct {
	MafiaTarget  *PlayerID `json:"mafiaTarget,omitempty"`
	DoctorTarget *PlayerID `json:"doctorTarget,omitempty"`
	PoliceTarget *PlayerID `json:"policeTarget,omitempty"`
}

// PoliceCheckRecord is one historical investigation result. Stored on State
// so a returning police officer (or a fresh snapshot) can replay every prior
// finding. The transport layer must mask this slice to non-police viewers.
type PoliceCheckRecord struct {
	Day    int      `json:"day"`
	Target PlayerID `json:"target"`
	Team   Team     `json:"team"`
}

// State is the entire serialized state of a single mafia game. It is the
// source of truth that callers persist via Engine.Snapshot.
type State struct {
	GameID    string    `json:"gameId"`
	Phase     Phase     `json:"phase"`
	Day       int       `json:"day"`
	Players   []Player  `json:"players"`
	HostID    PlayerID  `json:"hostId"`
	Settings  Options   `json:"settings"`
	StartedAt time.Time `json:"startedAt"`
	Deadline  time.Time `json:"deadline"`

	// INTRO progress
	IntroSpeakerIdx       int       `json:"introSpeakerIdx"`
	IntroSpeakerStartedAt time.Time `json:"introSpeakerStartedAt"`

	// Night accumulators (current NIGHT only)
	MafiaRepresentativeID  PlayerID  `json:"mafiaRepresentativeId,omitempty"`
	PendingMafiaTarget     *PlayerID `json:"pendingMafiaTarget,omitempty"`
	PendingDoctorTarget    *PlayerID `json:"pendingDoctorTarget,omitempty"`
	PendingPoliceTarget    *PlayerID `json:"pendingPoliceTarget,omitempty"`
	PoliceCheckedThisNight bool      `json:"policeCheckedThisNight"`
	// NightStep is the role currently expected to submit. Empty outside
	// PhaseNight. Engine forces MAFIA -> POLICE -> DOCTOR ordering.
	NightStep NightStep `json:"nightStep,omitempty"`
	// NightStepDeadline is the wall-clock instant at which the current
	// NightStep auto-advances. Iteration 5 R2: the deadline is the only
	// trigger for night-step transitions; action submission no longer
	// shortens it. Zero outside PhaseNight.
	NightStepDeadline time.Time `json:"nightStepDeadline,omitempty"`

	// Paused is true while the host has frozen all active timers
	// (Iteration 5 R4). PausedAt records the instant Pause was issued so
	// Resume can shift forward every active timer by the elapsed delta.
	Paused   bool      `json:"paused,omitempty"`
	PausedAt time.Time `json:"pausedAt,omitempty"`

	// PoliceHistory accumulates every successful PoliceCheck across the
	// whole game. The transport layer masks it to police-only views; it is
	// safe to embed in the canonical State because U3 redacts before it
	// reaches non-police clients.
	PoliceHistory []PoliceCheckRecord `json:"policeHistory,omitempty"`

	// Vote accumulators (current voting round only)
	Votes          map[PlayerID]PlayerID `json:"votes"`
	VoteRound      int                   `json:"voteRound"`
	VoteCandidates []PlayerID            `json:"voteCandidates,omitempty"`

	// End fields (set when Phase == PhaseEnd)
	Winner    *Team      `json:"winner,omitempty"`
	EndReason *EndReason `json:"endReason,omitempty"`

	// Last Tick processed time (used for idempotent Tick)
	LastTickAt time.Time `json:"lastTickAt"`
}

// Pending returns a snapshot of the current night accumulators.
func (s State) Pending() PendingActions {
	return PendingActions{
		MafiaTarget:  s.PendingMafiaTarget,
		DoctorTarget: s.PendingDoctorTarget,
		PoliceTarget: s.PendingPoliceTarget,
	}
}

// FindPlayer returns a pointer to the player with the given ID and a bool
// indicating whether it was found. The returned pointer is into the receiver's
// Players slice; use only for reads on State copies.
func (s *State) FindPlayer(id PlayerID) (*Player, bool) {
	for i := range s.Players {
		if s.Players[i].ID == id {
			return &s.Players[i], true
		}
	}
	return nil, false
}

// LiveCount returns the number of players with Alive == true.
func (s *State) LiveCount() int {
	n := 0
	for _, p := range s.Players {
		if p.Alive {
			n++
		}
	}
	return n
}

// LiveMafiaCount returns the number of living mafia players.
func (s *State) LiveMafiaCount() int {
	n := 0
	for _, p := range s.Players {
		if p.Alive && p.Role == RoleMafia {
			n++
		}
	}
	return n
}

// LiveCitizenSideCount returns the number of living citizen-side players
// (CITIZEN, DOCTOR, POLICE).
func (s *State) LiveCitizenSideCount() int {
	n := 0
	for _, p := range s.Players {
		if p.Alive && p.Role != RoleMafia {
			n++
		}
	}
	return n
}

// HasLivingDoctor reports whether at least one doctor is alive.
func (s *State) HasLivingDoctor() bool {
	for _, p := range s.Players {
		if p.Alive && p.Role == RoleDoctor {
			return true
		}
	}
	return false
}

// HasLivingPolice reports whether at least one police officer is alive.
func (s *State) HasLivingPolice() bool {
	for _, p := range s.Players {
		if p.Alive && p.Role == RolePolice {
			return true
		}
	}
	return false
}

// LivingMafiaIDs returns the IDs of living mafia players in
// state.Players order. Used for representative reassignment.
func (s *State) LivingMafiaIDs() []PlayerID {
	out := make([]PlayerID, 0, 3)
	for _, p := range s.Players {
		if p.Alive && p.Role == RoleMafia {
			out = append(out, p.ID)
		}
	}
	return out
}
