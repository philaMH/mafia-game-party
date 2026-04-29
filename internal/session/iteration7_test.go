package session_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

// validOpts is a shape-valid Options value used as the baseline for
// Iteration 7 SaveHostOptions tests. Mirrors the front-end's
// defaultOptions(8) shape.
func validOpts() game.Options {
	return game.Options{
		MafiaCount:            2,
		MaxPlayers:            8,
		IntroSecondsPerPlayer: 20,
		DiscussionSeconds:     180,
		NightMafiaSeconds:     30,
		NightPoliceSeconds:    10,
		NightDoctorSeconds:    10,
		DoctorSelfHealAllowed: true,
		AnnouncementVoiceOn:   true,
	}
}

func TestSaveHostOptions_NoHostToken(t *testing.T) {
	mgr, _ := newTestManager(t)
	err := mgr.SaveHostOptions(context.Background(), session.HostToken(""), validOpts())
	if err == nil {
		t.Fatalf("expected error for empty token, got nil")
	}
	var ee *game.EngineError
	if !errors.As(err, &ee) || ee.Code != game.CodePermissionDenied {
		t.Fatalf("want CodePermissionDenied, got %v", err)
	}
	if _, ok := mgr.SavedHostOptions(); ok {
		t.Errorf("hasSavedHostOptions=true after rejected save")
	}
}

func TestSaveHostOptions_BadToken(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.ClaimHost(ctx); err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	err := mgr.SaveHostOptions(ctx, session.HostToken("bogus-token"), validOpts())
	if err == nil {
		t.Fatalf("expected error for wrong token, got nil")
	}
	var ee *game.EngineError
	if !errors.As(err, &ee) || ee.Code != game.CodePermissionDenied {
		t.Fatalf("want CodePermissionDenied, got %v", err)
	}
}

func TestSaveHostOptions_ValidationFailure(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	bad := validOpts()
	bad.MaxPlayers = 5 // out of [6,12]
	bad.NightMafiaSeconds = 1
	if err := mgr.SaveHostOptions(ctx, tok, bad); err == nil {
		t.Fatalf("expected ValidationErrors, got nil")
	} else if _, ok := err.(game.ValidationErrors); !ok {
		t.Fatalf("want game.ValidationErrors, got %T (%v)", err, err)
	}
	if _, ok := mgr.SavedHostOptions(); ok {
		t.Errorf("invalid save should not flip hasSavedHostOptions")
	}
}

func TestSaveHostOptions_PersistsAcrossSessionReset(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	saved := validOpts()
	saved.IntroSecondsPerPlayer = 25
	if err := mgr.SaveHostOptions(ctx, tok, saved); err != nil {
		t.Fatalf("SaveHostOptions: %v", err)
	}

	other := validOpts()
	other.IntroSecondsPerPlayer = 99
	if _, err := mgr.OpenRoom(ctx, tok, other); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	if err := mgr.HostCloseRoom(ctx, tok); err != nil {
		t.Fatalf("HostCloseRoom: %v", err)
	}

	got, ok := mgr.SavedHostOptions()
	if !ok {
		t.Fatalf("hasSavedHostOptions=false after HostCloseRoom; expected the saved value to survive")
	}
	if got.IntroSecondsPerPlayer != 25 {
		t.Errorf("IntroSecondsPerPlayer=%d, want 25 (the saved value, not the OpenRoom payload)", got.IntroSecondsPerPlayer)
	}
}

func TestSaveHostOptions_OverwriteLatest(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	first := validOpts()
	first.DiscussionSeconds = 60
	if err := mgr.SaveHostOptions(ctx, tok, first); err != nil {
		t.Fatalf("first SaveHostOptions: %v", err)
	}
	second := validOpts()
	second.DiscussionSeconds = 240
	if err := mgr.SaveHostOptions(ctx, tok, second); err != nil {
		t.Fatalf("second SaveHostOptions: %v", err)
	}
	got, _ := mgr.SavedHostOptions()
	if got.DiscussionSeconds != 240 {
		t.Errorf("DiscussionSeconds=%d, want 240 (latest write wins)", got.DiscussionSeconds)
	}
}

func TestSaveHostOptions_ConcurrentSafe(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	tok, err := mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			opts := validOpts()
			opts.DiscussionSeconds = 30 + i*10
			if err := mgr.SaveHostOptions(ctx, tok, opts); err != nil {
				t.Errorf("goroutine %d: SaveHostOptions: %v", i, err)
			}
		}()
	}
	wg.Wait()
	got, ok := mgr.SavedHostOptions()
	if !ok {
		t.Fatalf("hasSavedHostOptions=false after concurrent saves")
	}
	// We don't assert a specific winner — just that the read is a value
	// that was actually written by some goroutine (no torn struct).
	min, max := 30, 30+(N-1)*10
	if got.DiscussionSeconds < min || got.DiscussionSeconds > max {
		t.Errorf("DiscussionSeconds=%d outside [%d,%d]; suggests torn write", got.DiscussionSeconds, min, max)
	}
}
