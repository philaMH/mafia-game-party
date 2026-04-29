package persistence_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/saltware/mafia-game/internal/persistence"
)

// LoadActiveSnapshot must surface a JSON unmarshal error when the on-disk
// payload is corrupt — covers the unmarshal error branches.
func TestLoadActiveSnapshot_CorruptStateJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.db")

	// Open via the regular constructor so schema is in place, then close.
	store, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Inject a row with malformed JSON in state_json.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT OR REPLACE INTO active_snapshot (id, game_id, state_json, member_json, host_id) VALUES (1, ?, ?, ?, ?)`,
		"g-corrupt", []byte("not json"), []byte("[]"), "h",
	); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("raw close: %v", err)
	}

	// Re-open and Load — must surface unmarshal error.
	store2, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer func() { _ = store2.Close() }()
	if _, _, err := store2.LoadActiveSnapshot(context.Background()); err == nil {
		t.Error("expected unmarshal error on corrupt state JSON")
	}
}

// Inject malformed members JSON instead — exercises the second unmarshal branch.
func TestLoadActiveSnapshot_CorruptMembersJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt2.db")

	store, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("raw open: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT OR REPLACE INTO active_snapshot (id, game_id, state_json, member_json, host_id) VALUES (1, ?, ?, ?, ?)`,
		"g", []byte(`{"phase":"DAY","day":1}`), []byte("not-json"), "h",
	); err != nil {
		t.Fatalf("inject: %v", err)
	}
	_ = raw.Close()

	store2, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer func() { _ = store2.Close() }()
	if _, _, err := store2.LoadActiveSnapshot(context.Background()); err == nil {
		t.Error("expected unmarshal error on corrupt members JSON")
	}
}

// ListResults: corrupt options_json — exercises that branch.
func TestListResults_CorruptOptionsJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt3.db")

	store, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("raw: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO game_results (game_id, started_at, ended_at, end_reason, options_json, members_json, reveal_json) VALUES (?, '2026-04-26', '2026-04-26', 'HOST_FORCE_END', ?, ?, ?)`,
		"g", []byte("not-json"), []byte("[]"), []byte("[]"),
	); err != nil {
		t.Fatalf("inject: %v", err)
	}
	_ = raw.Close()

	store2, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer func() { _ = store2.Close() }()
	if _, err := store2.ListResults(context.Background(), 10); err == nil {
		t.Error("expected unmarshal error on ListResults")
	}
}
