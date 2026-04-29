package game

import (
	"math/rand"
	"testing"
)

func TestAssign_Distribution(t *testing.T) {
	cases := []struct {
		n          int
		mafiaCount int
	}{
		{6, 1}, {7, 2}, {8, 2}, {9, 2}, {10, 3}, {11, 3}, {12, 3},
	}
	for _, tc := range cases {
		ids := make([]PlayerID, tc.n)
		for i := 0; i < tc.n; i++ {
			ids[i] = PlayerID(testID(i))
		}
		opts := Options{
			MafiaCount:            tc.mafiaCount,
			IntroSecondsPerPlayer: 20,
			DiscussionSeconds:     180,
		}
		rng := rand.New(rand.NewSource(1))
		asg, err := NewAssigner(NewDefaultKeywordPool()).Assign(ids, opts, rng)
		if err != nil {
			t.Fatalf("n=%d: %v", tc.n, err)
		}
		mafia, doctor, police, citizens := 0, 0, 0, 0
		for _, r := range asg.PlayerRoles {
			switch r {
			case RoleMafia:
				mafia++
			case RoleDoctor:
				doctor++
			case RolePolice:
				police++
			case RoleCitizen:
				citizens++
			}
		}
		if mafia != tc.mafiaCount {
			t.Errorf("n=%d: mafia=%d, want %d", tc.n, mafia, tc.mafiaCount)
		}
		if doctor != 1 {
			t.Errorf("n=%d: doctor=%d, want 1", tc.n, doctor)
		}
		if police != 1 {
			t.Errorf("n=%d: police=%d, want 1", tc.n, police)
		}
		if citizens != tc.n-tc.mafiaCount-2 {
			t.Errorf("n=%d: citizens=%d, want %d", tc.n, citizens, tc.n-tc.mafiaCount-2)
		}
	}
}

func TestAssign_SameRoleSameKeyword(t *testing.T) {
	ids := make([]PlayerID, 12)
	for i := 0; i < 12; i++ {
		ids[i] = PlayerID(testID(i))
	}
	rng := rand.New(rand.NewSource(7))
	opts := Options{MafiaCount: 3, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180}
	asg, err := NewAssigner(NewDefaultKeywordPool()).Assign(ids, opts, rng)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	byRole := map[Role]string{}
	for pid, r := range asg.PlayerRoles {
		kw := asg.PlayerKeywords[pid]
		if existing, ok := byRole[r]; ok {
			if existing != kw {
				t.Errorf("role %s has differing keywords: %q vs %q", r, existing, kw)
			}
		} else {
			byRole[r] = kw
		}
	}
}

func TestAssign_RepresentativeIsMafia(t *testing.T) {
	ids := make([]PlayerID, 8)
	for i := 0; i < 8; i++ {
		ids[i] = PlayerID(testID(i))
	}
	rng := rand.New(rand.NewSource(1))
	opts := Options{MafiaCount: 2, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180}
	asg, err := NewAssigner(NewDefaultKeywordPool()).Assign(ids, opts, rng)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if asg.RepresentativeID == "" {
		t.Fatalf("expected representative")
	}
	if asg.PlayerRoles[asg.RepresentativeID] != RoleMafia {
		t.Errorf("representative is not mafia: %s", asg.PlayerRoles[asg.RepresentativeID])
	}
	mafiaSet := map[PlayerID]bool{}
	for _, m := range asg.MafiaIDs {
		mafiaSet[m] = true
	}
	if !mafiaSet[asg.RepresentativeID] {
		t.Errorf("representative is not in MafiaIDs")
	}
}

func TestAssign_DeterministicWithSameSeed(t *testing.T) {
	ids := make([]PlayerID, 8)
	for i := 0; i < 8; i++ {
		ids[i] = PlayerID(testID(i))
	}
	opts := Options{MafiaCount: 2, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180}
	rng1 := rand.New(rand.NewSource(123))
	rng2 := rand.New(rand.NewSource(123))
	a1, err := NewAssigner(NewDefaultKeywordPool()).Assign(ids, opts, rng1)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := NewAssigner(NewDefaultKeywordPool()).Assign(ids, opts, rng2)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range a1.PlayerRoles {
		if a2.PlayerRoles[k] != v {
			t.Errorf("non-deterministic role for %s", k)
		}
	}
	if a1.RepresentativeID != a2.RepresentativeID {
		t.Errorf("non-deterministic representative")
	}
}

func TestAssign_RejectsBadOptions(t *testing.T) {
	ids := []PlayerID{"p1", "p2", "p3", "p4", "p5", "p6"}
	rng := rand.New(rand.NewSource(0))
	opts := Options{MafiaCount: 0, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180}
	if _, err := NewAssigner(NewDefaultKeywordPool()).Assign(ids, opts, rng); err == nil {
		t.Errorf("expected error for MafiaCount=0")
	}
}
