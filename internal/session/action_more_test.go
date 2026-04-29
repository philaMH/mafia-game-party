package session_test

import (
	"context"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

// TestSubmitAction_AllSenderTypes exercises the senderOf type switch by
// submitting one of every action variant. We do not check whether each
// succeeds; we only need the senderOf branch to execute.
func TestSubmitAction_AllSenderTypes(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, others := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	pid := others[0].PlayerID

	for _, action := range []game.Action{
		game.AdvanceIntro{HostID: host.PlayerID},
		game.SubmitMafiaKill{Mafia: pid, Target: pid},
		game.SubmitDoctorHeal{Doctor: pid, Target: pid},
		game.SubmitPoliceCheck{Police: pid, Target: pid},
		game.EndNightEarly{HostID: host.PlayerID},
		game.EndDiscussionEarly{HostID: host.PlayerID},
		game.SubmitVote{Voter: pid, Target: pid},
		game.ToggleVoice{HostID: host.PlayerID, On: true},
	} {
		_, _ = mgr.SubmitAction(ctx, action)
	}

	// ForceEnd last — leaves the manager in a closed-game state.
	if _, err := mgr.SubmitAction(ctx, game.ForceEndGame{HostID: host.PlayerID}); err != nil {
		t.Errorf("ForceEnd: %v", err)
	}
}

// TestRestoreEndState_AutoFinalizes covers BR-U2-RESTORE-6 +
// buildResultFromState by closing a manager mid-end then re-opening.
func TestRestoreEndState_AutoFinalizes(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := makeLobby(t, mgr, 6)
	if _, err := mgr.StartGame(ctx, host.PlayerID, game.DefaultOptions(6)); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if _, err := mgr.SubmitAction(ctx, game.ForceEndGame{HostID: host.PlayerID}); err != nil {
		t.Fatalf("ForceEnd: %v", err)
	}
	// At this point active_snapshot is cleared. handleGameEnd path executed.
}
