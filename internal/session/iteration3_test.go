package session_test

import (
	"context"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

func TestRoomSnapshot_BeforeOpenRoom(t *testing.T) {
	mgr, _ := newTestManager(t)
	snap := mgr.RoomSnapshot()
	if snap.RoomOpened {
		t.Errorf("RoomOpened=true, want false")
	}
	if snap.GameStarted {
		t.Errorf("GameStarted=true, want false")
	}
	if snap.HostOccupied {
		t.Errorf("HostOccupied=true, want false")
	}
	if snap.Options != (game.Options{}) {
		t.Errorf("Options=%+v, want zero", snap.Options)
	}
}

func TestRoomSnapshot_AfterClaimBeforeOpen(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.ClaimHost(ctx); err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	snap := mgr.RoomSnapshot()
	if snap.RoomOpened {
		t.Errorf("RoomOpened=true, want false (claim only, not open)")
	}
	if !snap.HostOccupied {
		t.Errorf("HostOccupied=false, want true after Claim")
	}
	if snap.GameStarted {
		t.Errorf("GameStarted=true, want false")
	}
}

func TestRoomSnapshot_AfterOpenRoom(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	if _, err := mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	snap := mgr.RoomSnapshot()
	if !snap.RoomOpened {
		t.Errorf("RoomOpened=false, want true after OpenRoom")
	}
	if !snap.HostOccupied {
		t.Errorf("HostOccupied=false, want true")
	}
	if snap.GameStarted {
		t.Errorf("GameStarted=true, want false (still LOBBY)")
	}
	if snap.Options.MafiaCount != opts.MafiaCount {
		t.Errorf("Options.MafiaCount=%d, want %d", snap.Options.MafiaCount, opts.MafiaCount)
	}
	if snap.Options.MaxPlayers != opts.MaxPlayers {
		t.Errorf("Options.MaxPlayers=%d, want %d", snap.Options.MaxPlayers, opts.MaxPlayers)
	}
}

func TestRoomSnapshot_AfterHostStartGame(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	if _, err := mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	for i := range 6 {
		if _, err := mgr.JoinPlayer(ctx, namesPool[i]); err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
	}
	if _, err := mgr.HostStartGame(ctx, tok); err != nil {
		t.Fatalf("HostStartGame: %v", err)
	}
	snap := mgr.RoomSnapshot()
	if !snap.GameStarted {
		t.Errorf("GameStarted=false, want true after HostStartGame")
	}
	if snap.State.Phase != game.PhaseIntro {
		t.Errorf("State.Phase=%s, want INTRO", snap.State.Phase)
	}
	if !snap.RoomOpened {
		t.Errorf("RoomOpened=false, want true (still open after start)")
	}
	if len(snap.State.Players) != 6 {
		t.Errorf("Players=%d, want 6", len(snap.State.Players))
	}
}

func TestRoomSnapshot_AfterReleaseHost(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	if _, err := mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	mgr.ReleaseHost(tok)
	snap := mgr.RoomSnapshot()
	if snap.HostOccupied {
		t.Errorf("HostOccupied=true, want false after Release")
	}
	if !snap.RoomOpened {
		t.Errorf("RoomOpened=false, want true (room state survives release)")
	}
}

func TestRoomSnapshot_StateIsDeepCopy(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	if _, err := mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	for i := range 6 {
		if _, err := mgr.JoinPlayer(ctx, namesPool[i]); err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
	}
	if _, err := mgr.HostStartGame(ctx, tok); err != nil {
		t.Fatalf("HostStartGame: %v", err)
	}

	snap1 := mgr.RoomSnapshot()
	if len(snap1.State.Players) == 0 {
		t.Fatalf("expected players in snapshot, got 0")
	}
	// Mutate the returned slice — must not affect engine state.
	snap1.State.Players[0].Name = "MUTATED"

	snap2 := mgr.RoomSnapshot()
	if snap2.State.Players[0].Name == "MUTATED" {
		t.Errorf("RoomSnapshot.State leaks mutation; got %q in second snapshot", snap2.State.Players[0].Name)
	}

	_ = session.RoomSnapshot{} // reference the type to ensure it is exported
}
