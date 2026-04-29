package game

import "time"

// handleStartGame is the action-form entry into a fresh game. It is rarely
// used by U2 because Engine.Start is the typical entry point; included for
// completeness and to give the action enum full coverage.
func (e *engine) handleStartGame(a StartGame) (State, []EventEnvelope, error) {
	if e.state.Phase != PhaseLobby {
		return e.state.Clone(), nil, errf(CodeWrongPhase, "StartGame requires LOBBY; got %s", e.state.Phase)
	}
	if a.HostID != e.state.HostID {
		return e.state.Clone(), nil, errf(CodePermissionDenied, "sender is not host")
	}
	playerIDs := make([]PlayerID, len(e.state.Players))
	for i, p := range e.state.Players {
		playerIDs[i] = p.ID
	}
	innerRand, err := newInnerRand(e.rng)
	if err != nil {
		return e.state.Clone(), nil, errf(CodeValidation, "rng read: %v", err)
	}
	asg, err := e.assigner.Assign(playerIDs, a.Options, innerRand)
	if err != nil {
		return e.state.Clone(), nil, err
	}

	now := e.clock.Now()
	for i := range e.state.Players {
		e.state.Players[i].Alive = true
		e.state.Players[i].Role = asg.PlayerRoles[e.state.Players[i].ID]
		e.state.Players[i].Keyword = asg.PlayerKeywords[e.state.Players[i].ID]
	}
	e.state.Settings = a.Options
	e.state.Phase = PhaseIntro
	e.state.Day = 1
	e.state.IntroSpeakerIdx = 0
	e.state.IntroSpeakerStartedAt = now
	e.state.StartedAt = now
	e.state.MafiaRepresentativeID = asg.RepresentativeID
	e.state.LastTickAt = now

	events := make([]EventEnvelope, 0, len(e.state.Players)+4)
	events = append(events, pub(GameStarted{State: e.state.Clone()}))
	for _, p := range e.state.Players {
		events = append(events, priv(RoleRevealedToPlayer{
			PlayerID: p.ID, Role: p.Role, Keyword: p.Keyword,
		}, p.ID))
	}
	if len(asg.MafiaIDs) > 0 {
		events = append(events, mafia(MafiaCohortRevealed{
			MafiaIDs:         asg.MafiaIDs,
			RepresentativeID: asg.RepresentativeID,
		}))
	}
	events = append(events, pub(PhaseChanged{Phase: PhaseIntro, Day: 1}))
	events = append(events, pub(IntroSpeakerChanged{
		PlayerID:    e.state.Players[0].ID,
		SecondsLeft: a.Options.IntroSecondsPerPlayer,
	}))
	return e.state.Clone(), events, nil
}

// handleEndSelfIntro: the current intro speaker ends their own turn. Advances
// to the next speaker, or transitions to DAY 1 when the last speaker finishes.
// Only the player matching State.Players[IntroSpeakerIdx] may invoke it.
func (e *engine) handleEndSelfIntro(a EndSelfIntro) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseIntro); err != nil {
		return e.state.Clone(), nil, err
	}
	if e.state.IntroSpeakerIdx < 0 || e.state.IntroSpeakerIdx >= len(e.state.Players) {
		return e.state.Clone(), nil, errf(CodeValidation, "intro speaker index out of range: %d", e.state.IntroSpeakerIdx)
	}
	current := e.state.Players[e.state.IntroSpeakerIdx].ID
	if a.PlayerID != current {
		return e.state.Clone(), nil, errf(CodePermissionDenied, "EndSelfIntro: %q is not the current speaker (%q)", a.PlayerID, current)
	}
	now := e.clock.Now()
	if e.state.IntroSpeakerIdx < len(e.state.Players)-1 {
		e.state.IntroSpeakerIdx++
		e.state.IntroSpeakerStartedAt = now
		return e.state.Clone(), []EventEnvelope{pub(IntroSpeakerChanged{
			PlayerID:    e.state.Players[e.state.IntroSpeakerIdx].ID,
			SecondsLeft: e.state.Settings.IntroSecondsPerPlayer,
		})}, nil
	}
	return e.transitionIntroToDay(now)
}

// handleAdvanceIntro: host forces progression to the next intro speaker, or
// transitions into DAY 1 if the last speaker has just finished.
func (e *engine) handleAdvanceIntro(a AdvanceIntro) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseIntro); err != nil {
		return e.state.Clone(), nil, err
	}
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	now := e.clock.Now()
	if e.state.IntroSpeakerIdx < len(e.state.Players)-1 {
		e.state.IntroSpeakerIdx++
		e.state.IntroSpeakerStartedAt = now
		return e.state.Clone(), []EventEnvelope{pub(IntroSpeakerChanged{
			PlayerID:    e.state.Players[e.state.IntroSpeakerIdx].ID,
			SecondsLeft: e.state.Settings.IntroSecondsPerPlayer,
		})}, nil
	}
	return e.transitionIntroToDay(now)
}

