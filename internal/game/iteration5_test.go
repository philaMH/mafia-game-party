package game

import (
	"errors"
	"testing"
	"time"
)

// Iteration 5 — fixed NightStep timers, host Pause/Resume, first-submit
// lock. The umbrella requirement is preventing role-death from leaking
// through abnormally fast night transitions; the timer is the only
// trigger that advances NightStep.

// I5-T1 — When the police is dead, the POLICE step is held for its full
// configured duration before advancing to DOCTOR. No mafia/police/doctor
// submission is required for the timer to fire.
func TestI5_DeadPoliceStepHeldFullDuration(t *testing.T) {
	e, clock := newTestEngine(t, 5001)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, _, police, _ := allRoles(state)

	// Kill police pre-night.
	for i, p := range state.Players {
		if p.ID == police {
			state.Players[i].Alive = false
		}
	}
	if err := e.Restore(state); err != nil {
		t.Fatal(err)
	}

	// Tick the entire MAFIA window (no mafia submission) -> POLICE.
	advanceNightStep(t, e, clock)
	mid := e.Snapshot()
	if mid.NightStep != NightStepPolice {
		t.Fatalf("NightStep=%s, want POLICE held even when dead", mid.NightStep)
	}
	policeDeadline := mid.NightStepDeadline

	// Halfway through the police window — must NOT have advanced yet.
	clock.T = policeDeadline.Add(-1 * time.Second)
	if _, _, err := e.Tick(clock.Now()); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().NightStep != NightStepPolice {
		t.Errorf("NightStep advanced before deadline (now=%v deadline=%v)",
			clock.Now(), policeDeadline)
	}

	// At the deadline -> DOCTOR.
	clock.T = policeDeadline.Add(time.Millisecond)
	if _, _, err := e.Tick(clock.Now()); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().NightStep != NightStepDoctor {
		t.Errorf("NightStep=%s, want DOCTOR after POLICE timer expired", e.Snapshot().NightStep)
	}
}

// I5-T2 — Mafia submits within the 30s window: target is recorded but
// NightStep stays MAFIA until the deadline; a second submission is
// rejected with ErrAlreadyDone (first-submit lock).
func TestI5_MafiaFirstSubmitLock(t *testing.T) {
	e, _ := newTestEngine(t, 5003)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, _, _, citizens := allRoles(state)

	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("first MafiaKill: %v", err)
	}
	snap := e.Snapshot()
	if snap.NightStep != NightStepMafia {
		t.Errorf("NightStep advanced on submission: %s", snap.NightStep)
	}
	if snap.PendingMafiaTarget == nil || *snap.PendingMafiaTarget != citizens[0] {
		t.Errorf("PendingMafiaTarget not recorded")
	}
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[1]}); !errors.Is(err, ErrAlreadyDone) {
		t.Errorf("second MafiaKill should be ErrAlreadyDone, got %v", err)
	}
	// Original target preserved.
	if *e.Snapshot().PendingMafiaTarget != citizens[0] {
		t.Errorf("PendingMafiaTarget changed after rejected resubmit")
	}
}

// I5-T3 — When the mafia rep does not submit, the night still resolves
// after MAFIA+POLICE+DOCTOR seconds elapse, and the result is a
// PeacefulNight.
func TestI5_NoMafiaSubmissionResolvesPeaceful(t *testing.T) {
	e, clock := newTestEngine(t, 5005)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	// Tick past every step without any submission.
	advanceNightStep(t, e, clock) // MAFIA -> POLICE
	advanceNightStep(t, e, clock) // POLICE -> DOCTOR
	advanceNightStep(t, e, clock) // DOCTOR -> resolveNight -> DAY
	snap := e.Snapshot()
	if snap.Phase != PhaseDay {
		t.Errorf("Phase=%s, want DAY after timer-only resolve", snap.Phase)
	}
	if snap.LiveCount() != len(state.Players) {
		t.Errorf("LiveCount=%d, want %d (no mafia kill)", snap.LiveCount(), len(state.Players))
	}
}

