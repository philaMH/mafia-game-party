package session_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

func TestCreateSession_NewLobby(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()

	jr, err := mgr.CreateSession(ctx, "호스트")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if !jr.IsHost {
		t.Error("expected IsHost=true")
	}
	if len(jr.Token) != 64 {
		t.Errorf("token len want 64, got %d", len(jr.Token))
	}
	if jr.PlayerID == "" {
		t.Error("expected non-empty PlayerID")
	}
}

func TestCreateSession_RejectsEmptyName(t *testing.T) {
	mgr, _ := newTestManager(t)
	if _, err := mgr.CreateSession(context.Background(), ""); !errors.Is(err, game.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestCreateSession_RejectsSecondLobby(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateSession(ctx, "호스트"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := mgr.CreateSession(ctx, "다른호스트"); !errors.Is(err, game.ErrWrongPhase) {
		t.Errorf("expected ErrWrongPhase, got %v", err)
	}
}

func TestJoinPlayer_AddsMembers(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateSession(ctx, "호스트"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	for i := 0; i < 5; i++ {
		jr, err := mgr.JoinPlayer(ctx, namesPool[i])
		if err != nil {
			t.Fatalf("Join #%d: %v", i, err)
		}
		if jr.IsHost {
			t.Error("non-host should be IsHost=false")
		}
		if jr.Token == "" {
			t.Error("expected token")
		}
	}
}

func TestJoinPlayer_RejectsDuplicateName(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateSession(ctx, "호스트"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := mgr.JoinPlayer(ctx, "철수"); err != nil {
		t.Fatalf("first join: %v", err)
	}
	if _, err := mgr.JoinPlayer(ctx, "철수"); !errors.Is(err, game.ErrValidation) {
		t.Errorf("expected ErrValidation on dup name, got %v", err)
	}
}

func TestJoinPlayer_RejectsLobbyFull(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateSession(ctx, "호스트"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	for i := 0; i < 11; i++ {
		if _, err := mgr.JoinPlayer(ctx, namesPool[i]); err != nil {
			t.Fatalf("Join #%d: %v", i, err)
		}
	}
	if _, err := mgr.JoinPlayer(ctx, "초과인원"); !errors.Is(err, game.ErrValidation) {
		t.Errorf("expected ErrValidation on full lobby, got %v", err)
	}
}

func TestJoinPlayer_RejectsBeforeCreateSession(t *testing.T) {
	mgr, _ := newTestManager(t)
	if _, err := mgr.JoinPlayer(context.Background(), "철수"); !errors.Is(err, game.ErrWrongPhase) {
		t.Errorf("expected ErrWrongPhase, got %v", err)
	}
}

func TestJoinPlayer_AllTokensUnique(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := mgr.CreateSession(ctx, "호스트")
	tokens := map[string]bool{host.Token: true}
	for i := 0; i < 5; i++ {
		jr, err := mgr.JoinPlayer(ctx, namesPool[i])
		if err != nil {
			t.Fatalf("Join: %v", err)
		}
		if tokens[jr.Token] {
			t.Errorf("duplicate token issued: %q", jr.Token)
		}
		tokens[jr.Token] = true
	}
}

func TestResumePlayer_ValidTokenReturnsSamePID(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	host, _ := mgr.CreateSession(ctx, "호스트")

	resumed, err := mgr.ResumePlayer(ctx, host.Token)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.PlayerID != host.PlayerID {
		t.Errorf("PID mismatch: %q vs %q", resumed.PlayerID, host.PlayerID)
	}
	if !resumed.IsHost {
		t.Error("expected IsHost=true")
	}
}

func TestResumePlayer_InvalidTokenRejected(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()
	if _, err := mgr.CreateSession(ctx, "호스트"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := mgr.ResumePlayer(ctx, "wrong-token"); !errors.Is(err, game.ErrUnknownPlayer) {
		t.Errorf("expected ErrUnknownPlayer, got %v", err)
	}
}

func TestResumePlayer_EmptyTokenRejected(t *testing.T) {
	mgr, _ := newTestManager(t)
	if _, err := mgr.ResumePlayer(context.Background(), ""); !errors.Is(err, game.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// TestLobbyMembership_BroadcastsPlayerJoined verifies that CreateSession
// + N JoinPlayer calls each emit one PlayerJoined envelope to subscribers
// (LOBBY membership events plan, Stage B). With 1 host + 5 joiners we
// expect 6 envelopes carrying the joiner's PlayerID and Name.
func TestLobbyMembership_BroadcastsPlayerJoined(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()

	var (
		mu       sync.Mutex
		joined   []game.PlayerJoined
		lobbyLen int
	)
	unsub := mgr.Subscribe(func(out session.EventOut) {
		if pj, ok := out.Envelope.Event.(game.PlayerJoined); ok {
			mu.Lock()
			joined = append(joined, pj)
			lobbyLen = len(out.State.Players)
			mu.Unlock()
		}
	})
	defer unsub()

	host, err := mgr.CreateSession(ctx, "호스트")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := mgr.JoinPlayer(ctx, namesPool[i]); err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(joined) != 6 {
		t.Fatalf("PlayerJoined count = %d, want 6 (1 host + 5 joiners): %+v", len(joined), joined)
	}
	if joined[0].PlayerID != host.PlayerID || joined[0].Name != "호스트" {
		t.Errorf("first envelope mismatch: %+v (want host)", joined[0])
	}
	for i, pj := range joined[1:] {
		if pj.Name != namesPool[i] {
			t.Errorf("envelope[%d].Name = %q, want %q", i+1, pj.Name, namesPool[i])
		}
		if pj.PlayerID == "" {
			t.Errorf("envelope[%d].PlayerID empty", i+1)
		}
	}
	if lobbyLen != 6 {
		t.Errorf("final EventOut.State.Players len = %d, want 6", lobbyLen)
	}
}

// TestLobbyMembership_JoinResultLobbyMembers asserts that JoinResult
// returned to the joiner reflects all current members (host + earlier
// joiners + self), so that a freshly-joined client can render the full
// lobby roster without waiting for a snapshot.
func TestLobbyMembership_JoinResultLobbyMembers(t *testing.T) {
	mgr, _ := newTestManager(t)
	ctx := context.Background()

	host, err := mgr.CreateSession(ctx, "호스트")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if len(host.CurrentState.Players) != 1 {
		t.Errorf("host CurrentState.Players len = %d, want 1", len(host.CurrentState.Players))
	}

	jr1, err := mgr.JoinPlayer(ctx, namesPool[0])
	if err != nil {
		t.Fatalf("JoinPlayer 1: %v", err)
	}
	if len(jr1.CurrentState.Players) != 2 {
		t.Errorf("joiner1 CurrentState.Players len = %d, want 2", len(jr1.CurrentState.Players))
	}

	jr2, err := mgr.JoinPlayer(ctx, namesPool[1])
	if err != nil {
		t.Fatalf("JoinPlayer 2: %v", err)
	}
	if len(jr2.CurrentState.Players) != 3 {
		t.Errorf("joiner2 CurrentState.Players len = %d, want 3", len(jr2.CurrentState.Players))
	}
	// Spot-check that the joiner's own ID appears.
	found := false
	for _, p := range jr2.CurrentState.Players {
		if p.ID == jr2.PlayerID {
			found = true
			if p.Name != namesPool[1] || !p.Alive || p.Role != "" {
				t.Errorf("self player malformed: %+v", p)
			}
		}
	}
	if !found {
		t.Errorf("joiner2 not in own CurrentState.Players: %+v", jr2.CurrentState.Players)
	}
}
