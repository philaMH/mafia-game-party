package session_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

func TestSubmitAction_RejectsBeforeStart(t *testing.T) {
	mgr, _ := newTestManager(t)
	if _, err := mgr.SubmitAction(context.Background(), game.AdvanceIntro{HostID: "x"}); !errors.Is(err, game.ErrWrongPhase) {
		t.Errorf("expected ErrWrongPhase, got %v", err)
	}
}

func TestSubmitAction_HostAdvanceIntroProducesEvents(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	outs, err := mgr.SubmitAction(ctx, game.AdvanceIntro{HostID: host.PlayerID})
	if err != nil {
		t.Fatalf("AdvanceIntro: %v", err)
	}
	if len(outs) == 0 {
		t.Fatal("expected events")
	}
}

func TestSubmitAction_NonHostAdvanceFails(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, others := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	outs, err := mgr.SubmitAction(ctx, game.AdvanceIntro{HostID: others[0].PlayerID})
	if !errors.Is(err, game.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
	// On error, we still get an EventOut carrying the announcement.
	if len(outs) == 0 || outs[0].Announcement == nil {
		t.Errorf("expected error announcement, got %+v", outs)
	}
}

func TestSubmitAction_ForceEndProducesGameEnded(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	outs, err := mgr.SubmitAction(ctx, game.ForceEndGame{HostID: host.PlayerID})
	if err != nil {
		t.Fatalf("ForceEndGame: %v", err)
	}
	var sawEnd bool
	for _, o := range outs {
		if _, ok := o.Envelope.Event.(game.GameEnded); ok {
			sawEnd = true
		}
	}
	if !sawEnd {
		t.Error("expected GameEnded event")
	}
}

func TestSubmitAction_AfterForceEndRejected(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if _, err := mgr.SubmitAction(ctx, game.ForceEndGame{HostID: host.PlayerID}); err != nil {
		t.Fatalf("ForceEndGame: %v", err)
	}
	if _, err := mgr.SubmitAction(ctx, game.AdvanceIntro{HostID: host.PlayerID}); !errors.Is(err, game.ErrWrongPhase) {
		t.Errorf("expected ErrWrongPhase post-end, got %v", err)
	}
}

func TestSubscribe_HandlerReceivesEvents(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)

	var received atomic.Int64
	unsub := mgr.Subscribe(func(out session.EventOut) {
		received.Add(1)
		_ = out
	})

	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if received.Load() == 0 {
		t.Error("expected handler to receive at least one event")
	}

	before := received.Load()
	unsub()
	if _, err := mgr.SubmitAction(ctx, game.ForceEndGame{HostID: host.PlayerID}); err != nil {
		t.Fatalf("ForceEnd: %v", err)
	}
	if received.Load() != before {
		t.Errorf("post-unsubscribe count grew: before=%d after=%d", before, received.Load())
	}
}

func TestSubscribe_PanicInHandlerIsIsolated(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)

	mgr.Subscribe(func(out session.EventOut) {
		panic("boom")
	})

	// StartGame should still succeed despite a panicking subscriber.
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Errorf("expected StartGame to survive handler panic, got %v", err)
	}
}
