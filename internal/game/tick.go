package game

import "time"

// Tick implements Engine. It advances time-driven phases and is idempotent
// for a given now value (NFR-U1-R3).
//
// INTRO: rotates speakers when their per-speaker time has elapsed; once the
// last speaker finishes, transitions to NIGHT.
//
// DAY: emits DiscussionTimerTick at threshold seconds remaining (typically
// 30, 10, 0) and transitions to VOTE when the discussion deadline has
// passed.
//
// NIGHT: drives MAFIA -> POLICE -> DOCTOR -> resolveNight transitions
// purely on wall-clock deadlines (Iteration 5 R2). Multiple steps may
// expire in a single Tick when a long elapsed duration is supplied.
//
// LOBBY / VOTE / RECOUNT / END: no-op (no time-driven progression).
//
// Iteration 5: when state.Paused is true Tick is a complete no-op — it
// does not advance LastTickAt nor evaluate any deadlines. Resume is the
// sole mechanism for shifting deadlines forward by the elapsed pause
// duration.
func (e *engine) Tick(now time.Time) (State, []EventEnvelope, error) {
	if e.state.Paused {
		return e.state.Clone(), nil, nil
	}
	if !now.After(e.state.LastTickAt) {
		return e.state.Clone(), nil, nil
	}
	// Iteration 9: pending end fires before any phase progression so a
	// vote/night-driven win lands GameEnded exactly defaultFinalResult-
	// BufferSeconds after the result subtitle (regardless of which Phase
	// the engine is currently parked in: VOTE/RECOUNT for vote-end, DAY
	// for night-end). firePendingEnd updates LastTickAt itself.
	if e.state.PendingGameEnd != nil && !now.Before(e.state.PendingGameEnd.Deadline) {
		return e.firePendingEnd(now)
	}
	prev := e.state.LastTickAt
	e.state.LastTickAt = now

	switch e.state.Phase {
	case PhaseIntro:
		return e.tickIntro(now)
	case PhaseDay:
		return e.tickDay(now, prev)
	case PhaseNight:
		return e.tickNight(now)
	default:
		return e.state.Clone(), nil, nil
	}
}

// tickIntro rotates the speaker if the per-speaker budget has elapsed.
// Multiple speakers may be skipped on a single tick when a long elapsed
// duration is supplied (e.g., after restoring a snapshot).
func (e *engine) tickIntro(now time.Time) (State, []EventEnvelope, error) {
	perPlayer := time.Duration(e.state.Settings.IntroSecondsPerPlayer) * time.Second
	events := []EventEnvelope{}

	for {
		if e.state.IntroSpeakerIdx >= len(e.state.Players)-1 {
			elapsed := now.Sub(e.state.IntroSpeakerStartedAt)
			if elapsed < perPlayer {
				break
			}
			_, more, err := e.transitionIntroToDay(now)
			if err != nil {
				return e.state.Clone(), nil, err
			}
			events = append(events, more...)
			break
		}
		elapsed := now.Sub(e.state.IntroSpeakerStartedAt)
		if elapsed < perPlayer {
			break
		}
		e.state.IntroSpeakerIdx++
		e.state.IntroSpeakerStartedAt = e.state.IntroSpeakerStartedAt.Add(perPlayer)
		events = append(events, pub(IntroSpeakerChanged{
			PlayerID:    e.state.Players[e.state.IntroSpeakerIdx].ID,
			SecondsLeft: e.state.Settings.IntroSecondsPerPlayer,
		}))
	}
	return e.state.Clone(), events, nil
}

// discussionTimerThresholds returns the seconds-remaining values where a
// DiscussionTimerTick should be emitted.
var discussionTimerThresholds = []int{30, 10, 0}

// tickDay handles DAY phase progression. It emits DiscussionTimerTick at
// each threshold the timer crossed since the previous tick, and transitions
// to VOTE when the deadline has passed.
func (e *engine) tickDay(now time.Time, prev time.Time) (State, []EventEnvelope, error) {
	events := []EventEnvelope{}
	deadline := e.state.Deadline

	for _, th := range discussionTimerThresholds {
		thInstant := deadline.Add(-time.Duration(th) * time.Second)
		// Emit when this tick has reached/passed the threshold but the
		// previous tick had not.
		if !now.Before(thInstant) && prev.Before(thInstant) {
			events = append(events, pub(DiscussionTimerTick{SecondsLeft: th}))
		}
	}

	if !now.Before(deadline) {
		_, more, err := e.transitionDayToVote(now)
		if err != nil {
			return e.state.Clone(), nil, err
		}
		events = append(events, more...)
	}
	return e.state.Clone(), events, nil
}

// tickNight drives the MAFIA -> POLICE -> DOCTOR sequence on wall-clock
// deadlines. The previous step's expiration time is used as the next
// step's start, so chained transitions in a single Tick remain
// deterministic regardless of how far ahead `now` jumped.
func (e *engine) tickNight(now time.Time) (State, []EventEnvelope, error) {
	events := []EventEnvelope{}

	for {
		if e.state.NightStep == "" || e.state.NightStepDeadline.IsZero() {
			break
		}
		if now.Before(e.state.NightStepDeadline) {
			break
		}

		expired := e.state.NightStepDeadline
		next := nextNightStep(e.state.NightStep)
		if next == NightStepResolved {
			e.state.NightStep = ""
			e.state.NightStepDeadline = time.Time{}
			more, err := e.resolveNight()
			if err != nil {
				return e.state.Clone(), nil, err
			}
			events = append(events, more...)
			return e.state.Clone(), events, nil
		}
		more := e.beginNightStep(next, expired)
		events = append(events, more...)
	}
	return e.state.Clone(), events, nil
}
