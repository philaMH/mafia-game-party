package httpx

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
	"github.com/saltware/mafia-game/internal/transport/ws"
)

// testRig owns the dependencies + the handler that mirrors what
// production wires up via httpx.New. Each test uses httptest.NewServer
// against this handler so we never touch real OS ports.
type testRig struct {
	store persistence.PersistenceStore
	mgr   session.SessionManager
	hub   ws.Hub
	srv   *httptest.Server
}

func newRig(t *testing.T) *testRig {
	t.Helper()
	dir := t.TempDir()
	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "u4.db"))
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	clock := &game.FakeClock{T: time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)}
	engine := game.New(game.NewAssigner(game.NewDefaultKeywordPool()), clock, rand.Reader)

	mgr, err := session.New(store, announce.NewDefaultCatalog(), engine, clock, rand.Reader,
		session.SessionOpts{TickInterval: time.Hour})
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}

	hub := ws.New(websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}, mgr, nil)

	cfg := Config{
		Hub:    hub,
		Store:  store,
		Assets: testAssets(),
		Logger: slog.New(slog.NewTextHandler(noopWriter{}, nil)),
	}
	mux := buildMux(cfg, cfg.Logger)
	handler := loggingMiddleware(cfg.Logger)(mux)

	srv := httptest.NewServer(handler)

	t.Cleanup(func() {
		srv.Close()
		_ = mgr.Close(context.Background())
		_ = hub.Close()
	})

	return &testRig{store: store, mgr: mgr, hub: hub, srv: srv}
}

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }

func testAssets() fs.FS {
	return fstest.MapFS{
		"index.html":          {Data: []byte("<html>spa</html>")},
		"assets/main-abc.js":  {Data: []byte("console.log(\"hi\")")},
		"assets/main-abc.css": {Data: []byte(".x{}")},
	}
}

func TestNew_RejectsMissingFields(t *testing.T) {
	cases := map[string]Config{
		"empty addr":     {Hub: dummyHub{}, Store: dummyStore{}, Assets: fstest.MapFS{}},
		"missing hub":    {Addr: "127.0.0.1:0", Store: dummyStore{}, Assets: fstest.MapFS{}},
		"missing store":  {Addr: "127.0.0.1:0", Hub: dummyHub{}, Assets: fstest.MapFS{}},
		"missing assets": {Addr: "127.0.0.1:0", Hub: dummyHub{}, Store: dummyStore{}},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(cfg); err == nil {
				t.Errorf("expected error for %s", name)
			}
		})
	}
}

func TestNew_NilLoggerFallsBackToDefault(t *testing.T) {
	cfg := Config{
		Addr:   "127.0.0.1:0",
		Hub:    dummyHub{},
		Store:  dummyStore{},
		Assets: fstest.MapFS{},
	}
	if _, err := New(cfg); err != nil {
		t.Errorf("expected nil-logger to be tolerated, got %v", err)
	}
}

func TestNew_ConstructsServerWithRoutes(t *testing.T) {
	cfg := Config{
		Addr:   "127.0.0.1:0",
		Hub:    dummyHub{},
		Store:  dummyStore{},
		Assets: testAssets(),
	}
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if srv == nil {
		t.Fatal("nil server")
	}
}

// dummy implementations for missing-field / configuration tests.
type dummyHub struct{}

func (dummyHub) Register(*websocket.Conn) (ws.ClientID, error) { return "", nil }
func (dummyHub) Unregister(ws.ClientID)                        {}
func (dummyHub) Run(context.Context) error                     { return nil }
func (dummyHub) Close() error                                  { return nil }
func (dummyHub) UpgradeHandler() http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {}
}

type dummyStore struct{}

