package persistence_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

func newStore(t *testing.T) (persistence.PersistenceStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, path
}

func sampleSnapshot() persistence.Snapshot {
	return persistence.Snapshot{
		GameID: "game-1",
		State: game.State{
			GameID:    "game-1",
			Phase:     game.PhaseDay,
			Day:       2,
			HostID:    game.PlayerID("h"),
			Players:   []game.Player{{ID: "h", Name: "host", Alive: true}},
			Settings:  game.DefaultOptions(6),
			StartedAt: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
			Votes:     map[game.PlayerID]game.PlayerID{},
		},
		Members: []persistence.PersistedMember{
			{ID: "h", Name: "host", Token: "tok-host", JoinedAt: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)},
		},
		HostID: game.PlayerID("h"),
	}
}

func TestOpenSqlite_CreatesFileWith0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode bits not portable on Windows")
	}
	_, path := newStore(t)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 perm, got %v", info.Mode().Perm())
	}
}

func TestSaveSnapshot_LoadRoundTrip(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	snap := sampleSnapshot()
	if err := store.SaveSnapshot(ctx, snap); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	loaded, found, err := store.LoadActiveSnapshot(ctx)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !found {
		t.Fatal("expected found=true after save")
	}
	if loaded.GameID != snap.GameID {
		t.Errorf("GameID mismatch: %q vs %q", loaded.GameID, snap.GameID)
	}
	if loaded.HostID != snap.HostID {
		t.Errorf("HostID mismatch: %q vs %q", loaded.HostID, snap.HostID)
	}
	if loaded.State.Phase != game.PhaseDay {
		t.Errorf("Phase mismatch: %q", loaded.State.Phase)
	}
	if len(loaded.Members) != 1 || loaded.Members[0].Token != "tok-host" {
		t.Errorf("Members round-trip wrong: %+v", loaded.Members)
	}
}

func TestLoadActiveSnapshot_EmptyReturnsFalse(t *testing.T) {
	store, _ := newStore(t)
	_, found, err := store.LoadActiveSnapshot(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if found {
		t.Error("expected found=false on empty store")
	}
}

func TestSaveSnapshot_OverwritesSingleRow(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		snap := sampleSnapshot()
		snap.State.Day = i + 1
		if err := store.SaveSnapshot(ctx, snap); err != nil {
			t.Fatalf("save#%d: %v", i, err)
		}
	}
	loaded, _, _ := store.LoadActiveSnapshot(ctx)
	if loaded.State.Day != 3 {
		t.Errorf("expected last write Day=3, got %d", loaded.State.Day)
	}
}

func TestDeleteActiveSnapshot_Idempotent(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	if err := store.DeleteActiveSnapshot(ctx); err != nil {
		t.Fatalf("delete on empty: %v", err)
	}
	_ = store.SaveSnapshot(ctx, sampleSnapshot())
	if err := store.DeleteActiveSnapshot(ctx); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, found, _ := store.LoadActiveSnapshot(ctx)
	if found {
		t.Error("expected snapshot gone after delete")
	}
}

func TestSaveResultAndClearActive_AtomicallyClears(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	if err := store.SaveSnapshot(ctx, sampleSnapshot()); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	winner := game.TeamCitizen
	if err := store.SaveResultAndClearActive(ctx, persistence.GameResult{
		GameID:    "game-1",
		StartedAt: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
		EndedAt:   time.Date(2026, 4, 26, 10, 30, 0, 0, time.UTC),
		Winner:    &winner,
		EndReason: game.EndCitizenWin,
		Options:   game.DefaultOptions(6),
		Members:   []persistence.PersistedMember{{ID: "h", Name: "host", Token: "tok"}},
		Reveal:    []game.Player{{ID: "h", Name: "host", Role: game.RoleCitizen}},
	}); err != nil {
		t.Fatalf("SaveResultAndClearActive: %v", err)
	}

	_, found, _ := store.LoadActiveSnapshot(ctx)
	if found {
		t.Error("active snapshot should be cleared")
	}
	results, err := store.ListResults(ctx, 10)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Winner == nil || *results[0].Winner != game.TeamCitizen {
		t.Errorf("winner round-trip wrong: %+v", results[0].Winner)
	}
	if results[0].EndReason != game.EndCitizenWin {
		t.Errorf("endReason round-trip wrong: %q", results[0].EndReason)
	}
}

func TestSaveResult_DuplicateGameIDFails(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	r := persistence.GameResult{
		GameID:    "dup",
		StartedAt: time.Now().UTC(),
		EndedAt:   time.Now().UTC(),
		EndReason: game.EndHostForceEnd,
	}
	if err := store.SaveResultAndClearActive(ctx, r); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := store.SaveResultAndClearActive(ctx, r); err == nil {
		t.Error("expected duplicate GameID to fail")
	}
}

func TestListResults_OrdersByEndedAtDesc(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	for i, ts := range []time.Time{
		time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC),
	} {
		err := store.SaveResultAndClearActive(ctx, persistence.GameResult{
			GameID:    "g-" + string(rune('a'+i)),
			StartedAt: ts.Add(-30 * time.Minute),
			EndedAt:   ts,
			EndReason: game.EndCitizenWin,
		})
		if err != nil {
			t.Fatalf("save #%d: %v", i, err)
		}
	}

	results, err := store.ListResults(ctx, 10)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
	for i := 0; i+1 < len(results); i++ {
		if results[i].EndedAt.Before(results[i+1].EndedAt) {
			t.Errorf("not desc: results[%d]=%v < results[%d]=%v", i, results[i].EndedAt, i+1, results[i+1].EndedAt)
		}
	}
}

func TestAppendEvent_Inserts(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	env := game.EventEnvelope{
		Event:      game.PhaseChanged{Phase: game.PhaseDay, Day: 1},
		Visibility: game.VisPublic,
	}
	if err := store.AppendEvent(ctx, "g-1", env); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	envPriv := game.EventEnvelope{
		Event:      game.RoleRevealedToPlayer{PlayerID: "p1", Role: game.RoleMafia, Keyword: "kw"},
		Visibility: game.VisPlayer,
		PlayerID:   "p1",
	}
	if err := store.AppendEvent(ctx, "g-1", envPriv); err != nil {
		t.Fatalf("AppendEvent priv: %v", err)
	}
}

func TestClose_IsIdempotent(t *testing.T) {
	store, _ := newStore(t)
	if err := store.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}
