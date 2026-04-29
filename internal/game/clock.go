package game

import "time"

// Clock supplies the current time. The engine accepts a Clock so tests can
// inject deterministic time progression.
type Clock interface {
	Now() time.Time
}

// realClock returns time.Now().
type realClock struct{}

// Now implements Clock.
func (realClock) Now() time.Time { return time.Now() }

// FakeClock is a controllable Clock for tests. Mutating T directly is
// allowed; Advance is a convenience.
type FakeClock struct {
	T time.Time
}

// Now implements Clock.
func (c *FakeClock) Now() time.Time { return c.T }

// Advance moves the fake clock forward by d.
func (c *FakeClock) Advance(d time.Duration) { c.T = c.T.Add(d) }