func (dummyStore) SaveSnapshot(context.Context, persistence.Snapshot) error { return nil }
func (dummyStore) LoadActiveSnapshot(context.Context) (persistence.Snapshot, bool, error) {
	return persistence.Snapshot{}, false, nil
}
func (dummyStore) DeleteActiveSnapshot(context.Context) error                             { return nil }
func (dummyStore) SaveResultAndClearActive(context.Context, persistence.GameResult) error { return nil }
func (dummyStore) ListResults(context.Context, int) ([]persistence.GameResult, error) {
	return nil, nil
}
func (dummyStore) AppendEvent(context.Context, string, game.EventEnvelope) error { return nil }
func (dummyStore) ArchiveCorrupt(context.Context) error                          { return nil }
func (dummyStore) Close() error                                                  { return nil }

// --- routing behaviour ---

func TestServer_Healthz(t *testing.T) {
	rig := newRig(t)
	resp, err := http.Get(rig.srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	body := readAll(t, resp)
	if string(body) != "ok" {
		t.Errorf("body = %q", string(body))
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %q", ct)
	}
}

func TestServer_AssetsImmutableCache(t *testing.T) {
	rig := newRig(t)
	resp, err := http.Get(rig.srv.URL + "/assets/main-abc.js")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	cc := resp.Header.Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Errorf("missing immutable: %q", cc)
	}
}

func TestServer_SPAFallback(t *testing.T) {
	rig := newRig(t)
	for _, path := range []string{"/", "/play", "/play/some/route", "/public"} {
		resp, err := http.Get(rig.srv.URL + path)
		if err != nil {
			t.Fatalf("Get %s: %v", path, err)
		}
		body := readAll(t, resp)
		_ = resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("path %s status = %d", path, resp.StatusCode)
		}
		if !strings.Contains(string(body), "spa") {
			t.Errorf("path %s body = %q", path, string(body))
		}
		if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("path %s cache-control = %q", path, cc)
		}
	}
}

func TestServer_APIResultsEmpty(t *testing.T) {
	rig := newRig(t)
	resp, err := http.Get(rig.srv.URL + "/api/results")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := got["results"]; !ok {
		t.Error("missing results key")
	}
}

func TestServer_APIResultsLimitValidation(t *testing.T) {
	rig := newRig(t)
	cases := []struct {
		limit  string
		status int
	}{
		{"", 200},
		{"1", 200},
		{"500", 200},
		{"0", 400},
		{"-1", 400},
		{"501", 400},
		{"abc", 400},
	}
	for _, tc := range cases {
		t.Run(tc.limit, func(t *testing.T) {
			u := rig.srv.URL + "/api/results"
			if tc.limit != "" {
				u += "?limit=" + tc.limit
			}
			resp, err := http.Get(u)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Errorf("limit=%q status = %d, want %d", tc.limit, resp.StatusCode, tc.status)
			}
		})
	}
}

func TestServer_APIResultsOmitsMemberToken(t *testing.T) {
	rig := newRig(t)

	winner := game.TeamCitizen
	r := persistence.GameResult{
		GameID:    "g-test",
		StartedAt: time.Now().UTC(),
		EndedAt:   time.Now().UTC(),
		Winner:    &winner,
		EndReason: game.EndCitizenWin,
		Options:   game.DefaultOptions(6),
		Members: []persistence.PersistedMember{
			{ID: "p1", Name: "철수", Token: "SECRET-TOKEN-XYZ", JoinedAt: time.Now().UTC()},
		},
		Reveal: []game.Player{{ID: "p1", Name: "철수", Role: game.RoleMafia}},
	}
	if err := rig.store.SaveResultAndClearActive(context.Background(), r); err != nil {
		t.Fatalf("seed: %v", err)
	}

	resp, err := http.Get(rig.srv.URL + "/api/results")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	body := readAll(t, resp)
	if strings.Contains(string(body), "SECRET-TOKEN-XYZ") {
		t.Errorf("token leaked into /api/results response: %s", string(body))
	}
	if strings.Contains(string(body), "\"token\"") {
		t.Errorf("token field present in JSON: %s", string(body))
	}
	if !strings.Contains(string(body), "철수") {
		t.Errorf("expected member name, got: %s", string(body))
	}
}

func readAll(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	const maxBody = 1 << 20
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > maxBody {
				t.Fatal("response body too large")
			}
		}
		if err != nil {
			break
		}
	}
	return buf
}
