package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/saltware/mafia-game/internal/game"
)

// sqliteStore is the modernc.org/sqlite implementation of PersistenceStore.
type sqliteStore struct {
	db   *sql.DB
	path string

	saveSnapshot   *sql.Stmt
	loadSnapshot   *sql.Stmt
	deleteSnapshot *sql.Stmt
	saveResult     *sql.Stmt
	listResults    *sql.Stmt
	appendEvent    *sql.Stmt
}

// OpenSqlite opens (or creates) a SQLite store at path. The parent directory
// is created with mode 0700 if missing; the file itself is chmod'd to 0600
// after the first open (NFR-U2-S3).
func OpenSqlite(ctx context.Context, path string) (PersistenceStore, error) {
	if path == "" {
		return nil, errors.New("persistence: empty path")
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mkdir %q: %w", dir, err)
		}
	}

	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	if err := applyPragmas(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := applySchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &sqliteStore{db: db, path: path}
	if err := s.prepareStmts(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("chmod 0600: %w", err)
	}
	return s, nil
}

func (s *sqliteStore) prepareStmts(ctx context.Context) error {
	plan := []struct {
		dst **sql.Stmt
		sql string
	}{
		{&s.saveSnapshot, `INSERT OR REPLACE INTO active_snapshot
			(id, game_id, state_json, member_json, host_id, updated_at)
			VALUES (1, ?, ?, ?, ?, CURRENT_TIMESTAMP)`},
		{&s.loadSnapshot, `SELECT game_id, state_json, member_json, host_id
			FROM active_snapshot WHERE id = 1`},
		{&s.deleteSnapshot, `DELETE FROM active_snapshot WHERE id = 1`},
		{&s.saveResult, `INSERT INTO game_results
			(game_id, started_at, ended_at, winner, end_reason,
			 options_json, members_json, reveal_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`},
		{&s.listResults, `SELECT game_id, started_at, ended_at, winner, end_reason,
			options_json, members_json, reveal_json
			FROM game_results ORDER BY ended_at DESC LIMIT ?`},
		{&s.appendEvent, `INSERT INTO events
			(game_id, event_type, visibility, recipient_id, payload_json)
			VALUES (?, ?, ?, ?, ?)`},
	}
	for _, p := range plan {
		stmt, err := s.db.PrepareContext(ctx, p.sql)
		if err != nil {
			return fmt.Errorf("prepare: %w", err)
		}
		*p.dst = stmt
	}
	return nil
}

// SaveSnapshot implements PersistenceStore.
func (s *sqliteStore) SaveSnapshot(ctx context.Context, snap Snapshot) error {
	stateJSON, err := json.Marshal(snap.State)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	memberJSON, err := json.Marshal(snap.Members)
	if err != nil {
		return fmt.Errorf("marshal members: %w", err)
	}
	if _, err := s.saveSnapshot.ExecContext(ctx,
		snap.GameID, stateJSON, memberJSON, string(snap.HostID),
	); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

// LoadActiveSnapshot implements PersistenceStore.
func (s *sqliteStore) LoadActiveSnapshot(ctx context.Context) (Snapshot, bool, error) {
	row := s.loadSnapshot.QueryRowContext(ctx)

	var (
		gameID     string
		stateJSON  []byte
		memberJSON []byte
		hostID     string
	)
	if err := row.Scan(&gameID, &stateJSON, &memberJSON, &hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, fmt.Errorf("scan snapshot: %w", err)
	}

	var state game.State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return Snapshot{}, false, fmt.Errorf("unmarshal state: %w", err)
	}
	var members []PersistedMember
	if err := json.Unmarshal(memberJSON, &members); err != nil {
		return Snapshot{}, false, fmt.Errorf("unmarshal members: %w", err)
	}

	return Snapshot{
		GameID:  gameID,
		State:   state,
		Members: members,
		HostID:  game.PlayerID(hostID),
	}, true, nil
}

