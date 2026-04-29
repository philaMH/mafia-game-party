package game

import "time"

// enterNight is called whenever the engine transitions into PhaseNight
// (from VOTE / RECOUNT). It clears per-night accumulators, sets NightStep
// to INTRO, and emits PhaseChanged{NIGHT} followed by NightStepChanged{INTRO}
// with a 5s deadline. Tick advances INTRO -> MAFIA when the buffer expires.
//
// Iteration 5: dead-role steps are NO LONGER auto-skipped. Each step is
// held for its configured duration so role deaths cannot leak through
// abnormally fast transitions (R1).
//
// Iteration 8: the new INTRO step gives the host's `phase.night` cue time
// to finish before mafia time begins. Catalog renders INTRO as silent so
// only the existing `phase.night` cue is heard.
func (e *engine) enterNight() []EventEnvelope {
	now := e.clock.Now()
	e.state.Phase = PhaseNight
	e.state.Deadline = time.Time{}
	e.state.Votes = map[PlayerID]PlayerID{}
	e.state.VoteRound = 0
	e.state.VoteCandidates = nil
	e.state.PendingMafiaTarget = nil
	e.state.PendingDoctorTarget = nil
	e.state.PendingPoliceTarget = nil
	e.state.PoliceCheckedThisNight = false
	e.state.NightStep = ""
	e.state.NightStepDeadline = time.Time{}
	e.state.LastTickAt = now

	events := []EventEnvelope{pub(PhaseChanged{Phase: PhaseNight, Day: e.state.Day})}
	events = append(events, e.beginNightStep(NightStepIntro, now)...)
	return events
}

// beginNightStep records a new night sub-step, computes its wall-clock
// deadline from the configured per-step duration, and emits the public
// NightStepChanged event so every viewer (including dead players) sees
// the same countdown. No auto-skip is performed: the dead-role step is
// announced and held for its full duration.
func (e *engine) beginNightStep(step NightStep, startedAt time.Time) []EventEnvelope {
	e.state.NightStep = step
	seconds := nightStepSeconds(e.state.Settings, step)
	deadline := startedAt.Add(time.Duration(seconds) * time.Second)
	e.state.NightStepDeadline = deadline
	return []EventEnvelope{pub(NightStepChanged{
		Step:     step,
		Day:      e.state.Day,
		Deadline: deadline,
	})}
}

// nextNightStep returns the successor in the INTRO -> MAFIA -> POLICE ->
// DOCTOR -> RESOLVED chain. RESOLVED is terminal.
func nextNightStep(s NightStep) NightStep {
	switch s {
	case NightStepIntro:
		return NightStepMafia
	case NightStepMafia:
		return NightStepPolice
	case NightStepPolice:
		return NightStepDoctor
	case NightStepDoctor:
		return NightStepResolved
	default:
		return NightStepResolved
	}
}

// resolveNight applies the accumulated night actions, transitions to DAY,
// and emits the appropriate events. See business-logic-model.md §4.
//
// Order:
//  1. Determine victim (mafia kill, possibly nullified by doctor protect).
//  2. Reset night accumulators.
//  3. Increment Day, set Phase = DAY, set Deadline = now +
//     defaultDayIntroSeconds + DiscussionSeconds, emit PhaseChanged{DAY}
//     **first** so the host announces the morning before the
//     victim/peaceful-night line. Iteration 8: the leading 5s buffer
//     (defaultDayIntroSeconds) gives the death/peaceful cue room to play
//     before discussion time effectively begins. transitionIntroToDay
//     (Day 1) skips the buffer because no DeathAnnounced/PeacefulNight is
//     emitted.
//  4. Emit DeathAnnounced (and potential MafiaRepresentativeReassigned) or
//     PeacefulNight as the previous-night summary.
//  5. Check end conditions; emit GameEnded and switch to PhaseEnd if met.
func (e *engine) resolveNight() ([]EventEnvelope, error) {
	now := e.clock.Now()
	events := make([]EventEnvelope, 0, 4)

	var victim *PlayerID
	if e.state.PendingMafiaTarget != nil {
		killTarget := *e.state.PendingMafiaTarget
		protected := e.state.PendingDoctorTarget != nil && *e.state.PendingDoctorTarget == killTarget
		if !protected {
			victim = &killTarget
		}
	}

	var deathFollowUp []EventEnvelope
	if victim != nil {
		p, ok := e.state.FindPlayer(*victim)
		if !ok {
			return nil, errf(CodeUnknownPlayer, "victim %q not found", *victim)
		}
		p.Alive = false
		deathFollowUp = append(deathFollowUp, pub(DeathAnnounced{Victim: *victim}))
		if *victim == e.state.MafiaRepresentativeID {
			deathFollowUp = append(deathFollowUp, e.reassignMafiaRepresentative(*victim)...)
		}
	} else {
		deathFollowUp = append(deathFollowUp, pub(PeacefulNight{}))
	}

	e.state.PendingMafiaTarget = nil
	e.state.PendingDoctorTarget = nil
	e.state.PendingPoliceTarget = nil
	e.state.PoliceCheckedThisNight = false
	e.state.NightStep = ""
	e.state.NightStepDeadline = time.Time{}

	e.state.Day++
	e.state.Phase = PhaseDay
	e.state.Deadline = now.Add(time.Duration(
		defaultDayIntroSeconds+e.state.Settings.DiscussionSeconds,
	) * time.Second)
	e.state.LastTickAt = now
	events = append(events, pub(PhaseChanged{
		Phase:    PhaseDay,
		Day:      e.state.Day,
		Deadline: e.state.Deadline,
	}))
	events = append(events, deathFollowUp...)

	// Iteration 9 FR-1/FR-2: keep Phase=DAY so the DeathAnnounced /
	// PeacefulNight subtitle and its cue render uninterrupted; Tick will
	// fire GameEnded once the buffer deadline elapses.
	if reason, winner, ok := e.evaluateEnd(); ok {
		e.scheduleGameEnd(reason, winner)
	}
	return events, nil
}

// reassignMafiaRepresentative picks a uniformly-random living mafia and
// records the change. Returns the corresponding events (private to mafia).
// When no living mafia remain, the representative ID is cleared and no event
// is emitted (the end condition will be checked by the caller).
func (e *engine) reassignMafiaRepresentative(oldID PlayerID) []EventEnvelope {
	living := e.state.LivingMafiaIDs()
	if len(living) == 0 {
		e.state.MafiaRepresentativeID = ""
		return nil
	}
	innerRand, err := newInnerRand(e.rng)
	if err != nil {
		// Fall back deterministically; reassignment must not block.
		e.state.MafiaRepresentativeID = living[0]
	} else {
		e.state.MafiaRepresentativeID = living[innerRand.Intn(len(living))]
	}
	return []EventEnvelope{mafia(MafiaRepresentativeReassigned{
		OldID: oldID,
		NewID: e.state.MafiaRepresentativeID,
	})}
}
