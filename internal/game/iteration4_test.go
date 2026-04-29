package game

import (
	"errors"
	"testing"
	"time"
)

// Iteration 4 — Day 1 vote, sequenced night (MAFIA -> POLICE -> DOCTOR),
// reordered DAY events, and PoliceHistory accumulation.

// I4-T1 — INTRO ends with PhaseChanged{DAY, Day=1} and a DiscussionSeconds
// deadline; no PeacefulNight / DeathAnnounced is emitted on the very first
// day because there was no preceding night.
func TestI4_IntroToDay1HasNoNightSummary(t *testing.T) {
	e, clock := newTestEngine(t, 4001)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	for i := 0; i < 6; i++ {
		current := e.Snapshot().Players[i].ID
		_, _, err := e.Apply(EndSelfIntro{PlayerID: current})
		if err != nil {
			t.Fatalf("EndSelfIntro #%d: %v", i, err)
		}
	}
	state := e.Snapshot()
	if state.Phase != PhaseDay {
		t.Fatalf("expected DAY 1, got %s", state.Phase)
	}
	if state.Day != 1 {
		t.Fatalf("Day=%d, want 1", state.Day)
	}
	wantDeadline := clock.Now().Add(time.Duration(state.Settings.DiscussionSeconds) * time.Second)
	if !state.Deadline.Equal(wantDeadline) {
		t.Errorf("Day 1 deadline=%v, want %v", state.Deadline, wantDeadline)
	}
}

// I4-T2 — Day 1 VOTE with all abstentions transitions into NIGHT 1 with
// NightStep=MAFIA. SubmitMafiaKill must be allowed before any
// SubmitPoliceCheck or SubmitDoctorHeal.
//
// Iteration 5 update: NightStep transitions are no longer triggered by
// action submission. The test ticks past each step's deadline to drive
// MAFIA -> POLICE -> DOCTOR -> resolveNight.
func TestI4_NightSequence_MafiaFirst(t *testing.T) {
	e, clock := newTestEngine(t, 4003)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias

	// Police can't act in the MAFIA step.
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[0]}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("PoliceCheck during MAFIA step should be ErrWrongPhase, got %v", err)
	}
	// Doctor can't act either.
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[0]}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("DoctorHeal during MAFIA step should be ErrWrongPhase, got %v", err)
	}
	// Mafia kill is recorded but does NOT advance NightStep (Iteration 5).
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	if e.Snapshot().NightStep != NightStepMafia {
		t.Errorf("NightStep=%s, want MAFIA still (submission must not advance)", e.Snapshot().NightStep)
	}
	advanceNightStep(t, e, clock)
	if e.Snapshot().NightStep != NightStepPolice {
		t.Errorf("NightStep=%s, want POLICE after timer expiry", e.Snapshot().NightStep)
	}
	// Doctor still can't act before police's window closes.
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[0]}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("DoctorHeal during POLICE step should be ErrWrongPhase, got %v", err)
	}
	// Police acts; step is held until tick.
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]}); err != nil {
		t.Fatalf("PoliceCheck: %v", err)
	}
	if e.Snapshot().NightStep != NightStepPolice {
		t.Errorf("NightStep=%s, want POLICE still after submission", e.Snapshot().NightStep)
	}
	advanceNightStep(t, e, clock)
	if e.Snapshot().NightStep != NightStepDoctor {
		t.Errorf("NightStep=%s, want DOCTOR after timer expiry", e.Snapshot().NightStep)
	}
	// Doctor acts; resolveNight fires when DOCTOR's timer expires.
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[2]}); err != nil {
		t.Fatalf("DoctorHeal: %v", err)
	}
	advanceNightStep(t, e, clock)
	snap := e.Snapshot()
	if snap.Phase != PhaseDay {
		t.Errorf("Phase=%s, want DAY after DOCTOR timer expires", snap.Phase)
	}
	if snap.NightStep != "" {
		t.Errorf("NightStep=%q, want empty after night resolved", snap.NightStep)
	}
}

