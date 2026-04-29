package game

import (
	"testing"
)

func TestCheckEnd_CitizenWinsWhenAllMafiaDead(t *testing.T) {
	e, _ := newTestEngine(t, 131)
	state, _ := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	// Manually kill all mafia by setting Alive=false.
	for i, p := range state.Players {
		if p.Role == RoleMafia {
			state.Players[i].Alive = false
		}
	}
	if err := e.Restore(state); err != nil {
		t.Fatal(err)
	}
	en := e.(*engine)
	evs, ended := en.checkEnd()
	if !ended {
		t.Errorf("expected end")
	}
	if !hasGameEnded(evs) {
		t.Errorf("missing GameEnded event")
	}
	snap := e.Snapshot()
	if snap.Phase != PhaseEnd {
		t.Errorf("phase=%s, want END", snap.Phase)
	}
	if snap.Winner == nil || *snap.Winner != TeamCitizen {
		t.Errorf("Winner=%v, want CITIZEN", snap.Winner)
	}
}

func TestCheckEnd_MafiaWinsWhenEqual(t *testing.T) {
	e, _ := newTestEngine(t, 137)
	state, _ := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	// Kill all citizens except mafia + 1.
	keep := 1
	for i, p := range state.Players {
		if p.Role != RoleMafia && keep > 0 {
			keep--
			continue
		}
		if p.Role != RoleMafia {
			state.Players[i].Alive = false
		}
	}
	if err := e.Restore(state); err != nil {
		t.Fatal(err)
	}
	en := e.(*engine)
	_, ended := en.checkEnd()
	if !ended {
		t.Errorf("expected end")
	}
	snap := e.Snapshot()
	if snap.Winner == nil || *snap.Winner != TeamMafia {
		t.Errorf("Winner=%v, want MAFIA", snap.Winner)
	}
}

func hasGameEnded(evs []EventEnvelope) bool {
	for _, ev := range evs {
		if _, ok := ev.Event.(GameEnded); ok {
			return true
		}
	}
	return false
}
