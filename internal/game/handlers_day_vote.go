package game

import "time"

// handleEndDiscussionEarly: host forces the DAY discussion timer to end and
// transitions to VOTE.
func (e *engine) handleEndDiscussionEarly(a EndDiscussionEarly) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseDay); err != nil {
		return e.state.Clone(), nil, err
	}
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	now := e.clock.Now()
	return e.transitionDayToVote(now)
}

// transitionDayToVote sets up VOTE round 1 from any DAY state and emits
// PhaseChanged.
func (e *engine) transitionDayToVote(now time.Time) (State, []EventEnvelope, error) {
	e.state.Phase = PhaseVote
	e.state.Deadline = time.Time{}
	e.state.Votes = map[PlayerID]PlayerID{}
	e.state.VoteRound = 1
	e.state.VoteCandidates = nil
	e.state.LastTickAt = now
	return e.state.Clone(), []EventEnvelope{pub(PhaseChanged{
		Phase: PhaseVote,
		Day:   e.state.Day,
	})}, nil
}

// handleVote records a single ballot. When all living players have voted,
// the result is tallied immediately. The latest submission per voter wins
// (BR-VOTE-2). An empty Target is an abstention; it is allowed in both
// VOTE and RECOUNT, never counts toward elimination, and survives recount
// candidate validation.
func (e *engine) handleVote(a SubmitVote) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseVote, PhaseRecount); err != nil {
		return e.state.Clone(), nil, err
	}
	abstain := a.Target == ""
	if abstain {
		if err := ensureAlive(&e.state, a.Voter); err != nil {
			return e.state.Clone(), nil, err
		}
	} else {
		if err := ensureAlive(&e.state, a.Voter, a.Target); err != nil {
			return e.state.Clone(), nil, err
		}
		if e.state.Phase == PhaseRecount {
			valid := false
			for _, c := range e.state.VoteCandidates {
				if c == a.Target {
					valid = true
					break
				}
			}
			if !valid {
				return e.state.Clone(), nil, errf(CodeInvalidTarget,
					"target %q is not in recount candidates", a.Target)
			}
		}
	}
	if e.state.Votes == nil {
		e.state.Votes = map[PlayerID]PlayerID{}
	}
	e.state.Votes[a.Voter] = a.Target

	if len(e.state.Votes) >= e.state.LiveCount() {
		events, err := e.tally()
		if err != nil {
			return e.state.Clone(), nil, err
		}
		return e.state.Clone(), events, nil
	}
	return e.state.Clone(), nil, nil
}
