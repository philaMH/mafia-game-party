package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/saltware/mafia-game/internal/game"
)

// HostToken is an opaque, unguessable identifier for the GM seat. Issued by
// HostAuthority.Claim on the first /public connection and verified on every
// host-only action. See FR-9.2 / FR-10.2.
type HostToken string

// hostAuthority gates the single-host invariant. Methods are safe for
// concurrent use; SessionManager invokes them while holding its own mutex
// (so contention is bounded).
type hostAuthority struct {
	mu      sync.Mutex
	current HostToken
}

func newHostAuthority() *hostAuthority { return &hostAuthority{} }

// Claim grants the host seat to the caller. Returns ErrHostOccupied when a
// token is already outstanding (the second /public connection is rejected).
func (a *hostAuthority) Claim() (HostToken, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.current != "" {
		return "", &game.EngineError{Code: game.CodePermissionDenied, Message: "host seat already occupied"}
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", &game.EngineError{Code: game.CodeValidation, Message: "host token rng failure"}
	}
	a.current = HostToken(hex.EncodeToString(b[:]))
	return a.current, nil
}

// Verify returns ErrInvalidHost if the token does not match the outstanding
// seat. Empty input always fails.
func (a *hostAuthority) Verify(t HostToken) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if t == "" || a.current == "" || t != a.current {
		return &game.EngineError{Code: game.CodePermissionDenied, Message: "invalid host token"}
	}
	return nil
}

// Release surrenders the host seat. A non-matching token is silently ignored
// (so a stale Release does not boot the rightful host).
func (a *hostAuthority) Release(t HostToken) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if t != "" && t == a.current {
		a.current = ""
	}
}

// IsClaimed reports whether the host seat is currently held.
func (a *hostAuthority) IsClaimed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.current != ""
}
