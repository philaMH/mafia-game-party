package game

import "time"

// sealedEvent makes Event a sealed interface: only types in this package
// that embed sealedEvent can implement it.
type sealedEvent struct{}

// isEvent is the marker method that defines the Event interface seal.
func (sealedEvent) isEvent() {}

// Event is the sealed interface implemented by all domain events.
type Event interface {
	isEvent()
}

// Visibility indicates which clients should receive an event. The transport
// unit (U3) routes envelopes accordingly; the engine only annotates.
type Visibility int

// Visibility constants.
const (
	// VisPublic is delivered to all public viewers and all living players.
	VisPublic Visibility = iota
	// VisPlayer is delivered to a single PlayerID. The PlayerID field of the
	// envelope must be set.
	VisPlayer
	// VisRoleMafia is delivered to all living mafia players.
	VisRoleMafia
)

// EventEnvelope carries an Event together with routing metadata.
type EventEnvelope struct {
	Event      Event
	Visibility Visibility
	PlayerID   PlayerID // populated when Visibility == VisPlayer
}

// Engine event types follow.

// PlayerJoined is emitted by the Session unit when a member joins the LOBBY
// (host create or player join). It is purely a domain notification: the
// engine never produces it — U2's lifecycle.go publishes envelopes carrying
// this event so all subscribers (PUBLIC + PLAYER clients) learn membership
// changes in real time. See the LOBBY membership events plan (2026-04-27).
type PlayerJoined struct {
	sealedEvent
	PlayerID PlayerID
	Name     string
}

// GameStarted is emitted once when StartGame succeeds.
type GameStarted struct {
	sealedEvent
	State State
}

// PhaseChanged is emitted on every state-machine transition.
type PhaseChanged struct {
	sealedEvent
	Phase    Phase
	Day      int
	Deadline time.Time
}

// RoleRevealedToPlayer is private: it tells exactly one player their secret
// role and keyword.
type RoleRevealedToPlayer struct {
	sealedEvent
	PlayerID PlayerID
	Role     Role
	Keyword  string
}

// MafiaCohortRevealed is private to the mafia: it lists all mafia IDs and
// the chosen representative.
type MafiaCohortRevealed struct {
	sealedEvent
	MafiaIDs         []PlayerID
	RepresentativeID PlayerID
}

// IntroSpeakerChanged is emitted whenever the intro speaker rotates (or on
// initial entry into INTRO).
type IntroSpeakerChanged struct {
	sealedEvent
	PlayerID    PlayerID
	SecondsLeft int
}

// MafiaTargetSelected is private to the mafia: it shows the current
// representative's selection in real time.
type MafiaTargetSelected struct {
	sealedEvent
	RepresentativeID PlayerID
	Target           PlayerID
}

// PoliceResult is private to the police officer who triggered the check.
type PoliceResult struct {
	sealedEvent
	Police PlayerID
	Target PlayerID
	Team   Team
}

// NightStepChanged is emitted whenever the engine advances the night
// sub-step (MAFIA -> POLICE -> DOCTOR -> RESOLVED). Public visibility so
// the host PC can announce "이제 경찰이 눈을 뜹니다" etc. The Day field
// matches State.Day at emission time so late joiners can disambiguate
// across nights. Iteration 5: Deadline carries the wall-clock instant
// at which this step auto-advances so the public timer bar can render
// a synchronized countdown across all viewers.
type NightStepChanged struct {
	sealedEvent
	Step     NightStep
	Day      int
	Deadline time.Time
}

// GamePaused is emitted when the host suspends every active timer
// (Iteration 5 R4). Public visibility so spectators see the freeze.
type GamePaused struct {
	sealedEvent
	Phase Phase
}

// GameResumed is emitted when the host releases a pause. Deadline is
// the freshly shifted-forward instant for whichever timer was active
// (DAY discussion, NIGHT step, or zero for INTRO/none).
type GameResumed struct {
	sealedEvent
	Phase    Phase
	Deadline time.Time
}

// DeathAnnounced reports a player's death (mafia kill).
type DeathAnnounced struct {
	sealedEvent
	Victim PlayerID
}

// PeacefulNight indicates that no one died during the night.
type PeacefulNight struct {
	sealedEvent
}

// DiscussionTimerTick is emitted at threshold seconds remaining (typically
// 30, 10, 0) during the DAY phase.
type DiscussionTimerTick struct {
	sealedEvent
	SecondsLeft int
}

// VoteTallied reports the result of a vote count.
type VoteTallied struct {
	sealedEvent
	Counts     map[PlayerID]int
	Eliminated *PlayerID
	Recount    bool
}

// Eliminated reports a player's death-by-vote and reveals their role.
type Eliminated struct {
	sealedEvent
	PlayerID PlayerID
	Role     Role
}

// MafiaRepresentativeReassigned is private to the mafia.
type MafiaRepresentativeReassigned struct {
	sealedEvent
	OldID PlayerID
	NewID PlayerID
}

// GameEnded carries the final outcome and all roles for reveal.
type GameEnded struct {
	sealedEvent
	Winner    *Team
	EndReason EndReason
	Reveal    []Player
}

// VoiceToggled is emitted in response to ToggleVoice.
type VoiceToggled struct {
	sealedEvent
	On bool
}

// pub wraps an event with VisPublic.
func pub(e Event) EventEnvelope { return EventEnvelope{Event: e, Visibility: VisPublic} }

// priv wraps an event with VisPlayer for a single recipient.
func priv(e Event, pid PlayerID) EventEnvelope {
	return EventEnvelope{Event: e, Visibility: VisPlayer, PlayerID: pid}
}

// mafia wraps an event with VisRoleMafia.
func mafia(e Event) EventEnvelope { return EventEnvelope{Event: e, Visibility: VisRoleMafia} }
