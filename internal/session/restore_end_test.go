package session_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
)

// TestRestore_FinalizesEndPhaseSnapshot covers BR-U2-RESTORE-6 +
// buildResultFromState + handleGameEnd's SaveResult path. We construct a
// PhaseEnd snapshot directly on disk (simulating "crash right at end"),
// then re-open the manager and confirm the active snapshot is cleared
// and game_results contains 1 row.
func TestRestore_FinalizesEndPhaseSnapshot(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "u2.db")

	// Create the schema by opening once.
	store, err := persistence.OpenSqlite(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Inject an end-state snapshot.
	winner := game.TeamCitizen
	endReason := game.EndCitizenWin
	state := game.State{
		GameID:    "g-end",
		Phase:     game.PhaseEnd,
		Day:       3,
		HostID:    "h",
		StartedAt: time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
		Players: []game.Player{
			{ID: "h", Name: "host", Alive: true, Role: game.RoleCitizen, Keyword: "kw1"},
		},
		Settings:  game.DefaultOptions(6),
		Votes:     map[game.PlayerID]game.PlayerID{},
		Winner:    &winner,
		EndReason: &endReason,
	}
	stateJSON, _ := json.Marshal(state)
	members := []persistence.PersistedMember{
		{ID: "h", Name: "host", Token: "tok-h", JoinedAt: time.Now()},
	}
	memberJSON, _ := json.Marshal(members)

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("raw: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT OR REPLACE INTO active_snapshot (id, game_id, state_json, member_json, host_id) VALUES (1, ?, ?, ?, ?)`,
		"g-end", stateJSON, memberJSON, "h",
	); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("raw close: %v", err)
	}

	// Re-open via the regular constructor + session.New. bootRestore should
	// detect the END snapshot and finalize it.
	store2, err := persistence.OpenSqlite(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	clock := &game.FakeClock{T: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)}
	engine := game.New(game.NewAssigner(game.NewDefaultKeywordPool()), clock, rand.Reader)
	mgr, err := session.New(store2, announce.NewDefaultCatalog(), engine, clock, rand.Reader,
		session.SessionOpts{TickInterval: time.Hour})
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	defer func() { _ = mgr.Close(context.Background()) }()

	// Active snapshot should be cleared.
	loaded, found, err := store2.LoadActiveSnapshot(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if found {
		t.Errorf("expected active snapshot cleared after end-phase restore, got %+v", loaded)
	}
	results, err := store2.ListResults(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 finalized result, got %d", len(results))
	}
}
