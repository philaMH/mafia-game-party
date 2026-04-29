package game

// sealedAction makes Action a sealed interface: only types in this package
// that embed sealedAction can implement it.
type sealedAction struct{}

// isAction is the marker method that defines the Action interface seal.
func (sealedAction) isAction() {}

// Action is the sealed interface implemented by all valid engine inputs.
// Use a type switch in Engine.Apply to dispatch to the correct handler.
type Action interface {
	isAction()
}

// StartGame triggers Lobby -> Intro transition. Allowed only for the host
// when the game is in Phase = LOBBY.
type StartGame struct {
	sealedAction
	HostID  PlayerID
	Options Options
}

// AdvanceIntro forces progression to the next intro speaker (or to NIGHT if
// the last speaker has finished). Host-only.
type AdvanceIntro struct {
	sealedAction
	HostID PlayerID
}

// EndSelfIntro is the player-initiated trigger to advance the intro
// round-robin. Allowed only when Phase == INTRO and PlayerID equals the
// current speaker (Players[IntroSpeakerIdx].ID).
type EndSelfIntro struct {
	sealedAction
	PlayerID PlayerID
}

// SubmitMafiaKill records the mafia representative's kill target during
// NIGHT. Only the player whose ID equals MafiaRepresentativeID may submit it.
type SubmitMafiaKill struct {
	sealedAction
	Mafia  PlayerID
	Target PlayerID
}

// SubmitDoctorHeal records the doctor's protect target. Self-heal is
// permitted only when Settings.DoctorSelfHealAllowed is true.
type SubmitDoctorHeal struct {
	sealedAction
	Doctor PlayerID
	Target PlayerID
}

// SubmitPoliceCheck triggers a one-shot per-night police investigation.
// Returns a private PoliceResult event that is visible only to the police.
type SubmitPoliceCheck struct {
	sealedAction
	Police PlayerID
	Target PlayerID
}

// EndNightEarly is the host's manual command to end NIGHT and apply
// whichever night actions were submitted. Required because NIGHT has no
// auto-deadline (Q-FD-U1-12=B).
type EndNightEarly struct {
	sealedAction
	HostID PlayerID
}

// EndDiscussionEarly is the host's manual command to end the DAY discussion
// timer early and transition to VOTE.
type EndDiscussionEarly struct {
	sealedAction
	HostID PlayerID
}

// SubmitVote records a single ballot from a living player during VOTE or
// RECOUNT. The latest submission wins (last-write-wins per voter).
type SubmitVote struct {
	sealedAction
	Voter  PlayerID
	Target PlayerID
}

// ToggleVoice flips the public-screen TTS announcement on/off. Host-only.
type ToggleVoice struct {
	sealedAction
	HostID PlayerID
	On     bool
}

// ForceEndGame ends the game immediately with EndReason = HOST_FORCE_END.
// Host-only; rejected when the game is already in PhaseEnd.
type ForceEndGame struct {
	sealedAction
	HostID PlayerID
}

// PauseGame freezes the active timer (INTRO speaker, DAY discussion, or
// NIGHT step) until ResumeGame is issued. Host-only; valid in
// {INTRO, DAY, NIGHT}. Idempotent: re-issuing while already paused is a
// no-op so accidental double-clicks are harmless. (Iteration 5 R4 / Q5=B)
type PauseGame struct {
	sealedAction
	HostID PlayerID
}

// ResumeGame releases a previously-issued pause and shifts every active
// timer forward by (now - PausedAt). Host-only. Idempotent: re-issuing
// while not paused is a no-op.
type ResumeGame struct {
	sealedAction
	HostID PlayerID
}
