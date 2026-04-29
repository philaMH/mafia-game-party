package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

func TestHostAuthority_FirstClaimSucceedsSecondRejected(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()

	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("first ClaimHost: %v", err)
	}
	if tok == "" {
		t.Errorf("empty token returned")
	}

	if _, err := mgr.ClaimHost(ctx); err == nil {
		t.Errorf("second ClaimHost should fail")
	} else if !errors.Is(err, game.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestHostAuthority_ReleaseAllowsReclaim(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	mgr.ReleaseHost(tok)
	if _, err := mgr.ClaimHost(ctx); err != nil {
		t.Errorf("Reclaim after Release: %v", err)
	}
}

func TestOpenRoom_HostNotInLobbyMembers(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	state, err := mgr.OpenRoom(ctx, tok, opts)
	if err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	if state.Phase != game.PhaseLobby {
		t.Errorf("Phase=%s, want LOBBY", state.Phase)
	}
	if len(state.Players) != 0 {
		t.Errorf("v2 LOBBY should have 0 players (host not member); got %d", len(state.Players))
	}
	if state.HostID != "" {
		t.Errorf("HostID=%q, want empty under v2 flow", state.HostID)
	}
}

func TestOpenRoom_RejectsInvalidToken(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.OpenRoom(ctx, session.HostToken("bogus"), game.DefaultOptions(6)); err == nil {
		t.Errorf("invalid token should be rejected")
	}
}

func TestHostStartGame_RequiresMinPlayersAndStarts(t *testing.T) {
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

	if _, err := mgr.HostStartGame(ctx, tok); err == nil {
		t.Errorf("HostStartGame with 0 players should fail")
	}

	for i := 0; i < 6; i++ {
		if _, err := mgr.JoinPlayer(ctx, namesPool[i]); err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
	}

	outs, err := mgr.HostStartGame(ctx, tok)
	if err != nil {
		t.Fatalf("HostStartGame after 6 joins: %v", err)
	}
	if len(outs) == 0 {
		t.Errorf("HostStartGame should emit events")
	}

	state := mgr.Snapshot()
	if state.Phase != game.PhaseIntro {
		t.Errorf("Phase=%s, want INTRO", state.Phase)
	}
	if len(state.Players) != 6 {
		t.Errorf("Players=%d, want 6 (host excluded)", len(state.Players))
	}
}

func TestHostForceTerminate_EndsGame(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(6)
	opts.MaxPlayers = 6
	if _, err := mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := mgr.JoinPlayer(ctx, namesPool[i]); err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
	}
	if _, err := mgr.HostStartGame(ctx, tok); err != nil {
		t.Fatalf("HostStartGame: %v", err)
	}

	outs, err := mgr.HostForceTerminate(ctx, tok)
	if err != nil {
		t.Fatalf("HostForceTerminate: %v", err)
	}
	gotEnded := false
	for _, o := range outs {
		if g, ok := o.Envelope.Event.(game.GameEnded); ok {
			gotEnded = true
			if g.EndReason != game.EndHostForceEnd {
				t.Errorf("EndReason=%v, want HOST_FORCE_END", g.EndReason)
			}
		}
	}
	if !gotEnded {
		t.Errorf("HostForceTerminate should emit GameEnded")
	}
}

func TestEndSelfIntro_DispatchesViaSubmitAction(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(6)
	opts.MaxPlayers = 6
	if _, err := mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	joiners := make([]session.JoinResult, 0, 6)
	for i := 0; i < 6; i++ {
		jr, err := mgr.JoinPlayer(ctx, namesPool[i])
		if err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
		joiners = append(joiners, jr)
	}
	if _, err := mgr.HostStartGame(ctx, tok); err != nil {
		t.Fatalf("HostStartGame: %v", err)
	}

	state := mgr.Snapshot()
	current := state.Players[state.IntroSpeakerIdx].ID
	outs, err := mgr.SubmitAction(ctx, game.EndSelfIntro{PlayerID: current})
	if err != nil {
		t.Fatalf("SubmitAction(EndSelfIntro): %v", err)
	}
	if len(outs) == 0 {
		t.Errorf("expected events from EndSelfIntro")
	}
	advanced := mgr.Snapshot()
	if advanced.IntroSpeakerIdx != 1 {
		t.Errorf("IntroSpeakerIdx=%d, want 1", advanced.IntroSpeakerIdx)
	}

	// Non-current speaker is rejected. Pick a player who is NOT at the
	// current intro index (idx=1 after the advance above).
	var nonCurrent game.PlayerID
	for i, p := range advanced.Players {
		if i != advanced.IntroSpeakerIdx {
			nonCurrent = p.ID
			break
		}
	}
	if _, err := mgr.SubmitAction(ctx, game.EndSelfIntro{PlayerID: nonCurrent}); err == nil {
		t.Errorf("non-current speaker EndSelfIntro should fail")
	}
	_ = joiners
}
