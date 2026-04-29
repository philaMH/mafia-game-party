package game

import (
	"testing"
)

// TestReassign_OnRepresentativeKilledByMafiaTarget verifies BR-REP-4 — when
// the mafia rep is killed (e.g., voted out by the citizens), reassignment
// fires and emits MafiaRepresentativeReassigned.
func TestReassign_OnRepresentativeVotedOut(t *testing.T) {
	// Use a seed that yields >=2 mafia.
	e, _ := newTestEngine(t, 161)
	mustStart(t, e, playerSet(10), "p1", DefaultOptions(10))
	advanceToNight(t, e)
	state := e.Snapshot()
	mafias, _, _, _ := allRoles(state)
	if len(mafias) < 2 {
		t.Skip("need 2+ mafia")
	}
	rep := state.MafiaRepresentativeID

	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}

	state = e.Snapshot()
	living := []PlayerID{}
	for _, p := range state.Players {
		if p.Alive {
			living = append(living, p.ID)
		}
	}
	// All living vote for the representative.
	var sawReassigned bool
	for _, voter := range living {
		_, evs, err := e.Apply(SubmitVote{Voter: voter, Target: rep})
		if err != nil {
			t.Fatal(err)
		}
		for _, ev := range evs {
			if _, ok := ev.Event.(MafiaRepresentativeReassigned); ok {
				sawReassigned = true
			}
		}
	}
	state = e.Snapshot()
	if state.MafiaRepresentativeID == rep && state.LiveMafiaCount() > 0 {
		t.Errorf("representative should have been reassigned")
	}
	if !sawReassigned && state.LiveMafiaCount() > 0 {
		t.Errorf("MafiaRepresentativeReassigned event missing")
	}
}

// TestReassign_RepKilledAtNight covers reassignment when victim ==
// representative during night kill (rare edge case where rep ID == victim).
// Achievable by setting representative to a citizen-target if the engine had
// a bug; here we test the helper directly.
func TestReassign_HelperEmitsEventOnRepKill(t *testing.T) {
	// Build an engine where rep is alive and we mark them dead manually.
	e, _ := newTestEngine(t, 167)
	state, _ := mustStart(t, e, playerSet(10), "p1", DefaultOptions(10))
	rep := state.MafiaRepresentativeID
	_ = rep
	// Kill rep manually
	for i, p := range state.Players {
		if p.ID == rep {
			state.Players[i].Alive = false
		}
	}
	if err := e.Restore(state); err != nil {
		t.Fatal(err)
	}
	en := e.(*engine)
	evs := en.reassignMafiaRepresentative(rep)
	if en.state.MafiaRepresentativeID == rep && en.state.LiveMafiaCount() > 0 {
		t.Errorf("rep not reassigned")
	}
	if len(evs) == 0 && en.state.LiveMafiaCount() > 0 {
		t.Errorf("expected MafiaRepresentativeReassigned event")
	}
}
