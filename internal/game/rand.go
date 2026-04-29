package game

import (
	"encoding/binary"
	"io"
	"math/rand"
)

// extractSeed64 reads 8 bytes from rng and returns them as an int64 seed.
// The engine uses this to derive a deterministic inner *rand.Rand from a
// caller-supplied source (crypto/rand in production, a fixed bytes.Reader in
// tests).
func extractSeed64(rng io.Reader) (int64, error) {
	var b [8]byte
	if _, err := io.ReadFull(rng, b[:]); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b[:])), nil
}

// newInnerRand creates a *rand.Rand whose seed is read from rng.
func newInnerRand(rng io.Reader) (*rand.Rand, error) {
	seed, err := extractSeed64(rng)
	if err != nil {
		return nil, err
	}
	return rand.New(rand.NewSource(seed)), nil
}
