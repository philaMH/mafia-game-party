package game

import "testing"

// TestPlayerJoinedFields confirms the LOBBY membership event preserves its
// PlayerID/Name fields through value equality. Sealed-interface conformance
// is verified separately in markers_test.go.
func TestPlayerJoinedFields(t *testing.T) {
	a := PlayerJoined{PlayerID: PlayerID("p1"), Name: "민수"}
	b := PlayerJoined{PlayerID: PlayerID("p1"), Name: "민수"}
	if a != b {
		t.Errorf("PlayerJoined value equality broken: %+v vs %+v", a, b)
	}
	if a.PlayerID != "p1" || a.Name != "민수" {
		t.Errorf("PlayerJoined fields unexpected: %+v", a)
	}
}

// TestPlayerJoinedEnvelopePublic asserts that pub() yields a PUBLIC envelope
// suitable for fanout to all viewers — the routing the Session unit relies on.
func TestPlayerJoinedEnvelopePublic(t *testing.T) {
	env := pub(PlayerJoined{PlayerID: PlayerID("p2"), Name: "수민"})
	if env.Visibility != VisPublic {
		t.Errorf("pub(PlayerJoined).Visibility = %v, want VisPublic", env.Visibility)
	}
	pj, ok := env.Event.(PlayerJoined)
	if !ok {
		t.Fatalf("pub(PlayerJoined).Event type assertion failed: %T", env.Event)
	}
	if pj.PlayerID != "p2" || pj.Name != "수민" {
		t.Errorf("envelope event fields lost: %+v", pj)
	}
}
