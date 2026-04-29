package game

import (
	"testing"
)

func TestResolveNight_PeacefulNight(t *testing.T) {
	e, _ := newTestEngine(t, 71)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	state := e.Snapshot()
	hostID := state.HostID
	_, evs, err := e.Apply(EndNightEarly{HostID: hostID})
	if err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	hasPeaceful := false
	for _, ev := range evs {
		if _, ok := ev.Event.(PeacefulNight); ok {
			hasPeaceful = true
		}
	}
	if !hasPeaceful {
		t.Errorf("expected PeacefulNight when no kill submitted")
	}
}

func TestResolveNight_DayIncrement(t *testing.T) {
	e, _ := newTestEngine(t, 73)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	if state.Day != 1 {
		t.Fatalf("initial Day=%d, want 1", state.Day)
	}
	advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	state = e.Snapshot()
	if state.Day != 2 {
		t.Errorf("after first NIGHT->DAY Day=%d, want 2", state.Day)
	}
}

func TestResolveNight_DiscussionDeadlineSet(t *testing.T) {
	e, clock := newTestEngine(t, 79)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	state := e.Snapshot()
	wantDeadline := clock.Now().Add(180 * 1e9)
	if !state.Deadline.Equal(wantDeadline) {
		t.Errorf("deadline=%v, want %v", state.Deadline, wantDeadline)
	}
}
