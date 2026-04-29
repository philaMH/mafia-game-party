package game

import (
	"testing"
	"time"
)

func TestTick_IdempotentSameTime(t *testing.T) {
	e, clock := newTestEngine(t, 101)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	now := clock.Now().Add(1 * time.Second)
	state1, _, err := e.Tick(now)
	if err != nil {
		t.Fatal(err)
	}
	state2, evs2, err := e.Tick(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs2) != 0 {
		t.Errorf("second Tick at same time should yield no events; got %d", len(evs2))
	}
	if state1.Phase != state2.Phase {
		t.Errorf("Tick mutated phase on idempotent call")
	}
}

func TestTick_AdvancesIntroSpeakers(t *testing.T) {
	e, clock := newTestEngine(t, 103)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	clock.Advance(20 * time.Second)
	state, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseIntro {
		t.Errorf("phase=%s, want INTRO", state.Phase)
	}
	if state.IntroSpeakerIdx != 1 {
		t.Errorf("speakerIdx=%d, want 1", state.IntroSpeakerIdx)
	}
	hasIntroEvent := false
	for _, ev := range evs {
		if _, ok := ev.Event.(IntroSpeakerChanged); ok {
			hasIntroEvent = true
		}
	}
	if !hasIntroEvent {
		t.Errorf("expected IntroSpeakerChanged event")
	}
}

func TestTick_TransitionsIntroToDayWhenAllDone(t *testing.T) {
	e, clock := newTestEngine(t, 107)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	clock.Advance(6 * 20 * time.Second)
	state, _, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseDay {
		t.Errorf("phase=%s, want DAY 1 after all intros elapsed", state.Phase)
	}
	if state.Day != 1 {
		t.Errorf("Day=%d, want 1", state.Day)
	}
}

func TestTick_DayDiscussionDeadlineTransitions(t *testing.T) {
	e, clock := newTestEngine(t, 109)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().Phase != PhaseDay {
		t.Fatalf("expected DAY after EndNightEarly")
	}
	clock.Advance(180 * time.Second)
	state, _, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseVote {
		t.Errorf("phase=%s, want VOTE after deadline", state.Phase)
	}
}

func TestTick_DiscussionTimerThresholds(t *testing.T) {
	e, clock := newTestEngine(t, 113)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	advanceToNight(t, e)
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatal(err)
	}
	// 30s remaining.
	clock.Advance(150 * time.Second)
	_, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	hasTick := false
	for _, ev := range evs {
		if d, ok := ev.Event.(DiscussionTimerTick); ok && d.SecondsLeft == 30 {
			hasTick = true
		}
	}
	if !hasTick {
		t.Errorf("expected DiscussionTimerTick(30) at 150s elapsed")
	}
}
