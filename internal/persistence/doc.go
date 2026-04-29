// Package persistence is U2's storage layer. It exposes the
// PersistenceStore interface that the SessionManager uses to durably record
// the active game snapshot, completed game results, and an optional event
// log. The default backend is a single-file SQLite database opened in WAL
// mode with synchronous=NORMAL and chmod 0600 (NFR-U2-S3, NFR-U2-R7).
//
// Design highlights:
//   - Single-writer connection pool via sql.DB(MaxOpenConns=1) (P-U2-1).
//   - Prepared statement caching (P-U2-2).
//   - Result + active-snapshot deletion in one transaction (NFR-U2-R4).
//   - Corrupt-snapshot archive(rename) recovery (P-U2-9).
package persistence
