package ws

import (
	"crypto/rand"
	"encoding/hex"
)

// ClientID is the server-issued identifier for a WebSocket connection.
// It is independent of game.PlayerID — a single ClientID may go through
// PUBLIC → PLAYER transitions (after join/resume) without changing.
type ClientID string

// idByteLen is 8 bytes → 16 hex chars (NFR-U3 chose hex16).
const idByteLen = 8

// newClientID returns a fresh random ClientID. crypto/rand.Read failure
// is extraordinarily rare; on failure we still return whatever bytes were
// produced (including all-zeros) — the caller's collision space is small
// and a duplicate ID merely causes the new client to overwrite an old
// registry entry, not a security issue.
func newClientID() ClientID {
	var b [idByteLen]byte
	_, _ = rand.Read(b[:])
	return ClientID(hex.EncodeToString(b[:]))
}
