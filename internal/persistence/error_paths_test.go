package persistence_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

// After Close, every method returns an error. This raises coverage on the
// error branches of SaveSnapshot / DeleteActiveSnapshot / etc. without
// requiring fault injection.
func TestStore_OperationsFailAfterClose(t *testing.T) {
	dir := t.TempDir()
	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx := context.Background()

	if err := store.SaveSnapshot(ctx, sampleSnapshot()); err == nil {
		t.Error("expected error post-close: SaveSnapshot")
	}
	if _, _, err := store.LoadActiveSnapshot(ctx); err == nil {
		t.Error("expected error post-close: Load")
	}
	if err := store.DeleteActiveSnapshot(ctx); err == nil {
		t.Error("expected error post-close: Delete")
	}
	if err := store.SaveResultAndClearActive(ctx, persistence.GameResult{
		GameID: "g", EndReason: game.EndHostForceEnd,
	}); err == nil {
		t.Error("expected error post-close: SaveResult")
	}
	if _, err := store.ListResults(ctx, 10); err == nil {
		t.Error("expected error post-close: ListResults")
	}
	if err := store.AppendEvent(ctx, "g", game.EventEnvelope{
		Event: game.PhaseChanged{Phase: game.PhaseDay}, Visibility: game.VisPublic,
	}); err == nil {
		t.Error("expected error post-close: AppendEvent")
	}
}
