package ws

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

func TestIncomingEnvelope_TypeOnlyDecode(t *testing.T) {
	raw := []byte(`{"type":"join","name":"철수","extra":"ignored"}`)
	var env incomingEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if env.Type != "join" {
		t.Errorf("Type = %q", env.Type)
	}
	// re-decode into joinPayload
	var p joinPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if p.Name != "철수" {
		t.Errorf("Name = %q", p.Name)
	}
}

func TestVisibilityToString(t *testing.T) {
	cases := map[game.Visibility]string{
		game.VisPublic:      "PUBLIC",
		game.VisPlayer:      "PLAYER",
		game.VisRoleMafia:   "ROLE_MAFIA",
		game.Visibility(99): "UNKNOWN",
	}
	for v, want := range cases {
		if got := visibilityToString(v); got != want {
			t.Errorf("visibilityToString(%d) = %q, want %q", v, got, want)
		}
	}
}

func TestBuildEventPayload_AllKinds(t *testing.T) {
	winner := game.TeamCitizen
	pid := game.PlayerID("p1")

	cases := []struct {
		ev   game.Event
		kind string
	}{
		{game.PlayerJoined{PlayerID: "p1", Name: "민수"}, "PlayerJoined"},
		{game.GameStarted{}, "GameStarted"},
		{game.PhaseChanged{Phase: game.PhaseDay, Day: 2}, "PhaseChanged"},
		{game.RoleRevealedToPlayer{PlayerID: "p1", Role: game.RoleMafia}, "RoleRevealedToPlayer"},
		{game.MafiaCohortRevealed{MafiaIDs: []game.PlayerID{"p1"}}, "MafiaCohortRevealed"},
		{game.IntroSpeakerChanged{PlayerID: "p1", SecondsLeft: 30}, "IntroSpeakerChanged"},
		{game.MafiaTargetSelected{Target: "p2"}, "MafiaTargetSelected"},
		{game.PoliceResult{Police: "p1", Target: "p2", Team: game.TeamMafia}, "PoliceResult"},
		{game.DeathAnnounced{Victim: "p1"}, "DeathAnnounced"},
		{game.PeacefulNight{}, "PeacefulNight"},
		{game.DiscussionTimerTick{SecondsLeft: 30}, "DiscussionTimerTick"},
		{game.VoteTallied{Counts: map[game.PlayerID]int{"p1": 1}, Eliminated: &pid}, "VoteTallied"},
		{game.Eliminated{PlayerID: "p1", Role: game.RoleMafia}, "Eliminated"},
		{game.MafiaRepresentativeReassigned{OldID: "p1", NewID: "p2"}, "MafiaRepresentativeReassigned"},
		{game.GameEnded{Winner: &winner, EndReason: game.EndCitizenWin}, "GameEnded"},
		{game.VoiceToggled{On: true}, "VoiceToggled"},
		{game.NightStepChanged{Step: game.NightStepPolice, Day: 2}, "NightStepChanged"},
		{game.GamePaused{Phase: game.PhaseNight}, "GamePaused"},
		{game.GameResumed{Phase: game.PhaseNight}, "GameResumed"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			p := buildEventPayload(tc.ev)
			if p.Kind != tc.kind {
				t.Errorf("Kind = %q, want %q", p.Kind, tc.kind)
			}
			if _, err := json.Marshal(p); err != nil {
				t.Errorf("Marshal: %v", err)
			}
		})
	}
}

func TestBuildEventPayload_PlayerJoinedCarriesName(t *testing.T) {
	p := buildEventPayload(game.PlayerJoined{PlayerID: "p7", Name: "민지"})
	if p.Kind != "PlayerJoined" {
		t.Errorf("Kind = %q, want PlayerJoined", p.Kind)
	}
	if p.PlayerID != "p7" {
		t.Errorf("PlayerID = %q, want p7", p.PlayerID)
	}
	if p.Name != "민지" {
		t.Errorf("Name = %q, want 민지", p.Name)
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(bytes), `"name":"민지"`) {
		t.Errorf("wire payload missing name field: %s", string(bytes))
	}
	if !strings.Contains(string(bytes), `"playerId":"p7"`) {
		t.Errorf("wire payload missing playerId field: %s", string(bytes))
	}
}

func TestBuildEventPayload_NightStepCarriesStep(t *testing.T) {
	p := buildEventPayload(game.NightStepChanged{Step: game.NightStepDoctor, Day: 3})
	if p.Kind != "NightStepChanged" {
		t.Errorf("Kind = %q, want NightStepChanged", p.Kind)
	}
	if p.Step != game.NightStepDoctor {
		t.Errorf("Step = %q, want DOCTOR", p.Step)
	}
	if p.Day != 3 {
		t.Errorf("Day = %d, want 3", p.Day)
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(bytes), `"step":"DOCTOR"`) {
		t.Errorf("wire payload missing step: %s", string(bytes))
	}
}

// Iteration 8 — INTRO step is passed through as a string just like the
// other steps. The wire treats NightStep as opaque enum; clients gate on
// the literal value.
func TestBuildEventPayload_NightStepIntroSerializes(t *testing.T) {
	p := buildEventPayload(game.NightStepChanged{Step: game.NightStepIntro, Day: 1})
	if p.Step != game.NightStepIntro {
		t.Errorf("Step = %q, want INTRO", p.Step)
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(bytes), `"step":"INTRO"`) {
		t.Errorf("wire payload missing step=INTRO: %s", string(bytes))
	}
}

func TestMustMarshal_RoundTrip(t *testing.T) {
	msg := errorMsg{Type: "error", Code: "VALIDATION_ERROR", Message: "bad"}
	bytes := mustMarshal(msg)
	if !strings.Contains(string(bytes), `"code":"VALIDATION_ERROR"`) {
		t.Errorf("missing code in %q", string(bytes))
	}
}
