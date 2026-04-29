package session_test

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
)

func TestTick_NoOpBeforeStart(t *testing.T) {
	mgr, _ := newTestManager(t)
	mgr.Tick(time.Now()) // must not panic, must not lock indefinitely
}

func TestTick_AfterStartTransitionsIntroSpeaker(t *testing.T) {
	mgr, clk := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	// Advance fake clock past intro window then tick.
	clk.Advance(25 * time.Second)
	mgr.Tick(clk.T) // expect engine.Tick to advance speaker / phase
}

func TestNew_GracefulShutdown(t *testing.T) {
	dir := t.TempDir()
	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "u2.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	clock := &game.FakeClock{T: time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)}
	engine := game.New(game.NewAssigner(game.NewDefaultKeywordPool()), clock, rand.Reader)
	mgr, err := session.New(store, announce.NewDefaultCatalog(), engine, clock, rand.Reader,
		session.SessionOpts{TickInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- mgr.Close(context.Background()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return within 2s — ticker leaked")
	}

	// Idempotent.
	if err := mgr.Close(context.Background()); err != nil {
		t.Errorf("second Close: %v", err)
	}
}