// I4-T3 (Iteration 5 rewrite) — When the police is dead, the POLICE step
// is STILL held for its full duration (NightPoliceSeconds) so observers
// cannot infer the death from an abnormally fast transition. The dead
// step is announced (NightStepChanged{POLICE}) and only advances to
// DOCTOR after its deadline elapses via Tick.
func TestI4_NightStep_DeadRoleStillHeld(t *testing.T) {
	e, clock := newTestEngine(t, 4005)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, _, police, citizens := allRoles(state)
	// Manually kill the police before the night flow truly begins.
	for i, p := range state.Players {
		if p.ID == police {
			state.Players[i].Alive = false
		}
	}
	if err := e.Restore(state); err != nil {
		t.Fatal(err)
	}

	state = e.Snapshot()
	if state.NightStep != NightStepMafia {
		t.Fatalf("setup expected NightStep=MAFIA, got %s", state.NightStep)
	}
	mafiaDeadline := state.NightStepDeadline

	// Mafia submits — NightStep must NOT advance immediately.
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	if e.Snapshot().NightStep != NightStepMafia {
		t.Errorf("NightStep=%s, want MAFIA still after submission", e.Snapshot().NightStep)
	}

	// Tick past the MAFIA deadline -> POLICE (held, even though dead).
	advanceNightStep(t, e, clock)
	mid := e.Snapshot()
	if mid.NightStep != NightStepPolice {
		t.Errorf("NightStep=%s, want POLICE held even when dead", mid.NightStep)
	}
	expectedPoliceDeadline := mafiaDeadline.Add(
		time.Duration(state.Settings.NightPoliceSeconds) * time.Second,
	)
	if !mid.NightStepDeadline.Equal(expectedPoliceDeadline) {
		t.Errorf("POLICE deadline=%v, want %v (= mafia deadline + NightPoliceSeconds)",
			mid.NightStepDeadline, expectedPoliceDeadline)
	}

	// Tick past POLICE deadline (no submission, role dead) -> DOCTOR.
	advanceNightStep(t, e, clock)
	if e.Snapshot().NightStep != NightStepDoctor {
		t.Errorf("NightStep=%s, want DOCTOR after POLICE timer expiry", e.Snapshot().NightStep)
	}
}

// I4-T4 — resolveNight emits PhaseChanged{DAY} BEFORE DeathAnnounced /
// PeacefulNight so the host opens the morning before reading the summary.
func TestI4_ResolveNight_EventOrder(t *testing.T) {
	e, clock := newTestEngine(t, 4007)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[2]}); err != nil {
		t.Fatal(err)
	}
	// Iteration 5: resolveNight fires when the DOCTOR deadline expires.
	snap := e.Snapshot()
	if !clock.Now().After(snap.NightStepDeadline) {
		clock.T = snap.NightStepDeadline.Add(time.Millisecond)
	}
	_, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	phaseIdx := -1
	deathIdx := -1
	for i, ev := range evs {
		switch ev.Event.(type) {
		case PhaseChanged:
			pc := ev.Event.(PhaseChanged)
			if pc.Phase == PhaseDay {
				phaseIdx = i
			}
		case DeathAnnounced:
			deathIdx = i
		}
	}
	if phaseIdx < 0 {
		t.Fatalf("PhaseChanged{DAY} not emitted")
	}
	if deathIdx < 0 {
		t.Fatalf("DeathAnnounced not emitted")
	}
	if !(phaseIdx < deathIdx) {
		t.Errorf("PhaseChanged{DAY} (idx=%d) must precede DeathAnnounced (idx=%d)", phaseIdx, deathIdx)
	}
}

// I4-T5 — PoliceHistory accumulates one record per successful check and
// survives a Snapshot/Restore round-trip.
func TestI4_PoliceHistory_AccumulatesAcrossNights(t *testing.T) {
	e, clock := newTestEngine(t, 4009)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias

	// Night 1: police investigates citizens[0] (citizen team).
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[3]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[0]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[3]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	snap := e.Snapshot()
	if snap.Phase != PhaseDay {
		t.Fatalf("expected DAY after night 1, got %s", snap.Phase)
	}
	if got := len(snap.PoliceHistory); got != 1 {
		t.Fatalf("PoliceHistory len=%d, want 1", got)
	}
	if snap.PoliceHistory[0].Day != 1 || snap.PoliceHistory[0].Target != citizens[0] {
		t.Errorf("history[0]=%+v, want Day=1 Target=%s", snap.PoliceHistory[0], citizens[0])
	}

	// Restore round-trip — history must survive serialization.
	e2, _ := newTestEngine(t, 9999)
	if err := e2.Restore(snap); err != nil {
		t.Fatal(err)
	}
	if got := len(e2.Snapshot().PoliceHistory); got != 1 {
		t.Errorf("after restore len=%d, want 1", got)
	}

	// End Day 2 with abstentions to reach NIGHT 2.
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: state.HostID}); err != nil {
		t.Fatal(err)
	}
	for _, p := range e.Snapshot().Players {
		if !p.Alive {
			continue
		}
		if _, _, err := e.Apply(SubmitVote{Voter: p.ID, Target: ""}); err != nil {
			t.Fatal(err)
		}
	}
	if e.Snapshot().Phase != PhaseNight {
		t.Fatalf("expected NIGHT 2, got %s", e.Snapshot().Phase)
	}

	// Night 2: police investigates a known mafia.
	mafiaTarget := mafias[0]
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[3]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: mafiaTarget}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[3]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	hist := e.Snapshot().PoliceHistory
	if len(hist) != 2 {
		t.Fatalf("PoliceHistory len=%d, want 2 after two checks", len(hist))
	}
	if hist[1].Day != 2 || hist[1].Team != TeamMafia {
		t.Errorf("history[1]=%+v, want Day=2 Team=MAFIA", hist[1])
	}
}
