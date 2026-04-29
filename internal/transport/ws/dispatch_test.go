package ws

import (
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

func TestRouteEvent_VisPublicReachesAll(t *testing.T) {
	h := &hub{registry: newClientRegistry()}
	pub := makeFakeClient("pub-1")
	plr := makeFakeClient("plr-1")
	plr.Kind = ClientPlayer
	plr.PlayerID = "p1"
	h.registry.add(pub)
	h.registry.add(plr) // PLAYER

	got := h.routeEvent(game.EventEnvelope{Visibility: game.VisPublic}, game.State{})
	if len(got) != 2 {
		t.Errorf("VisPublic should reach 2, got %d", len(got))
	}
}

func TestRouteEvent_VisPlayerSingle(t *testing.T) {
	h := &hub{registry: newClientRegistry()}
	plr := makeFakeClient("plr-1")
	plr.Kind = ClientPlayer
	plr.PlayerID = "p1"
	h.registry.add(plr)

	got := h.routeEvent(game.EventEnvelope{Visibility: game.VisPlayer, PlayerID: "p1"}, game.State{})
	if len(got) != 1 || got[0].ID != "plr-1" {
		t.Errorf("VisPlayer should reach p1 only, got %+v", got)
	}

	none := h.routeEvent(game.EventEnvelope{Visibility: game.VisPlayer, PlayerID: "missing"}, game.State{})
	if len(none) != 0 {
		t.Errorf("missing PID should reach 0, got %d", len(none))
	}
}

func TestRouteEvent_VisRoleMafiaUsesState(t *testing.T) {
	h := &hub{registry: newClientRegistry()}

	mafia := makeFakeClient("m1")
	mafia.Kind = ClientPlayer
	mafia.PlayerID = "p_mafia"
	h.registry.add(mafia)

	citizen := makeFakeClient("c1")
	citizen.Kind = ClientPlayer
	citizen.PlayerID = "p_citizen"
	h.registry.add(citizen)

	state := game.State{
		Players: []game.Player{
			{ID: "p_mafia", Alive: true, Role: game.RoleMafia},
			{ID: "p_citizen", Alive: true, Role: game.RoleCitizen},
			{ID: "p_dead_mafia", Alive: false, Role: game.RoleMafia},
		},
	}
	got := h.routeEvent(game.EventEnvelope{Visibility: game.VisRoleMafia}, state)
	if len(got) != 1 || got[0].PlayerID != "p_mafia" {
		t.Errorf("VisRoleMafia should reach p_mafia only, got %+v", got)
	}
}

func TestRouteEvent_UnknownVisibility(t *testing.T) {
	h := &hub{registry: newClientRegistry()}
	if got := h.routeEvent(game.EventEnvelope{Visibility: game.Visibility(99)}, game.State{}); got != nil {
		t.Errorf("unknown visibility should return nil, got %+v", got)
	}
}

func TestTimeToMs_NegativeReturnsZero(t *testing.T) {
	if timeToMs(-1) != 0 {
		t.Error("negative epoch should return 0")
	}
	if timeToMs(123) != 123 {
		t.Error("positive epoch should pass through")
	}
}
