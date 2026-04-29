package session_test

import (
	"context"
	"sync"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

// TestConcurrent_SubmitActionSerializes confirms NFR-U2-C1: even when N
// goroutines submit actions concurrently, the manager serializes them so
// the final engine state is consistent (no panic, no data race).
//
// Run with -race to validate NFR-U2-C2.
func TestConcurrent_SubmitActionSerializes(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Toggle voice from many goroutines; the engine accepts these in
			// any phase so all calls succeed regardless of interleaving.
			_, _ = mgr.SubmitAction(ctx, game.ToggleVoice{HostID: host.PlayerID, On: true})
		}()
	}
	wg.Wait()
}
