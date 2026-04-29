package persistence

import (
	"context"
	"fmt"
	"os"
	"time"
)

// ArchiveCorrupt implements PersistenceStore. It closes the underlying
// connection then renames the on-disk file to a timestamped sibling so the
// next OpenSqlite call starts from a fresh database (P-U2-9).
func (s *sqliteStore) ArchiveCorrupt(ctx context.Context) error {
	_ = ctx // reserved for future cancellation; rename is fast.

	if err := s.Close(); err != nil {
		// Continue: closing may fail on an already-broken DB; rename anyway.
		_ = err
	}

	if _, err := os.Stat(s.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat: %w", err)
	}

	ts := time.Now().UTC().Format("20060102-150405")
	archived := fmt.Sprintf("%s.corrupt-%s", s.path, ts)
	if err := os.Rename(s.path, archived); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	// Best-effort: rename WAL/SHM siblings too.
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Rename(s.path+suffix, archived+suffix)
	}
	return nil
}
