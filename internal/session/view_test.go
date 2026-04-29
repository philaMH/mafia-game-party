package session_test

import (
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

func makeState(phase game.Phase) game.State {
	return game.State{
		GameID: "g",
		Phase:  phase,
		HostID: "h",
		Players: []game.Player{
			{ID: "h", Name: "host", Alive: true, Role: game.RoleCitizen, Keyword: "kw-h"},
			{ID: "m1", Name: "m1", Alive: true, Role: game.RoleMafia, Keyword: "mafia-kw"},
			{ID: "m2", Name: "m2", Alive: true, Role: game.RoleMafia, Keyword: "mafia-kw"},
			{ID: "d", Name: "d", Alive: true, Role: game.RoleDoctor, Keyword: "kw-d"},
			{ID: "p", Name: "p", Alive: true, Role: game.RolePolice, Keyword: "kw-p"},
		},
		Votes: map[game.PlayerID]game.PlayerID{},
	}
}

func TestPrivateView_PublicMasksAllRoles(t *testing.T) {
	v := session.BuildPrivateView(makeState(game.PhaseDay), "", "h")
	for _, p := range v.State.Players {
		if p.Role != "" || p.Keyword != "" {
			t.Errorf("public view leaked: %+v", p)
		}
	}
	if v.YourRole != "" || v.IsHost {
		t.Errorf("self info leaked in public view: %+v", v)
	}
}

func TestPrivateView_PlayerSelfRoleVisible(t *testing.T) {
	v := session.BuildPrivateView(makeState(game.PhaseDay), "h", "h")
	if v.YourRole != game.RoleCitizen {
		t.Errorf("YourRole = %q", v.YourRole)
	}
	if v.YourKeyword != "kw-h" {
		t.Errorf("YourKeyword = %q", v.YourKeyword)
	}
	if !v.IsHost {
		t.Error("expected IsHost=true for host viewer")
	}
	for _, p := range v.State.Players {
		if p.ID == "h" {
			if p.Role != game.RoleCitizen || p.Keyword != "kw-h" {
				t.Errorf("self in players slice should be unmasked: %+v", p)
			}
		} else if p.Role != "" || p.Keyword != "" {
			t.Errorf("other player leaked: %+v", p)
		}
	}
}

func TestPrivateView_OtherPlayersMasked(t *testing.T) {
	v := session.BuildPrivateView(makeState(game.PhaseDay), "d", "h")
	if v.YourRole != game.RoleDoctor {
		t.Errorf("YourRole = %q", v.YourRole)
	}
	for _, p := range v.State.Players {
		if p.ID == "d" {
			continue
		}
		if p.Role != "" || p.Keyword != "" {
			t.Errorf("other player leaked: %+v", p)
		}
	}
	if v.IsHost {
		t.Error("non-host viewer should have IsHost=false")
	}
}

func TestPrivateView_MafiaSeesAllies(t *testing.T) {
	v := session.BuildPrivateView(makeState(game.PhaseNight), "m1", "h")
	if v.YourTeam != game.TeamMafia {
		t.Errorf("YourTeam = %q", v.YourTeam)
	}
	if len(v.MafiaCohort) != 2 {
		t.Errorf("expected 2 cohort entries, got %v", v.MafiaCohort)
	}
	mafiaShown := 0
	for _, p := range v.State.Players {
		if p.Role == game.RoleMafia {
			mafiaShown++
		}
	}
	if mafiaShown != 2 {
		t.Errorf("expected 2 mafia roles visible to mafia viewer, got %d", mafiaShown)
	}
	// Non-mafia players' roles still hidden.
	for _, p := range v.State.Players {
		if p.ID == "d" || p.ID == "p" || p.ID == "h" {
			if p.Role != "" {
				t.Errorf("non-mafia leaked role: %+v", p)
			}
		}
	}
	// Other-mafia keyword must remain hidden.
	for _, p := range v.State.Players {
		if p.ID == "m2" && p.Keyword != "" {
			t.Errorf("other mafia keyword leaked: %+v", p)
		}
	}
}

func TestPrivateView_GameEndedRevealsAll(t *testing.T) {
	v := session.BuildPrivateView(makeState(game.PhaseEnd), "h", "h")
	for _, p := range v.State.Players {
		if p.Role == "" {
			t.Errorf("END phase should reveal all roles, got blank for %+v", p)
		}
		if p.ID != "h" && p.Keyword != "" {
			t.Errorf("END phase should still hide keyword for non-self: %+v", p)
		}
	}
}
