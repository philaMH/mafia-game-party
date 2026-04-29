// Package session is U2's session-management layer. It owns the single GM
// mutex (P-U2-6), wraps internal/game.Engine with serialized access,
// drives a 1-second background tick loop, persists snapshots and final
// results via internal/persistence, and dispatches Korean announcement
// strings via internal/announce.
//
// Subscribe handlers run inside the manager's lock — they must return
// quickly. Heavy work (e.g., WebSocket fan-out) belongs in a separate
// goroutine fed via a buffered channel.
package session