// I5-T4 — PauseGame freezes the NightStep deadline; ResumeGame shifts it
// forward by exactly the pause duration.
func TestI5_PauseShiftsNightDeadline(t *testing.T) {
	e, clock := newTestEngine(t, 5007)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	originalDeadline := e.Snapshot().NightStepDeadline

	if _, _, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Fatalf("PauseGame: %v", err)
	}
	if !e.Snapshot().Paused {
		t.Fatalf("Paused=false after PauseGame")
	}
	// 5 seconds of paused wall-clock time.
	clock.Advance(5 * time.Second)
	// Tick during pause is a no-op.
	if _, evs, err := e.Tick(clock.Now()); err != nil {
		t.Fatal(err)
	} else if len(evs) != 0 {
		t.Errorf("Tick during pause emitted %d events; want 0", len(evs))
	}
	if e.Snapshot().NightStep != NightStepMafia {
		t.Errorf("NightStep changed during pause: %s", e.Snapshot().NightStep)
	}

	if _, _, err := e.Apply(ResumeGame{HostID: state.HostID}); err != nil {
		t.Fatalf("ResumeGame: %v", err)
	}
	got := e.Snapshot().NightStepDeadline
	want := originalDeadline.Add(5 * time.Second)
	if !got.Equal(want) {
		t.Errorf("NightStepDeadline=%v, want %v (= original + 5s pause)", got, want)
	}
	if e.Snapshot().Paused {
		t.Errorf("Paused=true after ResumeGame")
	}
}

// I5-T5 — Pause is allowed during INTRO; ResumeGame shifts
// IntroSpeakerStartedAt forward so the per-speaker budget is preserved.
func TestI5_PauseShiftsIntroSpeaker(t *testing.T) {
	e, clock := newTestEngine(t, 5009)
	state, _ := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	originalStartedAt := e.Snapshot().IntroSpeakerStartedAt

	if _, _, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Fatalf("PauseGame: %v", err)
	}
	clock.Advance(7 * time.Second)
	if _, _, err := e.Apply(ResumeGame{HostID: state.HostID}); err != nil {
		t.Fatalf("ResumeGame: %v", err)
	}
	want := originalStartedAt.Add(7 * time.Second)
	if got := e.Snapshot().IntroSpeakerStartedAt; !got.Equal(want) {
		t.Errorf("IntroSpeakerStartedAt=%v, want %v", got, want)
	}
}

// I5-T6 — Pause shifts the DAY discussion Deadline forward.
func TestI5_PauseShiftsDayDeadline(t *testing.T) {
	e, clock := newTestEngine(t, 5011)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: state.HostID}); err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	originalDeadline := e.Snapshot().Deadline

	if _, _, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Fatalf("PauseGame: %v", err)
	}
	clock.Advance(20 * time.Second)
	if _, _, err := e.Apply(ResumeGame{HostID: state.HostID}); err != nil {
		t.Fatalf("ResumeGame: %v", err)
	}
	want := originalDeadline.Add(20 * time.Second)
	if got := e.Snapshot().Deadline; !got.Equal(want) {
		t.Errorf("Deadline=%v, want %v", got, want)
	}
}

// I5-T7 — Pause does NOT block role action submission; the mafia can
// still record a target while the timer is frozen.
func TestI5_SubmissionAllowedDuringPause(t *testing.T) {
	e, _ := newTestEngine(t, 5013)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, _, _, citizens := allRoles(state)
	if _, _, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Fatalf("PauseGame: %v", err)
	}
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Errorf("MafiaKill during pause should succeed, got %v", err)
	}
	if e.Snapshot().PendingMafiaTarget == nil {
		t.Errorf("MafiaTarget not recorded")
	}
}

// I5-T8 — Pause is rejected during VOTE/RECOUNT and idempotent within
// pause-supporting phases.
func TestI5_PauseRejectedOutsideTimedPhases(t *testing.T) {
	e, _ := newTestEngine(t, 5015)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: state.HostID}); err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: state.HostID}); err != nil {
		t.Fatalf("EndDiscussionEarly: %v", err)
	}
	if e.Snapshot().Phase != PhaseVote {
		t.Fatalf("expected VOTE, got %s", e.Snapshot().Phase)
	}
	if _, _, err := e.Apply(PauseGame{HostID: state.HostID}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("PauseGame in VOTE should be ErrWrongPhase, got %v", err)
	}
}

