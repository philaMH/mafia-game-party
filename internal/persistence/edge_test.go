package persistence_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

func TestOpenSqlite_EmptyPathRejected(t *testing.T) {
	if _, err := persistence.OpenSqlite(context.Background(), ""); err == nil {
		t.Error("expected error on empty path")
	}
}

func TestAppendEvent_RoleMafiaVisibility(t *testing.T) {
	store, _ := newStore(t)
	env := game.EventEnvelope{
		Event:      game.MafiaCohortRevealed{MafiaIDs: []game.PlayerID{"p1"}, RepresentativeID: "p1"},
		Visibility: game.VisRoleMafia,
	}
	if err := store.AppendEvent(context.Background(), "g-1", env); err != nil {
		t.Errorf("AppendEvent VisRoleMafia: %v", err)
	}
}

func TestAppendEvent_UnknownVisibility(t *testing.T) {
	store, _ := newStore(t)
	env := game.EventEnvelope{
		Event:      game.PhaseChanged{Phase: game.PhaseDay, Day: 1},
		Visibility: game.Visibility(99),
	}
	if err := store.AppendEvent(context.Background(), "g-1", env); err != nil {
		t.Errorf("AppendEvent unknown: %v", err)
	}
}

func TestListResults_DefaultLimit(t *testing.T) {
	store, _ := newStore(t)
	results, err := store.ListResults(context.Background(), 0) // 0 → default 100
	if err != nil {
		t.Errorf("ListResults: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil empty slice")
	}
}

func TestSaveResultAndClearActive_NoWinnerEncodesNull(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	r := persistence.GameResult{
		GameID:    "no-winner",
		EndReason: game.EndHostForceEnd,
	}
	if err := store.SaveResultAndClearActive(ctx, r); err != nil {
		t.Fatalf("save: %v", err)
	}
	results, _ := store.ListResults(ctx, 10)
	if len(results) != 1 {
		t.Fatalf("len=%d", len(results))
	}
	if results[0].Winner != nil {
		t.Errorf("expected nil winner, got %+v", results[0].Winner)
	}
}

func TestNewStoreCreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c", "test.db")
	store, err := persistence.OpenSqlite(context.Background(), deep)
	if err != nil {
		t.Fatalf("Open with deep parent: %v", err)
	}
	_ = store.Close()
}
