package session_test

import (
	"context"
	"sync"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

func TestSnapshot_BeforeStartReturnsZero(t *testing.T) {
	mgr, _ := newTestManager(t)
	state := mgr.Snapshot()
	if state.Phase != "" {
		t.Errorf("expected empty Phase before StartGame, got %q", state.Phase)
	}
}

func TestSnapshot_AfterStartReflectsEngine(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	state := mgr.Snapshot()
	if state.Phase != game.PhaseIntro {
		t.Errorf("expected PhaseIntro, got %q", state.Phase)
	}
	if len(state.Players) != 6 {
		t.Errorf("expected 6 players, got %d", len(state.Players))
	}
}

func TestSnapshot_RaceFree(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = mgr.Snapshot()
		}()
		go func() {
			defer wg.Done()
			_, _ = mgr.SubmitAction(ctx, game.ToggleVoice{HostID: host.PlayerID, On: true})
		}()
	}
	wg.Wait()
}

func TestSnapshot_ReturnsClone(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	a := mgr.Snapshot()
	b := mgr.Snapshot()
	// Mutating one must not affect the other.
	if len(a.Players) > 0 {
		a.Players[0].Alive = false
		if !b.Players[0].Alive {
			t.Error("expected independent slices in returned States")
		}
	}
}
