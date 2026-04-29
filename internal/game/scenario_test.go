package game

import (
	"reflect"
	"testing"
	"time"
)

// TestScenario_GameStartToFirstDay covers requirements §5 scenario 1
// (game start, role/keyword reveal, intro start, transition to DAY 1
// once every speaker's intro budget elapses).
func TestScenario_GameStartToFirstDay(t *testing.T) {
	e, clock := newTestEngine(t, 201)
	state, _ := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if state.Phase != PhaseIntro {
		t.Errorf("phase=%s, want INTRO", state.Phase)
	}
	if state.Day != 1 {
		t.Errorf("Day=%d, want 1", state.Day)
	}
	clock.Advance(6 * 20 * time.Second)
	state, _, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseDay {
		t.Errorf("phase=%s, want DAY 1 after intro elapsed", state.Phase)
	}
	if state.Day != 1 {
		t.Errorf("Day=%d, want 1 (Day 1 immediately follows intro)", state.Day)
	}
}

// TestScenario_TieRecountNoElimination covers requirements §5 scenario 4
// (tie -> recount -> tie -> no elimination -> next NIGHT).
func TestScenario_TieRecountNoElimination(t *testing.T) {
	e, _ := newTestEngine(t, 203)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	state := e.Snapshot()
	living := []PlayerID{}
	for _, p := range state.Players {
		if p.Alive {
			living = append(living, p.ID)
		}
	}
	a, b := living[0], living[1]

	// Round 1 tie.
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
	if e.Snapshot().Phase != PhaseRecount {
		t.Fatalf("expected RECOUNT")
	}
	// Round 2 tie.
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
		t.Errorf("expected NIGHT after tied recount, got %s", state.Phase)
	}
}

// TestScenario_HostRestartRestoresState covers requirements §5 scenario 3
// (host PC restart restores in-progress game state).
func TestScenario_HostRestartRestoresState(t *testing.T) {
	e1, _ := newTestEngine(t, 211)
	mustStart(t, e1, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e1)
	if _, _, err := e1.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	snap := e1.Snapshot()

	// "Restart": fresh engine, restore.
	e2, _ := newTestEngine(t, 999)
	if err := e2.Restore(snap); err != nil {
		t.Fatal(err)
	}
	snap2 := e2.Snapshot()
	if !reflect.DeepEqual(snap, snap2) {
		t.Errorf("post-restore snapshot mismatch")
	}
	if snap2.Phase != PhaseDay {
		t.Errorf("phase=%s, want DAY", snap2.Phase)
	}
}