// I5-T9 — PauseGame is idempotent: a second pause emits no event and is
// not an error. Same for ResumeGame on an unpaused state.
func TestI5_PauseIdempotent(t *testing.T) {
	e, _ := newTestEngine(t, 5017)
	state, _ := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if _, evs, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Fatal(err)
	} else if len(evs) != 1 {
		t.Errorf("first PauseGame should emit 1 event, got %d", len(evs))
	}
	if _, evs, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Errorf("second PauseGame: %v", err)
	} else if len(evs) != 0 {
		t.Errorf("second PauseGame should be no-op, got %d events", len(evs))
	}
	if _, _, err := e.Apply(ResumeGame{HostID: state.HostID}); err != nil {
		t.Fatal(err)
	}
	if _, evs, err := e.Apply(ResumeGame{HostID: state.HostID}); err != nil {
		t.Errorf("second ResumeGame: %v", err)
	} else if len(evs) != 0 {
		t.Errorf("second ResumeGame should be no-op, got %d events", len(evs))
	}
}

// I5-T10 — Pause is host-only.
func TestI5_PauseRequiresHost(t *testing.T) {
	e, _ := newTestEngine(t, 5019)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if _, _, err := e.Apply(PauseGame{HostID: "p2"}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("non-host PauseGame should be denied, got %v", err)
	}
}

// I5-T11 — NightStepChanged events carry the freshly computed deadline.
func TestI5_NightStepChangedCarriesDeadline(t *testing.T) {
	e, clock := newTestEngine(t, 5021)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)

	// Move past MAFIA deadline -> POLICE; capture the emitted event.
	mafiaDeadline := state.NightStepDeadline
	clock.T = mafiaDeadline.Add(time.Millisecond)
	_, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if s, ok := ev.Event.(NightStepChanged); ok && s.Step == NightStepPolice {
			expected := mafiaDeadline.Add(time.Duration(state.Settings.NightPoliceSeconds) * time.Second)
			if !s.Deadline.Equal(expected) {
				t.Errorf("NightStepChanged{POLICE}.Deadline=%v, want %v", s.Deadline, expected)
			}
			return
		}
	}
	t.Errorf("NightStepChanged{POLICE} not emitted")
}

// I5-T12 — Options carry the new night-second fields and DefaultOptions
// fills them with the documented constants.
func TestI5_DefaultOptionsHasNightSeconds(t *testing.T) {
	opts := DefaultOptions(8)
	if opts.NightMafiaSeconds != defaultNightMafiaSeconds {
		t.Errorf("NightMafiaSeconds=%d, want %d", opts.NightMafiaSeconds, defaultNightMafiaSeconds)
	}
	if opts.NightPoliceSeconds != defaultNightPoliceSeconds {
		t.Errorf("NightPoliceSeconds=%d, want %d", opts.NightPoliceSeconds, defaultNightPoliceSeconds)
	}
	if opts.NightDoctorSeconds != defaultNightDoctorSeconds {
		t.Errorf("NightDoctorSeconds=%d, want %d", opts.NightDoctorSeconds, defaultNightDoctorSeconds)
	}
}

// I5-T13 — Custom Options values flow through to enterNight's deadline
// computation.
func TestI5_CustomNightSecondsRespected(t *testing.T) {
	e, _ := newTestEngine(t, 5023)
	opts := DefaultOptions(8)
	opts.NightMafiaSeconds = 45
	opts.NightPoliceSeconds = 7
	opts.NightDoctorSeconds = 8
	state, _, err := e.Start("g1", "p1", playerSet(8), opts)
	if err != nil {
		t.Fatal(err)
	}
	state = advanceToNight(t, e)
	enterAt := state.LastTickAt
	if got := state.NightStepDeadline.Sub(enterAt); got != 45*time.Second {
		t.Errorf("MAFIA deadline gap=%v, want 45s", got)
	}
}
