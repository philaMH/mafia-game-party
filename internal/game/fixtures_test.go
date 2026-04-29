package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"testing"
	"time"
)

// deterministicRNG returns an io.Reader that yields a SHA-256-derived stream
// from the given seed. Used as a stand-in for crypto/rand.Reader in tests.
func deterministicRNG(seed int64) io.Reader {
	var seedBytes [8]byte
	binary.LittleEndian.PutUint64(seedBytes[:], uint64(seed))
	h := sha256.Sum256(seedBytes[:])
	// Return a long buffer derived by chaining hashes for plenty of entropy.
	buf := make([]byte, 0, 1024)
	cur := h
	for len(buf) < 1024 {
		buf = append(buf, cur[:]...)
		cur = sha256.Sum256(cur[:])
	}
	return bytes.NewReader(buf)
}

// playerSet builds n test players with stable IDs ("p1".."pN").
func playerSet(n int) []Player {
	out := make([]Player, n)
	for i := 0; i < n; i++ {
		out[i] = Player{
			ID:    PlayerID(testID(i)),
			Name:  testID(i),
			Alive: true,
		}
	}
	return out
}

// testID returns "pN" for index 0..n-1 (1-based).
func testID(i int) string {
	const digits = "0123456789"
	b := []byte{'p'}
	n := i + 1
	if n >= 10 {
		b = append(b, digits[n/10])
	}
	b = append(b, digits[n%10])
	return string(b)
}

// newTestEngine constructs an Engine wired with deterministic dependencies.
func newTestEngine(t *testing.T, seed int64) (Engine, *FakeClock) {
	t.Helper()
	clock := &FakeClock{T: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)}
	pool := NewDefaultKeywordPool()
	rng := deterministicRNG(seed)
	return New(NewAssigner(pool), clock, rng), clock
}

// mustStart runs Start with the supplied options and returns the resulting
// state and emitted events. Test fails on error.
func mustStart(t *testing.T, e Engine, players []Player, host PlayerID, opts Options) (State, []EventEnvelope) {
	t.Helper()
	state, evs, err := e.Start("g1", host, players, opts)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	return state, evs
}

// findRole returns the first PlayerID with the given role from a state.
func findRole(s State, role Role) (PlayerID, bool) {
	for _, p := range s.Players {
		if p.Role == role {
			return p.ID, true
		}
	}
	return "", false
}

// engineFakeClock returns the *FakeClock backing the given Engine. Panics
// when the engine was not built with newTestEngine — useful for helpers
// that need to advance time without threading the clock through every
// call site.
func engineFakeClock(t *testing.T, e Engine) *FakeClock {
	t.Helper()
	impl, ok := e.(*engine)
	if !ok {
		t.Fatalf("engineFakeClock: engine is %T, not *engine", e)
	}
	fc, ok := impl.clock.(*FakeClock)
	if !ok {
		t.Fatalf("engineFakeClock: clock is %T, not *FakeClock", impl.clock)
	}
	return fc
}

// drainNightIntro advances the engine past the NightStepIntro buffer when
// the current night step is INTRO. No-op otherwise. Iteration 8: tests
// that re-enter NIGHT (e.g. via vote on Day 2+) should call this before
// submitting role actions, matching what advanceToNight does for Night 1.
func drainNightIntro(t *testing.T, e Engine) {
	t.Helper()
	snap := e.Snapshot()
	if snap.Phase != PhaseNight || snap.NightStep != NightStepIntro {
		return
	}
	fc := engineFakeClock(t, e)
	fc.T = snap.NightStepDeadline.Add(time.Millisecond)
	if _, _, err := e.Tick(fc.T); err != nil {
		t.Fatalf("drainNightIntro: %v", err)
	}
}

// advanceNightStep ticks the engine past the current NightStep deadline,
// causing the engine to either move to the next step or (when DOCTOR
// expires) call resolveNight() and transition to DAY. Iteration 5: this
// is the canonical helper for tests that previously relied on
// submission auto-advancing the night sequence.
func advanceNightStep(t *testing.T, e Engine, clock *FakeClock) {
	t.Helper()
	snap := e.Snapshot()
	if snap.Phase != PhaseNight {
		t.Fatalf("advanceNightStep: not in NIGHT (phase=%s)", snap.Phase)
	}
	if snap.NightStepDeadline.IsZero() {
		t.Fatalf("advanceNightStep: NightStepDeadline is zero (step=%s)", snap.NightStep)
	}
	if !clock.Now().After(snap.NightStepDeadline) {
		clock.T = snap.NightStepDeadline.Add(time.Millisecond)
	}
	if _, _, err := e.Tick(clock.Now()); err != nil {
		t.Fatalf("Tick: %v", err)
	}
}

// allRoles returns IDs by role, useful for handler-level tests that need
// to address specific roles.
func allRoles(s State) (mafia []PlayerID, doctor PlayerID, police PlayerID, citizens []PlayerID) {
	for _, p := range s.Players {
		switch p.Role {
		case RoleMafia:
			mafia = append(mafia, p.ID)
		case RoleDoctor:
			doctor = p.ID
		case RolePolice:
			police = p.ID
		case RoleCitizen:
			citizens = append(citizens, p.ID)
		}
	}
	return
}
