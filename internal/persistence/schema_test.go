package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// White-box test for PRAGMA application — uses the unexported applyPragmas
// to validate WAL + synchronous=NORMAL are reachable on a fresh DB.
func TestApplyPragmas_SetsWALAndSynchronous(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := applyPragmas(context.Background(), db); err != nil {
		t.Fatalf("applyPragmas: %v", err)
	}

	var mode string
	if err := db.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode want wal, got %q", mode)
	}

	var sync int
	if err := db.QueryRowContext(context.Background(), "PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}
	if sync != 1 { // 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA
		t.Errorf("synchronous want 1 (NORMAL), got %d", sync)
	}
}

func TestApplySchema_CreatesAllTables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := applySchema(context.Background(), db); err != nil {
		t.Fatalf("applySchema: %v", err)
	}

	for _, table := range []string{"active_snapshot", "game_results", "events"} {
		row := db.QueryRowContext(context.Background(),
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table)
		var got string
		if err := row.Scan(&got); err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
