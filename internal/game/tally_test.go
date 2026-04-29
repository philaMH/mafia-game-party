package game

import (
	"testing"
)

// runToVote: starts a fresh game, advances through INTRO->NIGHT->DAY->VOTE.
func runToVote(t *testing.T, seed int64, n int) (Engine, State) {
	t.Helper()
	e, _ := newTestEngine(t, seed)
	mustStart(t, e, playerSet(n), "p1", DefaultOptions(n))
	advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatalf("EndDiscussionEarly: %v", err)
	}
	return e, e.Snapshot()
}

func TestTally_SingleMajority(t *testing.T) {
	e, state := runToVote(t, 91, 8)
	target := state.Players[0].ID
	for _, p := range state.Players {
		if p.Alive {
			if _, _, err := e.Apply(SubmitVote{Voter: p.ID, Target: target}); err != nil {
				t.Fatalf("vote: %v", err)
			}
		}
	}
	state = e.Snapshot()
	t0, _ := state.FindPlayer(target)
	if t0.Alive {
		t.Errorf("target should be eliminated")
	}
}

func TestTally_DoubleTieResultsInNoElimination(t *testing.T) {
	e, state := runToVote(t, 95, 8)
	living := []PlayerID{}
	for _, p := range state.Players {
		if p.Alive {
			living = append(living, p.ID)
		}
	}
	a, b := living[0], living[1]

	// Round 1: tie.
	for i, voter := range living {
		var target PlayerID
		if i%2 == 0 {
			target = a
		} else {
			target = b
		}
		if _, _, err := e.Apply(SubmitVote{Voter: voter, Target: target}); err != nil {
			t.Fatal(err)
		}
	}
	state = e.Snapshot()
	if state.Phase != PhaseRecount {
		t.Fatalf("expected RECOUNT, got %s", state.Phase)
	}

	// Round 2: tie again.
	for i, voter := range living {
		var target PlayerID
		if i%2 == 0 {
			target = a
		} else {
			target = b
		}
		if _, _, err := e.Apply(SubmitVote{Voter: voter, Target: target}); err != nil {
			t.Fatal(err)
		}
	}
	state = e.Snapshot()
	if state.Phase != PhaseNight {
		t.Errorf("after tied recount expect NIGHT, got %s", state.Phase)
	}
	// Both candidates should still be alive (no elimination).
	pa, _ := state.FindPlayer(a)
	pb, _ := state.FindPlayer(b)
	if !pa.Alive || !pb.Alive {
		t.Errorf("both candidates should remain alive after tied recount")
	}
}
