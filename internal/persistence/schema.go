package persistence

import (
	"context"
	"database/sql"
	"fmt"
)

// schemaDDL contains all CREATE TABLE / CREATE INDEX statements applied at
// store open time. They are idempotent so existing databases skip work.
//
// Mirrors functional-design/domain-entities.md §6.1.
const schemaDDL = `
CREATE TABLE IF NOT EXISTS active_snapshot (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    game_id     TEXT    NOT NULL,
    state_json  BLOB    NOT NULL,
    member_json BLOB    NOT NULL,
    host_id     TEXT    NOT NULL,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_results (
    game_id      TEXT    PRIMARY KEY,
    started_at   DATETIME NOT NULL,
    ended_at     DATETIME NOT NULL,
    winner       TEXT,
    end_reason   TEXT    NOT NULL,
    options_json BLOB    NOT NULL,
    members_json BLOB    NOT NULL,
    reveal_json  BLOB    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_game_results_ended_at ON game_results(ended_at DESC);

CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id      TEXT    NOT NULL,
    event_type   TEXT    NOT NULL,
    visibility   TEXT    NOT NULL,
    recipient_id TEXT,
    payload_json BLOB    NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_game_id ON events(game_id);
`

// applyPragmas configures WAL journal mode and synchronous=NORMAL.
// These are enforced even if the DSN already requested them, so that
// schema_test can verify them via PRAGMA queries (NFR-U2-R7).
func applyPragmas(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("pragma %q: %w", s, err)
		}
	}
	return nil
}

// applySchema executes schemaDDL.
func applySchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
