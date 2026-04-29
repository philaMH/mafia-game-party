package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

func TestStartGame_RequiresHost(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	_, others := makeLobby(t, mgr, 6)

	if _, err := mgr.StartGame(ctx, others[0].PlayerID, game.DefaultOptions(6)); !errors.Is(err, game.ErrPermissionDenied) {
		t.Errorf("non-host should get ErrPermissionDenied, got %v", err)
	}
}

func TestStartGame_RequiresMinPlayers(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 4)

	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(4)); !errors.Is(err, game.ErrValidation) {
		t.Errorf("expected ErrValidation under min players, got %v", err)
	}
}

func TestStartGame_HappyPath(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)

	outs, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6))
	if err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if len(outs) == 0 {
		t.Fatal("expected events")
	}
	// Look for GameStarted, PhaseChanged{INTRO}, IntroSpeakerChanged.
	var sawStart, sawPhase, sawSpeaker bool
	for _, o := range outs {
		switch o.Envelope.Event.(type) {
		case game.GameStarted:
			sawStart = true
		case game.PhaseChanged:
			sawPhase = true
		case game.IntroSpeakerChanged:
			sawSpeaker = true
		}
	}
	if !sawStart || !sawPhase || !sawSpeaker {
		t.Errorf("missing events: start=%v phase=%v speaker=%v", sawStart, sawPhase, sawSpeaker)
	}
}

func TestStartGame_RejectsDoubleStart(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); !errors.Is(err, game.ErrWrongPhase) {
		t.Errorf("second start: expected ErrWrongPhase, got %v", err)
	}
}
