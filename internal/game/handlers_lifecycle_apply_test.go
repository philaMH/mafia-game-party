package game

import (
	"errors"
	"testing"
	"time"
)

// TestApplyStartGame_FromLobby exercises the Apply(StartGame) path which
// differs from Engine.Start (the typical entry).
func TestApplyStartGame_FromLobby(t *testing.T) {
	clock := &FakeClock{T: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)}
	pool := NewDefaultKeywordPool()
	rng := deterministicRNG(151)

	// Build a LOBBY engine state directly.
	players := playerSet(8)
	s := State{
		GameID:   "g1",
		Phase:    PhaseLobby,
		Players:  players,
		HostID:   "p1",
		Settings: Options{},
		Votes:    map[PlayerID]PlayerID{},
	}
	e := New(NewAssigner(pool), clock, rng).(*engine)
	if err := e.Restore(s); err != nil {
		t.Fatal(err)
	}

	state, evs, err := e.Apply(StartGame{HostID: "p1", Options: DefaultOptions(8)})
	if err != nil {
		t.Fatalf("Apply(StartGame): %v", err)
	}
	if state.Phase != PhaseIntro {
		t.Errorf("phase=%s, want INTRO", state.Phase)
	}
	if len(evs) == 0 {
		t.Errorf("expected events from StartGame")
	}
}

func TestApplyStartGame_RejectsNonLobby(t *testing.T) {
	e, _ := newTestEngine(t, 153)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	if _, _, err := e.Apply(StartGame{HostID: "p1", Options: DefaultOptions(6)}); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("StartGame after start should be ErrWrongPhase, got %v", err)
	}
}

func TestApplyStartGame_NonHostDenied(t *testing.T) {
	clock := &FakeClock{T: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)}
	pool := NewDefaultKeywordPool()
	rng := deterministicRNG(157)
	s := State{
		Phase:    PhaseLobby,
		Players:  playerSet(6),
		HostID:   "p1",
		Settings: Options{},
		Votes:    map[PlayerID]PlayerID{},
	}
	e := New(NewAssigner(pool), clock, rng).(*engine)
	if err := e.Restore(s); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.Apply(StartGame{HostID: "p2", Options: DefaultOptions(6)}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("non-host StartGame should be denied, got %v", err)
	}
}
