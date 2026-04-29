package game

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestStateClone_DeepCopyDecoupled(t *testing.T) {
	pid := PlayerID("p1")
	winner := TeamCitizen
	reason := EndCitizenWin
	original := State{
		GameID:                "g1",
		Phase:                 PhaseDay,
		Day:                   2,
		Players:               []Player{{ID: "p1", Name: "Alice", Alive: true, Role: RoleMafia, Keyword: "그림자"}},
		HostID:                "p1",
		Settings:              Options{MafiaCount: 1, IntroSecondsPerPlayer: 20, DiscussionSeconds: 180},
		Votes:                 map[PlayerID]PlayerID{"p1": "p2"},
		VoteCandidates:        []PlayerID{"p1", "p2"},
		PendingMafiaTarget:    &pid,
		MafiaRepresentativeID: "p1",
		Winner:                &winner,
		EndReason:             &reason,
	}

	clone := original.Clone()

	// Modify clone in every place that should be detached.
	clone.Players[0].Alive = false
	clone.Votes["p1"] = "p3"
	clone.VoteCandidates[0] = "pX"
	*clone.PendingMafiaTarget = "pX"
	*clone.Winner = TeamMafia
	*clone.EndReason = EndMafiaWin

	if original.Players[0].Alive != true {
		t.Errorf("Players slice shared")
	}
	if original.Votes["p1"] != "p2" {
		t.Errorf("Votes map shared")
	}
	if original.VoteCandidates[0] != "p1" {
		t.Errorf("VoteCandidates slice shared")
	}
	if *original.PendingMafiaTarget != "p1" {
		t.Errorf("PendingMafiaTarget pointer shared")
	}
	if *original.Winner != TeamCitizen {
		t.Errorf("Winner pointer shared")
	}
	if *original.EndReason != EndCitizenWin {
		t.Errorf("EndReason pointer shared")
	}
}

func TestStateClone_NilSafe(t *testing.T) {
	var s State
	clone := s.Clone()
	if !reflect.DeepEqual(s, clone) {
		t.Errorf("zero-value clone mismatch")
	}
}

func TestState_JSONDeterminism(t *testing.T) {
	e, _ := newTestEngine(t, 42)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))

	b1, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal 1: %v", err)
	}
	b2, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal 2: %v", err)
	}
	if string(b1) != string(b2) {
		t.Errorf("non-deterministic JSON output:\n%s\nvs\n%s", b1, b2)
	}
}

func TestState_JSONRoundTrip(t *testing.T) {
	e, _ := newTestEngine(t, 42)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))

	b, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got State
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// time fields may have monotonic clock stripped; compare wall fields
	if got.Phase != state.Phase || got.Day != state.Day || len(got.Players) != len(state.Players) {
		t.Errorf("round-trip mismatch")
	}
}

func TestState_JSONSizeUnder32KB_TwelvePlayers(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	state, _ := mustStart(t, e, playerSet(12), "p1", DefaultOptions(12))
	b, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(b) >= 32*1024 {
		t.Errorf("snapshot size %d exceeds 32 KB", len(b))
	}
}

func TestTeamOf(t *testing.T) {
	cases := map[Role]Team{
		RoleMafia:   TeamMafia,
		RoleCitizen: TeamCitizen,
		RoleDoctor:  TeamCitizen,
		RolePolice:  TeamCitizen,
	}
	for r, want := range cases {
		if got := TeamOf(r); got != want {
			t.Errorf("TeamOf(%s) = %s, want %s", r, got, want)
		}
	}
}

func TestRecommendedMafiaCount(t *testing.T) {
	cases := map[int]int{6: 1, 7: 2, 8: 2, 9: 2, 10: 3, 11: 3, 12: 3}
	for n, want := range cases {
		if got := recommendedMafiaCount(n); got != want {
			t.Errorf("recommendedMafiaCount(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestStateLiveCounts(t *testing.T) {
	s := State{Players: []Player{
		{ID: "a", Alive: true, Role: RoleMafia},
		{ID: "b", Alive: false, Role: RoleMafia},
		{ID: "c", Alive: true, Role: RoleCitizen},
		{ID: "d", Alive: true, Role: RoleDoctor},
		{ID: "e", Alive: true, Role: RolePolice},
		{ID: "f", Alive: false, Role: RolePolice},
	}}
	if s.LiveCount() != 4 {
		t.Errorf("LiveCount=%d, want 4", s.LiveCount())
	}
	if s.LiveMafiaCount() != 1 {
		t.Errorf("LiveMafiaCount=%d, want 1", s.LiveMafiaCount())
	}
	if s.LiveCitizenSideCount() != 3 {
		t.Errorf("LiveCitizenSideCount=%d, want 3", s.LiveCitizenSideCount())
	}
	if !s.HasLivingDoctor() || !s.HasLivingPolice() {
		t.Errorf("HasLivingDoctor/Police should be true")
	}
}

// silence unused-time import for environments where the file evolves.
var _ = time.Time{}
