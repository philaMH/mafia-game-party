package game

import (
	"errors"
	"reflect"
	"testing"
)

// TestApply_RejectsAfterEnd verifies BR-COMMON-4: PhaseEnd accepts no
// actions.
func TestApply_RejectsAfterEnd(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if _, _, err := e.Apply(ForceEndGame{HostID: "p1"}); err != nil {
		t.Fatalf("ForceEndGame: %v", err)
	}
	if _, _, err := e.Apply(ForceEndGame{HostID: "p1"}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Apply after END should return ErrWrongPhase, got %v", err)
	}
	if _, _, err := e.Apply(ToggleVoice{HostID: "p1", On: true}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("any action after END should return ErrWrongPhase")
	}
}

// TestApply_UnknownActionType verifies the default branch.
func TestApply_UnknownActionType(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	type fakeAction struct{ Action }
	if _, _, err := e.Apply(fakeAction{}); err == nil {
		t.Errorf("expected error for unknown action type")
	}
}

// TestApply_ErrorLeavesStateUnchanged verifies NFR-U1-R2.
func TestApply_ErrorLeavesStateUnchanged(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	state, _ := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	before := e.Snapshot()

	// Phase=INTRO, but try a NIGHT-only action.
	_, _, err := e.Apply(SubmitMafiaKill{Mafia: "p1", Target: "p2"})
	if err == nil {
		t.Fatalf("expected error")
	}
	after := e.Snapshot()

	if !reflect.DeepEqual(before, after) {
		t.Errorf("state changed despite Apply returning error")
	}
	_ = state
}

// TestStart_RejectsInvalidPlayerCount verifies BR-OPT-1.
func TestStart_RejectsInvalidPlayerCount(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	if _, _, err := e.Start("g1", "p1", playerSet(5), DefaultOptions(5)); err == nil {
		t.Errorf("expected error for 5 players")
	}
	if _, _, err := e.Start("g1", "p1", playerSet(13), DefaultOptions(13)); err == nil {
		t.Errorf("expected error for 13 players")
	}
}

// TestStart_HostMustBePlayer verifies host is in players list.
func TestStart_HostMustBePlayer(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	if _, _, err := e.Start("g1", "phantom", playerSet(6), DefaultOptions(6)); err == nil {
		t.Errorf("expected error for host not in players")
	}
}

// TestStart_EventStream verifies the event sequence emitted on Start.
func TestStart_EventStream(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	_, evs := mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))

	var sawGameStarted, sawPhaseChanged, sawIntroSpeaker, sawMafiaCohort bool
	roleRevealCount := 0
	for _, ev := range evs {
		switch ev.Event.(type) {
		case GameStarted:
			sawGameStarted = true
		case PhaseChanged:
			sawPhaseChanged = true
		case IntroSpeakerChanged:
			sawIntroSpeaker = true
		case MafiaCohortRevealed:
			sawMafiaCohort = true
		case RoleRevealedToPlayer:
			roleRevealCount++
		}
	}
	if !sawGameStarted || !sawPhaseChanged || !sawIntroSpeaker || !sawMafiaCohort {
		t.Errorf("missing core events: gs=%v pc=%v is=%v mc=%v",
			sawGameStarted, sawPhaseChanged, sawIntroSpeaker, sawMafiaCohort)
	}
	if roleRevealCount != 6 {
		t.Errorf("RoleRevealedToPlayer count=%d, want 6", roleRevealCount)
	}
}

// TestSnapshotRestore_RoundTrip verifies NFR-U1-R5.
func TestSnapshotRestore_RoundTrip(t *testing.T) {
	e1, _ := newTestEngine(t, 99)
	mustStart(t, e1, playerSet(8), "p1", DefaultOptions(8))
	snap := e1.Snapshot()

	e2, _ := newTestEngine(t, 99)
	if err := e2.Restore(snap); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	snap2 := e2.Snapshot()
	if !reflect.DeepEqual(snap, snap2) {
		t.Errorf("Snapshot/Restore round-trip mismatch")
	}
}

// TestRestore_RejectsEmptyState verifies validation.
func TestRestore_RejectsEmptyState(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	if err := e.Restore(State{}); err == nil {
		t.Errorf("expected error for empty State")
	}
}