// DeleteActiveSnapshot implements PersistenceStore.
func (s *sqliteStore) DeleteActiveSnapshot(ctx context.Context) error {
	if _, err := s.deleteSnapshot.ExecContext(ctx); err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}
	return nil
}

// SaveResultAndClearActive implements PersistenceStore.
func (s *sqliteStore) SaveResultAndClearActive(ctx context.Context, r GameResult) error {
	if s.db == nil {
		return errors.New("persistence: store is closed")
	}
	optsJSON, err := json.Marshal(r.Options)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	membersJSON, err := json.Marshal(r.Members)
	if err != nil {
		return fmt.Errorf("marshal members: %w", err)
	}
	revealJSON, err := json.Marshal(r.Reveal)
	if err != nil {
		return fmt.Errorf("marshal reveal: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var winner sql.NullString
	if r.Winner != nil {
		winner = sql.NullString{String: string(*r.Winner), Valid: true}
	}
	if _, err := tx.StmtContext(ctx, s.saveResult).ExecContext(ctx,
		r.GameID,
		r.StartedAt.UTC(),
		r.EndedAt.UTC(),
		winner,
		string(r.EndReason),
		optsJSON,
		membersJSON,
		revealJSON,
	); err != nil {
		return fmt.Errorf("insert result: %w", err)
	}
	if _, err := tx.StmtContext(ctx, s.deleteSnapshot).ExecContext(ctx); err != nil {
		return fmt.Errorf("delete active in tx: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListResults implements PersistenceStore.
func (s *sqliteStore) ListResults(ctx context.Context, limit int) ([]GameResult, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.listResults.QueryContext(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]GameResult, 0, limit)
	for rows.Next() {
		var (
			r           GameResult
			startedAt   time.Time
			endedAt     time.Time
			winner      sql.NullString
			endReason   string
			optsJSON    []byte
			membersJSON []byte
			revealJSON  []byte
		)
		if err := rows.Scan(
			&r.GameID, &startedAt, &endedAt, &winner, &endReason,
			&optsJSON, &membersJSON, &revealJSON,
		); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		r.StartedAt = startedAt
		r.EndedAt = endedAt
		if winner.Valid {
			t := game.Team(winner.String)
			r.Winner = &t
		}
		r.EndReason = game.EndReason(endReason)
		if err := json.Unmarshal(optsJSON, &r.Options); err != nil {
			return nil, fmt.Errorf("unmarshal options: %w", err)
		}
		if err := json.Unmarshal(membersJSON, &r.Members); err != nil {
			return nil, fmt.Errorf("unmarshal members: %w", err)
		}
		if err := json.Unmarshal(revealJSON, &r.Reveal); err != nil {
			return nil, fmt.Errorf("unmarshal reveal: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// AppendEvent implements PersistenceStore.
func (s *sqliteStore) AppendEvent(ctx context.Context, gameID string, env game.EventEnvelope) error {
	payload, err := json.Marshal(env.Event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	visibility := visibilityToString(env.Visibility)
	var recipient sql.NullString
	if env.Visibility == game.VisPlayer {
		recipient = sql.NullString{String: string(env.PlayerID), Valid: true}
	}
	eventType := fmt.Sprintf("%T", env.Event)
	if _, err := s.appendEvent.ExecContext(ctx,
		gameID, eventType, visibility, recipient, payload,
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

func visibilityToString(v game.Visibility) string {
	switch v {
	case game.VisPublic:
		return "PUBLIC"
	case game.VisPlayer:
		return "PLAYER"
	case game.VisRoleMafia:
		return "ROLE_MAFIA"
	default:
		return "UNKNOWN"
	}
}

// Close implements PersistenceStore.
func (s *sqliteStore) Close() error {
	stmts := []*sql.Stmt{
		s.saveSnapshot, s.loadSnapshot, s.deleteSnapshot,
		s.saveResult, s.listResults, s.appendEvent,
	}
	for _, st := range stmts {
		if st != nil {
			_ = st.Close()
		}
	}
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	return nil
}
