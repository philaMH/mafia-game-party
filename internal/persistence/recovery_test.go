package persistence_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saltware/mafia-game/internal/persistence"
)

func TestArchiveCorrupt_RenamesDBFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := store.SaveSnapshot(context.Background(), sampleSnapshot()); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	if err := store.ArchiveCorrupt(context.Background()); err != nil {
		t.Fatalf("ArchiveCorrupt: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected original to be renamed away, got err=%v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.db.corrupt-") &&
			!strings.HasSuffix(e.Name(), "-wal") &&
			!strings.HasSuffix(e.Name(), "-shm") {
			found = true
		}
	}
	if !found {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected archived file with prefix test.db.corrupt-; got %v", names)
	}
}

func TestArchiveCorrupt_NoFileIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.db")
	store, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Manually remove the file before archive.
	_ = store.Close()
	_ = os.Remove(path)
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")

	store2, err := persistence.OpenSqlite(context.Background(), path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	_ = store2.Close()
	_ = os.Remove(path)
	if err := store2.ArchiveCorrupt(context.Background()); err != nil {
		t.Errorf("expected no error when file missing, got %v", err)
	}
}
