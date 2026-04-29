package game

import (
	"errors"
	"testing"
)

// advanceToNight runs the engine through INTRO -> DAY 1 -> VOTE -> NIGHT 1.
// Day 1 is forced to end immediately and every living player abstains so no
// elimination occurs before the first night begins.
func advanceToNight(t *testing.T, e Engine) State {
	t.Helper()
	state := e.Snapshot()
	for state.Phase == PhaseIntro {
		_, _, err := e.Apply(AdvanceIntro{HostID: state.HostID})
		if err != nil {
			t.Fatalf("AdvanceIntro: %v", err)
		}
		state = e.Snapshot()
	}
	if state.Phase != PhaseDay {
		t.Fatalf("expected DAY 1 after intro, got %s", state.Phase)
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: state.HostID}); err != nil {
		t.Fatalf("EndDiscussionEarly (Day 1): %v", err)
	}
	state = e.Snapshot()
	if state.Phase != PhaseVote {
		t.Fatalf("expected VOTE after Day 1 discussion, got %s", state.Phase)
	}
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		if _, _, err := e.Apply(SubmitVote{Voter: p.ID, Target: ""}); err != nil {
			t.Fatalf("SubmitVote abstain (Day 1): %v", err)
		}
	}
	state = e.Snapshot()
	if state.Phase != PhaseNight {
		t.Fatalf("expected NIGHT after Day 1 vote, got %s", state.Phase)
	}
	if state.NightStep != NightStepMafia {
		t.Fatalf("expected NightStep=MAFIA at NIGHT entry, got %q", state.NightStep)
	}
	return state
}

func TestMafiaKill_RepresentativeOnly(t *testing.T) {
	e, _ := newTestEngine(t, 7)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, _, _, citizens := allRoles(state)

	// Find a non-rep mafia (when 2 mafia)
	if len(mafias) < 2 {
		t.Skip("need >=2 mafia for this test")
	}
	var nonRep PlayerID
	for _, m := range mafias {
		if m != state.MafiaRepresentativeID {
			nonRep = m
			break
		}
	}
	if nonRep == "" {
		t.Skip("no non-rep mafia available")
	}
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: nonRep, Target: citizens[0]}); !errors.Is(err, ErrNotRepresentative) {
		t.Errorf("non-rep mafia kill should be denied, got %v", err)
	}
}

func TestMafiaKill_CannotTargetMafia(t *testing.T) {
	e, _ := newTestEngine(t, 5)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, _, _, _ := allRoles(state)
	if len(mafias) < 2 {
		t.Skip("need >=2 mafia")
	}
	var otherMafia PlayerID
	for _, m := range mafias {
		if m != state.MafiaRepresentativeID {
			otherMafia = m
			break
		}
	}
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: otherMafia}); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("mafia targeting mafia should be invalid, got %v", err)
	}
}

func TestDoctorHeal_SelfHealAllowedByDefault(t *testing.T) {
	e, clock := newTestEngine(t, 13)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias
	// Iteration 5: each NightStep is held for its configured duration;
	// tick past the MAFIA/POLICE deadlines before the doctor can act.
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	advanceNightStep(t, e, clock) // -> POLICE
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]}); err != nil {
		t.Fatalf("PoliceCheck: %v", err)
	}
	advanceNightStep(t, e, clock) // -> DOCTOR
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: doctor}); err != nil {
		t.Errorf("self-heal should be allowed by default, got %v", err)
	}
}

func TestDoctorHeal_SelfHealDisabled(t *testing.T) {
	e, clock := newTestEngine(t, 17)
	opts := DefaultOptions(8)
	opts.DoctorSelfHealAllowed = false
	state, _, err := e.Start("g1", "p1", playerSet(8), opts)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	state = advanceToNight(t, e)
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]}); err != nil {
		t.Fatalf("PoliceCheck: %v", err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: doctor}); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("self-heal should be denied when disabled, got %v", err)
	}
}

func TestPoliceCheck_OncePerNight(t *testing.T) {
	e, clock := newTestEngine(t, 23)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, _, police, citizens := allRoles(state)
	_ = mafias
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	advanceNightStep(t, e, clock) // MAFIA -> POLICE
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[0]}); err != nil {
		t.Fatalf("first check: %v", err)
	}
	// Iteration 5: NightStep no longer auto-advances after a check, so a
	// second submission lands during the same POLICE step. The
	// PoliceCheckedThisNight flag now trips first → ErrAlreadyDone.
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]}); !errors.Is(err, ErrAlreadyDone) {
		t.Errorf("second check should be ErrAlreadyDone, got %v", err)
	}
}

func TestPoliceCheck_NoSelfInvestigate(t *testing.T) {
	e, clock := newTestEngine(t, 29)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, _, police, citizens := allRoles(state)
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: police}); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("self-investigate should be invalid, got %v", err)
	}
}

func TestPoliceCheck_ResultIsPrivate(t *testing.T) {
	e, clock := newTestEngine(t, 31)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, _, police, citizens := allRoles(state)
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	advanceNightStep(t, e, clock)
	_, evs, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]})
	if err != nil {
		t.Fatalf("PoliceCheck: %v", err)
	}
	for _, ev := range evs {
		if pr, ok := ev.Event.(PoliceResult); ok {
			if ev.Visibility != VisPlayer {
				t.Errorf("PoliceResult visibility=%v, want VisPlayer", ev.Visibility)
			}
			if ev.PlayerID != police {
				t.Errorf("PoliceResult recipient=%s, want %s", ev.PlayerID, police)
			}
			if pr.Team != TeamCitizen {
				t.Errorf("citizen target should report TeamCitizen, got %s", pr.Team)
			}
			return
		}
	}
	t.Errorf("PoliceResult not emitted")
}

func TestEndNightEarly_HostForcesResolve(t *testing.T) {
	e, _ := newTestEngine(t, 37)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	state, _, err := e.Apply(EndNightEarly{HostID: "p1"})
	if err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	if state.Phase != PhaseDay {
		t.Errorf("after EndNightEarly expect DAY, got %s", state.Phase)
	}
}

func TestNight_AutoResolveOnAllSubmitted(t *testing.T) {
	e, clock := newTestEngine(t, 41)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias
	target := citizens[0]
	// Iteration 5: each step is held for its configured duration. After
	// the last submission a final tick past the DOCTOR deadline triggers
	// resolveNight() and the engine flips to DAY.
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: target}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[2]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[1]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock) // DOCTOR expiry -> resolveNight -> DAY
	snap := e.Snapshot()
	if snap.Phase != PhaseDay {
		t.Errorf("expected resolve-to-DAY after DOCTOR expiry, got %s", snap.Phase)
	}
	if snap.LiveCount() != 7 {
		t.Errorf("LiveCount=%d, want 7 (one death)", snap.LiveCount())
	}
}

func TestNight_DoctorProtectsTarget(t *testing.T) {
	e, clock := newTestEngine(t, 43)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state = advanceToNight(t, e)
	_, doctor, police, citizens := allRoles(state)
	target := citizens[0]
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: target}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[2]}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: target}); err != nil {
		t.Fatal(err)
	}
	advanceNightStep(t, e, clock)
	snap := e.Snapshot()
	if snap.LiveCount() != 8 {
		t.Errorf("doctor protect should keep all alive, got LiveCount=%d", snap.LiveCount())
	}
}
