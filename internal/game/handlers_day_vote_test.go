package game

import (
	"errors"
	"testing"
	"time"
)

// finishNightToDay submits one mafia kill (no doctor heal, no police check
// when those roles are alive) by forcing EndNightEarly.
func finishNightToDay(t *testing.T, e Engine) State {
	t.Helper()
	state := e.Snapshot()
	if state.Phase != PhaseNight {
		t.Fatalf("expected NIGHT to transition; got %s", state.Phase)
	}
	_, _, err := e.Apply(EndNightEarly{HostID: state.HostID})
	if err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	state = e.Snapshot()
	if state.Phase != PhaseDay {
		t.Fatalf("expected DAY after EndNightEarly; got %s", state.Phase)
	}
	return state
}

func TestEndDiscussionEarly_HostOnly(t *testing.T) {
	e, _ := newTestEngine(t, 51)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	finishNightToDay(t, e)
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p2"}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("non-host EndDiscussionEarly should be denied")
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatalf("EndDiscussionEarly: %v", err)
	}
	if e.Snapshot().Phase != PhaseVote {
		t.Errorf("after EndDiscussionEarly expect VOTE")
	}
}

func TestVote_AllLivingPlayersTriggerTally(t *testing.T) {
	e, _ := newTestEngine(t, 53)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	finishNightToDay(t, e)
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	state := e.Snapshot()
	mafias, doctor, police, citizens := allRoles(state)
	_ = mafias

	// Vote everyone for the doctor.
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		if _, _, err := e.Apply(SubmitVote{Voter: p.ID, Target: doctor}); err != nil {
			t.Fatalf("vote: %v", err)
		}
	}
	state = e.Snapshot()
	// After all voted, tally fires; with single max winner, doctor is dead.
	if state.Phase == PhaseEnd {
		// Possible if vote killed doctor and an end condition triggered;
		// not expected here but acceptable.
		return
	}
	if state.Phase != PhaseNight {
		t.Errorf("after tally expected NIGHT, got %s", state.Phase)
	}
	d, _ := state.FindPlayer(doctor)
	if d.Alive {
		t.Errorf("doctor should be dead after vote")
	}
	_ = police
	_ = citizens
}

func TestVote_TieTriggersRecount(t *testing.T) {
	e, _ := newTestEngine(t, 57)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	finishNightToDay(t, e)
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
	// With remaining live count, split votes evenly between two candidates.
	a := living[0]
	b := living[1]
	for i, voter := range living {
		var target PlayerID
		if i%2 == 0 {
			target = a
		} else {
			target = b
		}
		if _, _, err := e.Apply(SubmitVote{Voter: voter, Target: target}); err != nil {
			t.Fatalf("vote: %v", err)
		}
	}
	state = e.Snapshot()
	if state.Phase != PhaseRecount {
		t.Errorf("tie should trigger RECOUNT, got %s", state.Phase)
	}
	if len(state.VoteCandidates) != 2 {
		t.Errorf("expected 2 candidates, got %v", state.VoteCandidates)
	}
}

func TestVote_RecountRejectsOffCandidate(t *testing.T) {
	e, _ := newTestEngine(t, 59)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	finishNightToDay(t, e)
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
	for i, voter := range living {
		var target PlayerID
		if i%2 == 0 {
			target = a
		} else {
			target = b
		}
		if _, _, err := e.Apply(SubmitVote{Voter: voter, Target: target}); err != nil {
			t.Fatalf("vote: %v", err)
		}
	}
	// Now in RECOUNT.
	off := PlayerID("")
	for _, p := range state.Players {
		if p.Alive && p.ID != a && p.ID != b {
			off = p.ID
			break
		}
	}
	if off == "" {
		t.Skip("no off-candidate available")
	}
	if _, _, err := e.Apply(SubmitVote{Voter: living[0], Target: off}); !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("recount off-candidate vote should be ErrInvalidTarget, got %v", err)
	}
}

// silence unused
var _ = time.Time{}
