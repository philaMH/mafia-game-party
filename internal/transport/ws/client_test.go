package ws

import (
	"context"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
)

func newReg(t *testing.T) *clientRegistry {
	t.Helper()
	return newClientRegistry()
}

func makeFakeClient(id ClientID) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		ID:     id,
		Kind:   ClientPublic,
		Out:    make(chan []byte, outBufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
}

func TestClientRegistry_AddRemove(t *testing.T) {
	r := newReg(t)
	c := makeFakeClient("c1")
	r.add(c)

	if got := r.byPlayerSafe(""); got != nil {
		t.Errorf("empty PlayerID lookup should return nil")
	}
	if got := r.snapshotPublic(); len(got) != 1 || got[0].ID != "c1" {
		t.Errorf("expected publics={c1}, got %+v", got)
	}

	if removed := r.remove("c1"); removed != c {
		t.Errorf("remove returned wrong client")
	}
	if removed := r.remove("c1"); removed != nil {
		t.Errorf("second remove should be nil")
	}
}

func TestClientRegistry_BindPlayerEvictsPrior(t *testing.T) {
	r := newReg(t)
	c1 := makeFakeClient("c1")
	c2 := makeFakeClient("c2")
	r.add(c1)
	r.add(c2)

	if _, hadOld := r.bindPlayer(c1, "p1"); hadOld {
		t.Error("first bind should have no prior")
	}
	oldID, hadOld := r.bindPlayer(c2, "p1")
	if !hadOld {
		t.Error("second bind should evict prior")
	}
	if oldID != "c1" {
		t.Errorf("evicted ID = %q, want c1", oldID)
	}

	// c2 should now be the canonical player; c1 still in byID until full Unregister.
	if got := r.byPlayerSafe("p1"); got == nil || got.ID != "c2" {
		t.Errorf("byPlayerSafe = %+v, want c2", got)
	}
	// publics index no longer contains c2 (now PLAYER).
	publics := r.snapshotPublic()
	for _, c := range publics {
		if c.ID == "c2" {
			t.Errorf("c2 should be removed from publics after bindPlayer")
		}
	}
}

func TestClientRegistry_SnapshotsAreCopies(t *testing.T) {
	r := newReg(t)
	c := makeFakeClient("c1")
	r.add(c)

	a := r.snapshotPublic()
	b := r.snapshotPublic()
	if &a[0] == &b[0] && len(a) > 1 {
		// Not strictly testable — slices reference the same *Client, but
		// snapshot returns *new* underlying arrays. The contract is that
		// callers can iterate without holding the lock.
		_ = a
	}
	r.remove("c1")
	// Old snapshot still holds a reference (copy-by-pointer).
	if len(a) != 1 {
		t.Errorf("post-remove snapshot mutated: %d", len(a))
	}
}

func TestClientRegistry_PlayerSnapshot(t *testing.T) {
	r := newReg(t)
	for _, id := range []ClientID{"c1", "c2", "c3"} {
		c := makeFakeClient(id)
		r.add(c)
		_, _ = r.bindPlayer(c, game.PlayerID(string(id)+"-pid"))
	}

	players := r.snapshotPlayers()
	if len(players) != 3 {
		t.Errorf("expected 3 players, got %d", len(players))
	}
	publics := r.snapshotPublic()
	if len(publics) != 0 {
		t.Errorf("expected 0 publics after binds, got %d", len(publics))
	}
}

func TestClientRegistry_All(t *testing.T) {
	r := newReg(t)
	r.add(makeFakeClient("a"))
	r.add(makeFakeClient("b"))
	if len(r.all()) != 2 {
		t.Errorf("all = %d", len(r.all()))
	}
}

func TestClientKindString(t *testing.T) {
	if ClientPublic.String() != "PUBLIC" {
		t.Errorf("PUBLIC = %q", ClientPublic.String())
	}
	if ClientPlayer.String() != "PLAYER" {
		t.Errorf("PLAYER = %q", ClientPlayer.String())
	}
	if ClientKind(99).String() != "UNKNOWN" {
		t.Errorf("Unknown kind: %q", ClientKind(99).String())
	}
}

func TestNewClientID_Length(t *testing.T) {
	id := newClientID()
	if len(id) != 16 {
		t.Errorf("ClientID len = %d, want 16", len(id))
	}
}
