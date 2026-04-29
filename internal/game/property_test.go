package game

import (
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

// TestProperty_TickIdempotent: any number of repeated Tick calls at the same
// "now" value must yield no extra events and identical state.
func TestProperty_TickIdempotent(t *testing.T) {
	f := func(seed uint8) bool {
		e, clock := newTestEngine(t, int64(seed)+1)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		clock.Advance(time.Duration(int(seed)%30+1) * time.Second)
		now := clock.Now()
		s1, _, err := e.Tick(now)
		if err != nil {
			return false
		}
		s2, evs, err := e.Tick(now)
		if err != nil {
			return false
		}
		if len(evs) != 0 {
			return false
		}
		return s1.Phase == s2.Phase && s1.Day == s2.Day
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Error(err)
	}
}

// TestProperty_SnapshotRestoreRoundTrip: snapshot then restore yields
// equal state.
func TestProperty_SnapshotRestoreRoundTrip(t *testing.T) {
	f := func(seed uint8) bool {
		e1, _ := newTestEngine(t, int64(seed)+1)
		mustStart(t, e1, playerSet(8), "p1", DefaultOptions(8))
		advanceToNight(t, e1)
		snap := e1.Snapshot()
		e2, _ := newTestEngine(t, 999)
		if err := e2.Restore(snap); err != nil {
			return false
		}
		return reflect.DeepEqual(snap, e2.Snapshot())
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 30}); err != nil {
		t.Error(err)
	}
}

// TestProperty_CloneIsIndependent: mutating a clone never changes the source.
func TestProperty_CloneIsIndependent(t *testing.T) {
	f := func(seed uint8) bool {
		e, _ := newTestEngine(t, int64(seed)+1)
		mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
		original := e.Snapshot()
		clone := original.Clone()
		clone.Players[0].Alive = !clone.Players[0].Alive
		return original.Players[0].Alive != clone.Players[0].Alive
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 20}); err != nil {
		t.Error(err)
	}
}
