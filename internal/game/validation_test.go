package game

import (
	"errors"
	"testing"
)

func TestValidateOptions_Cases(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		playerCount int
		wantErr     bool
		wantField   string
	}{
		{
			name:        "valid 6 players, 1 mafia",
			opts:        Options{MafiaCount: 1, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 6,
			wantErr:     false,
		},
		{
			name:        "valid 12 players, 3 mafia",
			opts:        Options{MafiaCount: 3, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 12,
			wantErr:     false,
		},
		{
			name:        "player count too low",
			opts:        Options{MafiaCount: 1, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 5,
			wantErr:     true,
			wantField:   "playerCount",
		},
		{
			name:        "player count too high",
			opts:        Options{MafiaCount: 1, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 13,
			wantErr:     true,
			wantField:   "playerCount",
		},
		{
			name:        "mafia count zero",
			opts:        Options{MafiaCount: 0, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 6,
			wantErr:     true,
			wantField:   "mafiaCount",
		},
		{
			name:        "mafia count too high (citizens not majority)",
			opts:        Options{MafiaCount: 5, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 8,
			wantErr:     true,
			wantField:   "mafiaCount",
		},
		{
			name:        "intro too short",
			opts:        Options{MafiaCount: 1, IntroSecondsPerPlayer: 4, DiscussionSeconds: 180},
			playerCount: 6,
			wantErr:     true,
			wantField:   "introSecondsPerPlayer",
		},
		{
			name:        "discussion too short",
			opts:        Options{MafiaCount: 1, IntroSecondsPerPlayer: 20, DiscussionSeconds: 29},
			playerCount: 6,
			wantErr:     true,
			wantField:   "discussionSeconds",
		},
		{
			name:        "max players unset (zero) accepted",
			opts:        Options{MafiaCount: 1, MaxPlayers: 0, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 6,
			wantErr:     false,
		},
		{
			name:        "max players below 6",
			opts:        Options{MafiaCount: 1, MaxPlayers: 5, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 6,
			wantErr:     true,
			wantField:   "maxPlayers",
		},
		{
			name:        "max players above 12",
			opts:        Options{MafiaCount: 1, MaxPlayers: 13, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 6,
			wantErr:     true,
			wantField:   "maxPlayers",
		},
		{
			name:        "actual players exceed max",
			opts:        Options{MafiaCount: 1, MaxPlayers: 6, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 8,
			wantErr:     true,
			wantField:   "maxPlayers",
		},
		{
			name:        "actual players within max",
			opts:        Options{MafiaCount: 2, MaxPlayers: 10, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
			playerCount: 8,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOptions(tt.opts, tt.playerCount)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				var ve ValidationErrors
				if !errors.As(err, &ve) {
					t.Fatalf("error is not ValidationErrors: %v", err)
				}
				found := false
				for _, fe := range ve {
					if fe.Field == tt.wantField {
						found = true
					}
				}
				if !found {
					t.Errorf("expected violation on field %q; got %v", tt.wantField, ve)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnsureHelpers(t *testing.T) {
	s := State{
		HostID: "p1",
		Players: []Player{
			{ID: "p1", Alive: true, Role: RoleMafia},
			{ID: "p2", Alive: false, Role: RoleCitizen},
		},
		Phase: PhaseDay,
	}

	if err := ensureHost(&s, "p2"); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("ensureHost should reject non-host")
	}
	if err := ensurePhase(&s, PhaseNight); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("ensurePhase should reject wrong phase")
	}
	if err := ensureRole(&s, "p1", RoleCitizen); !errors.Is(err, ErrRoleMismatch) {
		t.Errorf("ensureRole should reject mismatched role")
	}
	if err := ensureAlive(&s, "p2"); !errors.Is(err, ErrDeadPlayer) {
		t.Errorf("ensureAlive should reject dead player")
	}
	if err := ensureAlive(&s, "unknown"); !errors.Is(err, ErrUnknownPlayer) {
		t.Errorf("ensureAlive should reject unknown player")
	}
}
