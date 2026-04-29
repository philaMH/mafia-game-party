package game

// checkEnd evaluates the win conditions and, if met, transitions to PhaseEnd
// and returns the GameEnded event envelope. The bool reports whether the
// game has ended on this call.
//
//   - liveMafiaCount == 0          -> citizens win
//   - liveMafiaCount >= citizenSide -> mafia win
func (e *engine) checkEnd() ([]EventEnvelope, bool) {
	if e.state.Phase == PhaseEnd {
		return nil, false
	}
	mafia := e.state.LiveMafiaCount()
	citizens := e.state.LiveCitizenSideCount()

	if mafia == 0 {
		return e.endGame(EndCitizenWin, TeamCitizen), true
	}
	if mafia >= citizens {
		return e.endGame(EndMafiaWin, TeamMafia), true
	}
	return nil, false
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
