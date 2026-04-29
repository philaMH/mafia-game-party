package game

import (
	"errors"
	"testing"
)

// TestHandlersErrorPaths exercises the error branches that the success-path
// scenarios skip. These are essentially negative table-driven tests.
func TestHandlersErrorPaths(t *testing.T) {
	t.Run("MafiaKill_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 301)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		// In INTRO, MafiaKill should fail.
		state := e.Snapshot()
		mafias, _, _, citizens := allRoles(state)
		if _, _, err := e.Apply(SubmitMafiaKill{Mafia: mafias[0], Target: citizens[0]}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("MafiaKill_NonMafia", func(t *testing.T) {
		e, _ := newTestEngine(t, 303)
		state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		state = advanceToNight(t, e)
		_, doctor, _, citizens := allRoles(state)
		if _, _, err := e.Apply(SubmitMafiaKill{Mafia: doctor, Target: citizens[0]}); !errors.Is(err, ErrRoleMismatch) {
			t.Errorf("got %v, want ErrRoleMismatch", err)
		}
	})

	t.Run("MafiaKill_DeadAttacker", func(t *testing.T) {
		e, _ := newTestEngine(t, 305)
		state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		state = advanceToNight(t, e)
		_, _, _, citizens := allRoles(state)
		// Manually kill the rep.
		for i, p := range state.Players {
			if p.ID == state.MafiaRepresentativeID {
				state.Players[i].Alive = false
			}
		}
		if err := e.Restore(state); err != nil {
			t.Fatal(err)
		}
		if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); !errors.Is(err, ErrDeadPlayer) {
			t.Errorf("got %v, want ErrDeadPlayer", err)
		}
	})

	t.Run("DoctorHeal_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 307)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		state := e.Snapshot()
		_, doctor, _, citizens := allRoles(state)
		if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: doctor, Target: citizens[0]}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("DoctorHeal_NonDoctor", func(t *testing.T) {
		e, clock := newTestEngine(t, 309)
		state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		state = advanceToNight(t, e)
		_, _, police, citizens := allRoles(state)
		// Drive NightStep to DOCTOR so the role check is what trips first.
		if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); err != nil {
			t.Fatal(err)
		}
		advanceNightStep(t, e, clock)
		if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[1]}); err != nil {
			t.Fatal(err)
		}
		advanceNightStep(t, e, clock)
		if _, _, err := e.Apply(SubmitDoctorHeal{Doctor: citizens[0], Target: citizens[1]}); !errors.Is(err, ErrRoleMismatch) {
			t.Errorf("got %v, want ErrRoleMismatch", err)
		}
	})

	t.Run("PoliceCheck_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 311)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		state := e.Snapshot()
		_, _, police, citizens := allRoles(state)
		if _, _, err := e.Apply(SubmitPoliceCheck{Police: police, Target: citizens[0]}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("EndNightEarly_NonHost", func(t *testing.T) {
		e, _ := newTestEngine(t, 313)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		advanceToNight(t, e)
		if _, _, err := e.Apply(EndNightEarly{HostID: "p2"}); !errors.Is(err, ErrPermissionDenied) {
			t.Errorf("got %v, want ErrPermissionDenied", err)
		}
	})

	t.Run("EndNightEarly_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 315)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		// Still INTRO.
		if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("EndDiscussionEarly_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 317)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("Vote_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 319)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		if _, _, err := e.Apply(SubmitVote{Voter: "p1", Target: "p2"}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("AdvanceIntro_WrongPhase", func(t *testing.T) {
		e, _ := newTestEngine(t, 321)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		advanceToNight(t, e)
		if _, _, err := e.Apply(AdvanceIntro{HostID: "p1"}); !errors.Is(err, ErrWrongPhase) {
			t.Errorf("got %v, want ErrWrongPhase", err)
		}
	})

	t.Run("ForceEnd_NonHost", func(t *testing.T) {
		e, _ := newTestEngine(t, 323)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		if _, _, err := e.Apply(ForceEndGame{HostID: "p2"}); !errors.Is(err, ErrPermissionDenied) {
			t.Errorf("got %v, want ErrPermissionDenied", err)
		}
	})

	t.Run("MafiaKill_DeadTarget", func(t *testing.T) {
		e, _ := newTestEngine(t, 325)
		state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		state = advanceToNight(t, e)
		_, _, _, citizens := allRoles(state)
		// Mark a citizen dead.
		for i, p := range state.Players {
			if p.ID == citizens[0] {
				state.Players[i].Alive = false
			}
		}
		if err := e.Restore(state); err != nil {
			t.Fatal(err)
		}
		if _, _, err := e.Apply(SubmitMafiaKill{Mafia: state.MafiaRepresentativeID, Target: citizens[0]}); !errors.Is(err, ErrDeadPlayer) {
			t.Errorf("got %v, want ErrDeadPlayer", err)
		}
	})
}

// TestPoolForUnknownRole exercises the default branch.
func TestPoolForUnknownRole(t *testing.T) {
	p := NewDefaultKeywordPool().(mapKeywordPool)
	if got := p.poolFor(Role("UNKNOWN")); got != nil {
		t.Errorf("poolFor unknown should be nil")
	}
}

// TestEnsureRole_UnknownPlayer covers the unknown-player branch.
func TestEnsureRole_UnknownPlayer(t *testing.T) {
	s := State{Players: []Player{{ID: "a", Alive: true, Role: RoleMafia}}}
	if err := ensureRole(&s, "ghost", RoleMafia); !errors.Is(err, ErrUnknownPlayer) {
		t.Errorf("got %v, want ErrUnknownPlayer", err)
	}
}
