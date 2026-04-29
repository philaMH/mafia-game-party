package session_test

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
)

// newTestManager builds a SessionManager wired to a tempdir SQLite store
// and an in-memory engine. The returned cleanup must be called by tests
// (via t.Cleanup) — t.Helper marks line numbers correctly.
func newTestManager(t *testing.T) (session.SessionManager, *game.FakeClock) {
	t.Helper()
	dir := t.TempDir()
	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "u2.db"))
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	clock := &game.FakeClock{T: time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)}
	engine := game.New(game.NewAssigner(game.NewDefaultKeywordPool()), clock, rand.Reader)

	mgr, err := session.New(store, announce.NewDefaultCatalog(), engine, clock, rand.Reader,
		session.SessionOpts{TickInterval: time.Hour}) // disable real ticking
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	return mgr, clock
}

// makeLobby creates a host + (count-1) extra players. Returns (host JoinResult, all members in join order).
func makeLobby(t *testing.T, mgr session.SessionManager, count int) (session.JoinResult, []session.JoinResult) {
	t.Helper()
	ctx := context.Background()
	host, err := mgr.CreateSession(ctx, "호스트")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	others := make([]session.JoinResult, 0, count-1)
	for i := 0; i < count-1; i++ {
		jr, err := mgr.JoinPlayer(ctx, namesPool[i])
		if err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
		others = append(others, jr)
	}
	return host, others
}

var namesPool = []string{"민수", "철수", "영희", "수정", "지훈", "서연", "지민", "예은", "도윤", "하윤", "은지"}