// transitionIntroToDay starts Day 1 directly from INTRO. Day 1 has no prior
// night, so no DeathAnnounced / PeacefulNight is emitted — only the DAY
// PhaseChanged with the DiscussionSeconds deadline. The first NIGHT will
// be entered after the Day 1 vote tally.
func (e *engine) transitionIntroToDay(now time.Time) (State, []EventEnvelope, error) {
	e.state.Phase = PhaseDay
	e.state.Deadline = now.Add(time.Duration(e.state.Settings.DiscussionSeconds) * time.Second)
	e.state.PendingMafiaTarget = nil
	e.state.PendingDoctorTarget = nil
	e.state.PendingPoliceTarget = nil
	e.state.PoliceCheckedThisNight = false
	e.state.NightStep = ""
	e.state.LastTickAt = now
	return e.state.Clone(), []EventEnvelope{pub(PhaseChanged{
		Phase:    PhaseDay,
		Day:      e.state.Day,
		Deadline: e.state.Deadline,
	})}, nil
}

// handleToggleVoice flips the public-screen TTS preference.
func (e *engine) handleToggleVoice(a ToggleVoice) (State, []EventEnvelope, error) {
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	e.state.Settings.AnnouncementVoiceOn = a.On
	return e.state.Clone(), []EventEnvelope{pub(VoiceToggled{On: a.On})}, nil
}

// handlePauseGame freezes the active timer (Iteration 5 R4 / Q5=B). It
// only records the pause instant; deadline shifting is performed on
// Resume so a never-resumed pause cannot strand a deadline in the past.
// Idempotent — re-issuing while already paused is a no-op.
func (e *engine) handlePauseGame(a PauseGame) (State, []EventEnvelope, error) {
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	if !canPause(e.state.Phase) {
		return e.state.Clone(), nil, errf(CodeWrongPhase,
			"cannot pause during phase %s", e.state.Phase)
	}
	if e.state.Phase == PhaseNight && e.state.NightStep == NightStepIntro {
		return e.state.Clone(), nil, errf(CodeWrongPhase,
			"cannot pause during night intro")
	}
	if e.state.Paused {
		return e.state.Clone(), nil, nil
	}
	e.state.Paused = true
	e.state.PausedAt = e.clock.Now()
	return e.state.Clone(), []EventEnvelope{pub(GamePaused{Phase: e.state.Phase})}, nil
}

// handleResumeGame releases a pause and shifts every active timer
// forward by (now - PausedAt) so each role/speaker keeps the time they
// had remaining when the host pressed Pause. Idempotent — re-issuing
// while not paused is a no-op.
func (e *engine) handleResumeGame(a ResumeGame) (State, []EventEnvelope, error) {
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	if !e.state.Paused {
		return e.state.Clone(), nil, nil
	}
	now := e.clock.Now()
	shift := now.Sub(e.state.PausedAt)
	if shift < 0 {
		shift = 0
	}
	switch e.state.Phase {
	case PhaseIntro:
		e.state.IntroSpeakerStartedAt = e.state.IntroSpeakerStartedAt.Add(shift)
	case PhaseDay:
		if !e.state.Deadline.IsZero() {
			e.state.Deadline = e.state.Deadline.Add(shift)
		}
	case PhaseNight:
		if !e.state.NightStepDeadline.IsZero() {
			e.state.NightStepDeadline = e.state.NightStepDeadline.Add(shift)
		}
	}
	e.state.Paused = false
	e.state.PausedAt = time.Time{}
	// Reset LastTickAt so the next Tick re-evaluates the (shifted)
	// deadline instead of treating the pause window as elapsed.
	e.state.LastTickAt = now

	var deadline time.Time
	switch e.state.Phase {
	case PhaseDay:
		deadline = e.state.Deadline
	case PhaseNight:
		deadline = e.state.NightStepDeadline
	}
	return e.state.Clone(), []EventEnvelope{pub(GameResumed{
		Phase:    e.state.Phase,
		Deadline: deadline,
	})}, nil
}

// canPause reports whether PauseGame/ResumeGame are meaningful for the
// given phase. VOTE/RECOUNT have no timer; LOBBY/END are out of scope.
func canPause(p Phase) bool {
	return p == PhaseIntro || p == PhaseDay || p == PhaseNight
}

// handleForceEnd ends the game immediately with EndReason = HOST_FORCE_END.
func (e *engine) handleForceEnd(a ForceEndGame) (State, []EventEnvelope, error) {
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	if e.state.Phase == PhaseEnd {
		return e.state.Clone(), nil, errf(CodeWrongPhase, "already ended")
	}
	reason := EndHostForceEnd
	e.state.Phase = PhaseEnd
	e.state.EndReason = &reason
	e.state.Winner = nil
	reveal := make([]Player, len(e.state.Players))
	copy(reveal, e.state.Players)
	return e.state.Clone(), []EventEnvelope{pub(GameEnded{
		Winner:    nil,
		EndReason: reason,
		Reveal:    reveal,
	})}, nil
}
