package game

import "time"

// evaluateEnd reports whether the current state already meets a win
// condition without mutating anything. Iteration 9 introduced this pure
// inspection helper so both immediate-end (HOST_FORCE_END) and deferred-end
// (vote/night) paths share the same rules; checkEnd / scheduleGameEnd both
// call into it.
//
//   - liveMafiaCount == 0           -> citizens win
//   - liveMafiaCount >= citizenSide -> mafia win
func (e *engine) evaluateEnd() (EndReason, Team, bool) {
	if e.state.Phase == PhaseEnd {
		return "", "", false
	}
	mafia := e.state.LiveMafiaCount()
	citizens := e.state.LiveCitizenSideCount()
	switch {
	case mafia == 0:
		return EndCitizenWin, TeamCitizen, true
	case mafia >= citizens:
		return EndMafiaWin, TeamMafia, true
	}
	return "", "", false
}

// checkEnd evaluates the win conditions and, if met, immediately transitions
// to PhaseEnd and returns the GameEnded event envelope. The bool reports
// whether the game has ended on this call. Used by paths that intentionally
// bypass the result-announcement buffer (e.g. test fixtures that restore a
// terminal state directly).
func (e *engine) checkEnd() ([]EventEnvelope, bool) {
	reason, winner, ok := e.evaluateEnd()
	if !ok {
		return nil, false
	}
	return e.endGame(reason, winner), true
}

// scheduleGameEnd defers GameEnded emission by defaultFinalResultBufferSeconds
// (Iteration 9 FR-2). Phase is preserved so the caller's just-emitted result
// event (Eliminated / DeathAnnounced / PeacefulNight) stays on screen during
// the buffer. Idempotent: a second schedule before firePendingEnd is a no-op,
// preventing the deadline from being reset by spurious end conditions.
func (e *engine) scheduleGameEnd(reason EndReason, winner Team) {
	if e.state.PendingGameEnd != nil {
		return
	}
	w := winner
	e.state.PendingGameEnd = &PendingGameEnd{
		Reason:   reason,
		Winner:   &w,
		Deadline: e.clock.Now().Add(time.Duration(defaultFinalResultBufferSeconds) * time.Second),
	}
}

// firePendingEnd consumes the deferred end record set by scheduleGameEnd
// and emits GameEnded while transitioning to PhaseEnd. Called from Tick
// once the buffer deadline has been reached.
func (e *engine) firePendingEnd(now time.Time) (State, []EventEnvelope, error) {
	pending := e.state.PendingGameEnd
	e.state.PendingGameEnd = nil
	e.state.LastTickAt = now
	if pending.Winner == nil {
		return e.state.Clone(), nil, errf(CodeValidation, "firePendingEnd: nil winner")
	}
	return e.state.Clone(), e.endGame(pending.Reason, *pending.Winner), nil
}

// endGame finalizes state, populates Winner/EndReason, and emits GameEnded.
func (e *engine) endGame(reason EndReason, winner Team) []EventEnvelope {
	w := winner
	e.state.Phase = PhaseEnd
	e.state.EndReason = &reason
	e.state.Winner = &w
	reveal := make([]Player, len(e.state.Players))
	copy(reveal, e.state.Players)
	return []EventEnvelope{pub(GameEnded{
		Winner:    &w,
		EndReason: reason,
		Reveal:    reveal,
	})}
}
