package game

// tally counts votes, decides elimination or recount, applies side effects,
// and returns the resulting events. Caller has already checked that all
// living players have voted.
//
// Round 1 (Phase == VOTE):
//   - single max  -> eliminate, advance to NIGHT (or END)
//   - tie         -> RECOUNT with VoteCandidates set to tied IDs
//
// Round 2 (Phase == RECOUNT):
//   - single max  -> eliminate, advance to NIGHT (or END)
//   - tie         -> no elimination, advance to NIGHT (or END)
func (e *engine) tally() ([]EventEnvelope, error) {
	counts := make(map[PlayerID]int, len(e.state.Votes))
	for _, t := range e.state.Votes {
		// Abstentions (Target=="") are recorded for transparency on the wire
		// (clients may show abstain counts) but never contribute to
		// elimination tallies.
		if t == "" {
			continue
		}
		counts[t]++
	}

	maxCount := 0
	var topCandidates []PlayerID
	// Iterate Players for deterministic ordering of topCandidates.
	for _, p := range e.state.Players {
		c := counts[p.ID]
		if c == 0 {
			continue
		}
		switch {
		case c > maxCount:
			maxCount = c
			topCandidates = []PlayerID{p.ID}
		case c == maxCount:
			topCandidates = append(topCandidates, p.ID)
		}
	}

	// Everyone abstained (or only abstentions remain in scope): no candidate
	// to eliminate, no recount — proceed to NIGHT.
	if maxCount == 0 {
		events := []EventEnvelope{pub(VoteTallied{
			Counts:     counts,
			Eliminated: nil,
			Recount:    false,
		})}
		events = append(events, e.transitionVoteToNight()...)
		return events, nil
	}

	if e.state.VoteRound == 1 {
		if len(topCandidates) == 1 {
			elim := topCandidates[0]
			events := []EventEnvelope{pub(VoteTallied{
				Counts:     counts,
				Eliminated: &elim,
				Recount:    false,
			})}
			events = append(events, e.applyElimination(elim)...)
			return events, nil
		}
		// Tie -> RECOUNT.
		e.state.Phase = PhaseRecount
		e.state.VoteRound = 2
		e.state.VoteCandidates = append([]PlayerID(nil), topCandidates...)
		e.state.Votes = map[PlayerID]PlayerID{}
		events := []EventEnvelope{
			pub(VoteTallied{Counts: counts, Eliminated: nil, Recount: true}),
			pub(PhaseChanged{Phase: PhaseRecount, Day: e.state.Day}),
		}
		return events, nil
	}

	// Round 2 (RECOUNT)
	if len(topCandidates) == 1 {
		elim := topCandidates[0]
		events := []EventEnvelope{pub(VoteTallied{
			Counts:     counts,
			Eliminated: &elim,
			Recount:    false,
		})}
		events = append(events, e.applyElimination(elim)...)
		return events, nil
	}
	// Tie again -> no elimination.
	events := []EventEnvelope{pub(VoteTallied{
		Counts:     counts,
		Eliminated: nil,
		Recount:    false,
	})}
	events = append(events, e.transitionVoteToNight()...)
	return events, nil
}

// applyElimination marks the player dead, emits Eliminated, possibly
// reassigns the mafia representative, then transitions to NIGHT or END.
func (e *engine) applyElimination(id PlayerID) []EventEnvelope {
	events := make([]EventEnvelope, 0, 3)
	p, ok := e.state.FindPlayer(id)
	if !ok {
		return events
	}
	p.Alive = false
	events = append(events, pub(Eliminated{PlayerID: id, Role: p.Role}))
	if id == e.state.MafiaRepresentativeID {
		events = append(events, e.reassignMafiaRepresentative(id)...)
	}
	if endEv, ok := e.checkEnd(); ok {
		events = append(events, endEv...)
		return events
	}
	events = append(events, e.transitionVoteToNight()...)
	return events
}

// transitionVoteToNight resets vote state and advances to NIGHT, kicking
// off the MAFIA -> POLICE -> DOCTOR sub-step machine via enterNight().
func (e *engine) transitionVoteToNight() []EventEnvelope {
	return e.enterNight()
}
