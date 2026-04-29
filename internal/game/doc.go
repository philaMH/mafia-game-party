// Package game implements the domain core of the mafia game: state machine,
// role assignment, voting tally, end conditions, and snapshot/restore.
//
// The package has zero external dependencies; only the Go standard library is
// used. All infrastructure concerns (persistence, transport, UI) live in other
// units that consume this package's types and interfaces.
//
// Engine is the entry point. It is not safe for concurrent use; callers must
// serialize access (the SessionManager unit holds a single mutex / actor).
//
// Construction:
//
//	pool := game.NewDefaultKeywordPool()
//	engine := game.NewDefault(pool) // crypto/rand for production
//
// Tests inject a deterministic RNG and clock:
//
//	rng := bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
//	clock := &game.FakeClock{T: time.Unix(0, 0)}
//	engine := game.New(game.NewAssigner(pool), clock, rng)
package game
