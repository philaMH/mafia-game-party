package persistence

import (
	"context"
	"time"

	"github.com/saltware/mafia-game/internal/game"
)

// PersistedMember is the persistence-layer view of a session participant.
// It mirrors session.Member but is defined here to avoid an
// import cycle (session imports persistence, never the reverse).
type PersistedMember struct {
	ID        game.PlayerID `json:"id"`
	Name      string        `json:"name"`
	Token     string        `json:"token"`
	Connected bool          `json:"connected"`
	JoinedAt  time.Time     `json:"joinedAt"`
}

// Snapshot is the durable record of an active game. It is overwritten in a
// single row (id=1) of the active_snapshot table on every persist trigger
// (BR-U2-PERSIST-1, BR-U2-PERSIST-2).
type Snapshot struct {
	GameID  string
	State   game.State
	Members []PersistedMember
	HostID  game.PlayerID
}

// GameResult is a completed-game summary written to game_results when a
// GameEnded event fires (BR-U2-PERSIST-3).
type GameResult struct {
	GameID    string
	StartedAt time.Time
	EndedAt   time.Time
	Winner    *game.Team
	EndReason game.EndReason
	Options   game.Options
	Members   []PersistedMember
	Reveal    []game.Player
}

// PersistenceStore is the durable backing store for U2.
// All methods must be safe to call from a single goroutine; callers
// (SessionManager) hold a single mutex while invoking these methods, so
// implementations may assume serialized access (P-U2-1).
type PersistenceStore interface {
	// SaveSnapshot upserts the single active snapshot row.
	SaveSnapshot(ctx context.Context, snap Snapshot) error

	// LoadActiveSnapshot returns the active snapshot if present.
	// found=false (and a nil error) means no active game on disk.
	LoadActiveSnapshot(ctx context.Context) (snap Snapshot, found bool, err error)

	// DeleteActiveSnapshot removes the active snapshot row (idempotent).
	DeleteActiveSnapshot(ctx context.Context) error

	// SaveResultAndClearActive inserts a game result and deletes the active
	// snapshot atomically in a single transaction (NFR-U2-R4).
	SaveResultAndClearActive(ctx context.Context, r GameResult) error

	// ListResults returns the most recent results, newest first.
	ListResults(ctx context.Context, limit int) ([]GameResult, error)

	// AppendEvent records a single domain event (optional debug log).
	// Implementations may treat this as a no-op when event logging is off.
	AppendEvent(ctx context.Context, gameID string, env game.EventEnvelope) error

	// ArchiveCorrupt renames the underlying DB file to a timestamped
	// "corrupt-" sibling so the next process boot starts on a fresh DB.
	// Subsequent calls on the same store are not supported.
	ArchiveCorrupt(ctx context.Context) error

	// Close releases all underlying resources.
	Close() error
}
