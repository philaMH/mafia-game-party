package game

// Clone returns a deep copy of the State. Mutating the returned value (or any
// of its slices/maps/pointers) does not affect the receiver.
//
// Required by Engine.Snapshot to ensure that callers cannot accidentally
// mutate engine state through the returned snapshot. See NFR-design pattern P2.
func (s State) Clone() State {
	out := s

	if s.Players != nil {
		out.Players = make([]Player, len(s.Players))
		copy(out.Players, s.Players)
	}

	if s.Votes != nil {
		out.Votes = make(map[PlayerID]PlayerID, len(s.Votes))
		for k, v := range s.Votes {
			out.Votes[k] = v
		}
	}

	if s.VoteCandidates != nil {
		out.VoteCandidates = make([]PlayerID, len(s.VoteCandidates))
		copy(out.VoteCandidates, s.VoteCandidates)
	}

	if s.PoliceHistory != nil {
		out.PoliceHistory = make([]PoliceCheckRecord, len(s.PoliceHistory))
		copy(out.PoliceHistory, s.PoliceHistory)
	}

	if s.PendingMafiaTarget != nil {
		v := *s.PendingMafiaTarget
		out.PendingMafiaTarget = &v
	}
	if s.PendingDoctorTarget != nil {
		v := *s.PendingDoctorTarget
		out.PendingDoctorTarget = &v
	}
	if s.PendingPoliceTarget != nil {
		v := *s.PendingPoliceTarget
		out.PendingPoliceTarget = &v
	}
	if s.Winner != nil {
		v := *s.Winner
		out.Winner = &v
	}
	if s.EndReason != nil {
		v := *s.EndReason
		out.EndReason = &v
	}

	return out
}
