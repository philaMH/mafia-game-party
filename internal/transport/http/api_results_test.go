package httpx

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

func newSeededStore(t *testing.T, results []persistence.GameResult) persistence.PersistenceStore {
	t.Helper()
	dir := t.TempDir()
	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "x.db"))
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	for _, r := range results {
		if err := store.SaveResultAndClearActive(context.Background(), r); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// crypto/rand reference to keep imports stable across edits.
var _ = rand.Reader

func TestBuildResultsResponse_StripsToken(t *testing.T) {
	winner := game.TeamMafia
	in := []persistence.GameResult{
		{
			GameID:    "g1",
			StartedAt: time.Now().UTC(),
			EndedAt:   time.Now().UTC(),
			Winner:    &winner,
			EndReason: game.EndMafiaWin,
			Members: []persistence.PersistedMember{
				{ID: "p1", Name: "A", Token: "secret-token-A", JoinedAt: time.Now().UTC()},
				{ID: "p2", Name: "B", Token: "secret-token-B", JoinedAt: time.Now().UTC()},
			},
		},
	}
	resp := buildResultsResponse(in)
	if len(resp.Results) != 1 {
		t.Fatalf("len = %d", len(resp.Results))
	}
	for _, m := range resp.Results[0].Members {
		// memberEntry has no Token field — compile-time guarantee. Belt
		// and braces: marshal to JSON and verify "token" key is absent.
		raw, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		// Use distinctive token values that can't collide with ISO 8601
		// (which embeds 'T' as the date/time separator).
		if strings.Contains(string(raw), "\"token\"") ||
			strings.Contains(string(raw), "secret-token") {
			t.Errorf("token leaked in member JSON: %s", string(raw))
		}
	}
}

func TestBuildResultsResponse_NilWinner(t *testing.T) {
	in := []persistence.GameResult{
		{GameID: "g1", EndReason: game.EndHostForceEnd},
	}
	resp := buildResultsResponse(in)
	if resp.Results[0].Winner != nil {
		t.Errorf("winner should be nil")
	}
	raw, _ := json.Marshal(resp)
	if !strings.Contains(string(raw), `"winner":null`) {
		t.Errorf("winner not null in JSON: %s", string(raw))
	}
}

func TestResultsHandler_DBError(t *testing.T) {
	store := errorStore{}
	h := resultsHandler(store, slog.Default())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/results", nil)
	h.ServeHTTP(w, r)
	if w.Code != 500 {
		t.Errorf("status = %d", w.Code)
	}
}

type errorStore struct {
	dummyStore
}

func (errorStore) ListResults(context.Context, int) ([]persistence.GameResult, error) {
	return nil, context.DeadlineExceeded
}

func TestResultsHandler_LimitBoundary(t *testing.T) {
	store := newSeededStore(t, nil)
	h := resultsHandler(store, slog.Default())

	cases := []struct {
		query  string
		status int
	}{
		{"", 200},
		{"?limit=1", 200},
		{"?limit=500", 200},
		{"?limit=0", 400},
		{"?limit=-5", 400},
		{"?limit=501", 400},
		{"?limit=garbage", 400},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/results"+tc.query, nil)
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Errorf("status = %d, want %d", w.Code, tc.status)
			}
		})
	}
}
