package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// tokenByteLen is the entropy width for ResumePlayer tokens (NFR-U2-S1).
const tokenByteLen = 32

// errTokenCollision is returned when issueUniqueToken fails to find a
// non-colliding value within tokenAttempts. It is intentionally unwrapped:
// callers translate it into game.ErrValidation as needed.
var errTokenCollision = errors.New("token collision after retries")

// tokenAttempts caps the retry budget for issueUniqueToken. With a 256-bit
// space the probability of >1 collisions in a single LAN session is so
// small that any failure here implies a broken RNG.
const tokenAttempts = 5

// newToken returns a 32-byte (64 hex char) random token sourced from the
// supplied reader (typically crypto/rand.Reader).
func newToken(r io.Reader) (string, error) {
	if r == nil {
		r = rand.Reader
	}
	var b [tokenByteLen]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// issueUniqueToken returns a fresh token guaranteed not to collide with any
// member's existing token. It retries up to tokenAttempts times.
func (s *session) issueUniqueToken() (string, error) {
	for i := 0; i < tokenAttempts; i++ {
		t, err := newToken(s.rand)
		if err != nil {
			return "", err
		}
		if !s.tokenInUse(t) {
			return t, nil
		}
	}
	return "", errTokenCollision
}

// tokenInUse reports whether any member currently holds the given token.
// Caller must hold s.mu.
func (s *session) tokenInUse(t string) bool {
	for _, m := range s.sess.Members {
		if m.Token == t {
			return true
		}
	}
	return false
}
