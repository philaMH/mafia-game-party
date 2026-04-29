package game

import (
	"errors"
	"testing"
	"time"
)

func TestAdvanceIntro_HostOnly(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if _, _, err := e.Apply(AdvanceIntro{HostID: "p2"}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("non-host AdvanceIntro should be denied, got %v", err)
	}
}

func TestAdvanceIntro_ProgressesSpeakers(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	for i := 0; i < 5; i++ {
		_, _, err := e.Apply(AdvanceIntro{HostID: "p1"})
		if err != nil {
			t.Fatalf("AdvanceIntro %d: %v", i, err)
		}
	}
	state := e.Snapshot()
	if state.Phase != PhaseIntro {
		t.Errorf("after 5 advances expected still INTRO; got %s", state.Phase)
	}
	if state.IntroSpeakerIdx != 5 {
		t.Errorf("IntroSpeakerIdx=%d, want 5", state.IntroSpeakerIdx)
	}
	if _, _, err := e.Apply(AdvanceIntro{HostID: "p1"}); err != nil {
		t.Fatalf("final AdvanceIntro: %v", err)
	}
	if e.Snapshot().Phase != PhaseDay {
		t.Errorf("final AdvanceIntro should transition to DAY 1")
	}
}

func TestForceEndGame_TerminalState(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	state, evs, err := e.Apply(ForceEndGame{HostID: "p1"})
	if err != nil {
		t.Fatalf("ForceEndGame: %v", err)
	}
	if state.Phase != PhaseEnd {
		t.Errorf("phase=%s, want END", state.Phase)
	}
	if state.EndReason == nil || *state.EndReason != EndHostForceEnd {
		t.Errorf("EndReason=%v, want HOST_FORCE_END", state.EndReason)
	}
	hasGameEnded := false
	for _, ev := range evs {
		if _, ok := ev.Event.(GameEnded); ok {
			hasGameEnded = true
		}
	}
	if !hasGameEnded {
		t.Errorf("GameEnded event not emitted")
	}
}

func TestToggleVoice_HostOnly(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if _, _, err := e.Apply(ToggleVoice{HostID: "p2", On: false}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("non-host ToggleVoice should be denied")
	}
	if _, _, err := e.Apply(ToggleVoice{HostID: "p1", On: false}); err != nil {
		t.Fatalf("host ToggleVoice: %v", err)
	}
	if e.Snapshot().Settings.AnnouncementVoiceOn {
		t.Errorf("voice should be off")
	}
}

// silence unused-time import
var _ = time.Time{}

func TestEndSelfIntro_AdvancesToNextSpeaker(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	current := e.Snapshot().Players[0].ID
	state, evs, err := e.Apply(EndSelfIntro{PlayerID: current})
	if err != nil {
		t.Fatalf("EndSelfIntro: %v", err)
	}
	if state.Phase != PhaseIntro {
		t.Errorf("phase=%s, want INTRO", state.Phase)
	}
	if state.IntroSpeakerIdx != 1 {
		t.Errorf("IntroSpeakerIdx=%d, want 1", state.IntroSpeakerIdx)
	}
	gotSpeakerEvent := false
	for _, ev := range evs {
		if c, ok := ev.Event.(IntroSpeakerChanged); ok {
			gotSpeakerEvent = true
			if c.PlayerID != state.Players[1].ID {
				t.Errorf("event speaker=%s, want %s", c.PlayerID, state.Players[1].ID)
			}
		}
	}
	if !gotSpeakerEvent {
		t.Errorf("expected IntroSpeakerChanged event")
	}
}

func TestEndSelfIntro_LastSpeakerTransitionsToDay(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	for i := 0; i < 5; i++ {
		current := e.Snapshot().Players[i].ID
		if _, _, err := e.Apply(EndSelfIntro{PlayerID: current}); err != nil {
			t.Fatalf("EndSelfIntro #%d: %v", i, err)
		}
	}
	last := e.Snapshot().Players[5].ID
	state, evs, err := e.Apply(EndSelfIntro{PlayerID: last})
	if err != nil {
		t.Fatalf("final EndSelfIntro: %v", err)
	}
	if state.Phase != PhaseDay {
		t.Errorf("phase=%s, want DAY (Day 1 discussion)", state.Phase)
	}
	if state.Day != 1 {
		t.Errorf("Day=%d, want 1", state.Day)
	}
	gotPhaseChanged := false
	for _, ev := range evs {
		if pc, ok := ev.Event.(PhaseChanged); ok && pc.Phase == PhaseDay && pc.Day == 1 {
			gotPhaseChanged = true
		}
	}
	if !gotPhaseChanged {
		t.Errorf("expected PhaseChanged{DAY, Day=1} event")
	}
}

func TestEndSelfIntro_RejectsNonCurrentSpeaker(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	other := e.Snapshot().Players[2].ID
	if _, _, err := e.Apply(EndSelfIntro{PlayerID: other}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("non-current speaker EndSelfIntro should be denied, got %v", err)
	}
}

func TestEndSelfIntro_RejectsInNonIntroPhase(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	for i := 0; i < 6; i++ {
		current := e.Snapshot().Players[i].ID
		if _, _, err := e.Apply(EndSelfIntro{PlayerID: current}); err != nil {
			t.Fatalf("setup EndSelfIntro #%d: %v", i, err)
		}
	}
	if e.Snapshot().Phase != PhaseDay {
		t.Fatalf("setup expected DAY 1, got %s", e.Snapshot().Phase)
	}
	speaker := e.Snapshot().Players[0].ID
	if _, _, err := e.Apply(EndSelfIntro{PlayerID: speaker}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("EndSelfIntro in DAY should fail with ErrWrongPhase, got %v", err)
	}
}
