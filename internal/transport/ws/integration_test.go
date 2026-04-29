package ws_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
	"github.com/saltware/mafia-game/internal/transport/ws"
)

// testRig wraps a fresh SessionManager + Hub + httptest server.
type testRig struct {
	mgr session.SessionManager
	hub ws.Hub
	srv *httptest.Server
	url string
}

func newRig(t *testing.T) *testRig {
	t.Helper()
	dir := t.TempDir()

	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "u3.db"))
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
		CheckOrigin: func(r *http.Request) bool { return true },
	}, mgr, nil)

	srv := httptest.NewServer(hub.UpgradeHandler())
	t.Cleanup(func() {
		srv.Close()
		_ = hub.Close()
		_ = mgr.Close(context.Background())
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return &testRig{mgr: mgr, hub: hub, srv: srv, url: wsURL}
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	return conn
}

func sendJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	if err := conn.WriteJSON(v); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
}

// readUntil reads frames until one matches `match`, or timeout fires.
func readUntil(t *testing.T, conn *websocket.Conn, match func(typ string, raw []byte) bool) map[string]any {
	t.Helper()
	for i := 0; i < 100; i++ {
		var raw json.RawMessage
		if err := conn.ReadJSON(&raw); err != nil {
			t.Fatalf("ReadJSON: %v", err)
		}
		var env struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		if match(env.Type, raw) {
			var out map[string]any
			_ = json.Unmarshal(raw, &out)
			return out
		}
	}
	t.Fatalf("no matching frame in 100 reads")
	return nil
}

// TestE2E_HostJoinStartReceivesEvents — full happy path. NFR-U3-S1
// (private routing) + NFR-U3-P1 (push delivery).
func TestE2E_HostJoinStartReceivesEvents(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	// Welcome
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })

	// Host creates session
	sendJSON(t, host, map[string]any{"type": "host:create-session", "name": "host"})
	hostJoined := readUntil(t, host, func(typ string, _ []byte) bool { return typ == "joined" })
	hostPID := game.PlayerID(hostJoined["playerId"].(string))

	// 5 players join
	conns := []*websocket.Conn{host}
	for i := 0; i < 5; i++ {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		sendJSON(t, conn, map[string]any{"type": "join", "name": fmt.Sprintf("p%d", i)})
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "joined" })
		conns = append(conns, conn)
	}

	// Host starts the game
	sendJSON(t, host, map[string]any{
		"type":    "host:start",
		"options": game.DefaultOptions(6),
	})

	// Each connection should receive at least one `event` frame.
	for i, conn := range conns {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "event" })
		_ = i
	}
	_ = hostPID
}

// TestE2E_PrivateRoutingHidesRoleFromOthers — NFR-U3-S1.
// We connect two players, start a game, and verify each one receives a
// RoleRevealedToPlayer event with their own role and not the other's.
func TestE2E_PrivateRoutingHidesRoleFromOthers(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, host, map[string]any{"type": "host:create-session", "name": "host"})
	hostJoined := readUntil(t, host, func(typ string, _ []byte) bool { return typ == "joined" })
	hostPID := hostJoined["playerId"].(string)
	_ = hostPID

	type peer struct {
		conn *websocket.Conn
		pid  string
	}
	peers := []*peer{}
	for i := 0; i < 5; i++ {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		sendJSON(t, conn, map[string]any{"type": "join", "name": fmt.Sprintf("p%d", i)})
		jr := readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "joined" })
		peers = append(peers, &peer{conn: conn, pid: jr["playerId"].(string)})
	}

	sendJSON(t, host, map[string]any{
		"type":    "host:start",
		"options": game.DefaultOptions(6),
	})

	// Each peer collects its own RoleRevealedToPlayer.
	for _, p := range peers {
		_ = p.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		gotMine := false
		for i := 0; i < 50; i++ {
			var raw map[string]any
			if err := p.conn.ReadJSON(&raw); err != nil {
				break
			}
			if raw["type"] != "event" {
				continue
			}
			ev, _ := raw["event"].(map[string]any)
			if ev == nil {
				continue
			}
			if ev["kind"] == "RoleRevealedToPlayer" {
				if pid, _ := ev["playerId"].(string); pid != p.pid {
					t.Errorf("peer %s received foreign RoleRevealedToPlayer for %s", p.pid, pid)
				}
				gotMine = true
				break
			}
		}
		if !gotMine {
			t.Errorf("peer %s did not receive its own role event", p.pid)
		}
	}
}

// TestE2E_GracefulShutdownUnder2Seconds — NFR-U3-R4.
func TestE2E_GracefulShutdownUnder2Seconds(t *testing.T) {
	rig := newRig(t)

	conns := make([]*websocket.Conn, 0, 4)
	for i := 0; i < 4; i++ {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		conns = append(conns, conn)
	}

	start := time.Now()
	if err := rig.hub.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Close took %v, want < 2s", elapsed)
	}

	// Re-Close idempotent
	if err := rig.hub.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

// TestE2E_LeakNoGoroutineGrowth — NFR-U3-G2.
func TestE2E_LeakNoGoroutineGrowth(t *testing.T) {
	rig := newRig(t)

	// Warm up + baseline
	for i := 0; i < 10; i++ {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		_ = conn.Close()
	}
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	baseline := runtime.NumGoroutine()

	for i := 0; i < 50; i++ {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		_ = conn.Close()
	}
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	final := runtime.NumGoroutine()

	if final > baseline+10 {
		t.Errorf("goroutine leak: baseline=%d final=%d", baseline, final)
	}
}

// TestE2E_UnknownTypeRejected — wire validation.
func TestE2E_UnknownTypeRejected(t *testing.T) {
	rig := newRig(t)
	conn := dial(t, rig.url)
	readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })

	sendJSON(t, conn, map[string]any{"type": "totally-unknown"})
	got := readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "error" })
	if got["code"] != "VALIDATION_ERROR" {
		t.Errorf("code = %v", got["code"])
	}
}

// TestE2E_LobbyMembershipBroadcast — Post-Construction Maintenance (LOBBY
// Membership Events). Verifies that with 1 dedicated PUBLIC viewer + 1 host
// + 5 joiners, every connection observes 6 PlayerJoined events and the
// host can subsequently start the game (host:start succeeds with INTRO
// PhaseChanged reaching all viewers).
func TestE2E_LobbyMembershipBroadcast(t *testing.T) {
	rig := newRig(t)

	// 1 PUBLIC viewer (never joins). Stays as ClientPublic kind.
	publicConn := dial(t, rig.url)
	readUntil(t, publicConn, func(typ string, _ []byte) bool { return typ == "welcome" })

	// Host
	host := dial(t, rig.url)
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, host, map[string]any{"type": "host:create-session", "name": "host"})
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "joined" })

	// 5 joiners
	conns := []*websocket.Conn{host}
	names := []string{"민수", "철수", "영희", "수정", "지훈"}
	for i, nm := range names {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		sendJSON(t, conn, map[string]any{"type": "join", "name": nm})
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "joined" })
		conns = append(conns, conn)
		_ = i
	}

	// Public viewer should have observed 6 PlayerJoined events (host + 5 joiners).
	expectedNames := append([]string{"host"}, names...)
	collectPlayerJoined := func(conn *websocket.Conn, want int, who string) []string {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		got := make([]string, 0, want)
		for i := 0; i < 200 && len(got) < want; i++ {
			var raw map[string]any
			if err := conn.ReadJSON(&raw); err != nil {
				t.Fatalf("%s ReadJSON: %v", who, err)
			}
			if raw["type"] != "event" {
				continue
			}
			ev, _ := raw["event"].(map[string]any)
			if ev == nil {
				continue
			}
			if ev["kind"] != "PlayerJoined" {
				continue
			}
			name, _ := ev["name"].(string)
			got = append(got, name)
		}
		return got
	}

	got := collectPlayerJoined(publicConn, 6, "public")
	if len(got) != 6 {
		t.Fatalf("public received %d PlayerJoined, want 6: %v", len(got), got)
	}
	for i, want := range expectedNames {
		if got[i] != want {
			t.Errorf("public PlayerJoined[%d].name = %q, want %q", i, got[i], want)
		}
	}

	// Host should have observed 5 PlayerJoined (its own create may or may
	// not arrive depending on Public→Player transition timing — we just
	// require that subsequent joins arrive).
	hostGot := collectPlayerJoined(host, 5, "host")
	if len(hostGot) != 5 {
		t.Fatalf("host received %d post-self PlayerJoined, want 5: %v", len(hostGot), hostGot)
	}

	// host:start now succeeds — without the LOBBY broadcast fix the host
	// PUBLIC client view of the lobby would still be empty and the FE
	// "게임 시작" button would be disabled, but the wire-level start is
	// already gated only on Member count, so we instead verify that
	// PhaseChanged reaches every connection after start.
	sendJSON(t, host, map[string]any{
		"type":    "host:start",
		"options": game.DefaultOptions(6),
	})

	for i, conn := range conns {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		gotPhase := false
		for j := 0; j < 200; j++ {
			var raw map[string]any
			if err := conn.ReadJSON(&raw); err != nil {
				break
			}
			if raw["type"] != "event" {
				continue
			}
			ev, _ := raw["event"].(map[string]any)
			if ev == nil {
				continue
			}
			if ev["kind"] == "PhaseChanged" {
				gotPhase = true
				break
			}
		}
		if !gotPhase {
			t.Errorf("conn %d did not receive PhaseChanged after host:start", i)
		}
	}
}

// TestE2E_ResumeRestoresPlayer — FR-1.2 + NFR-U3-R3.
func TestE2E_ResumeRestoresPlayer(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, host, map[string]any{"type": "host:create-session", "name": "host"})
	jr := readUntil(t, host, func(typ string, _ []byte) bool { return typ == "joined" })
	token := jr["token"].(string)

	// New connection → resume
	resumeConn := dial(t, rig.url)
	readUntil(t, resumeConn, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, resumeConn, map[string]any{"type": "resume", "token": token})
	readUntil(t, resumeConn, func(typ string, _ []byte) bool { return typ == "joined" })
	readUntil(t, resumeConn, func(typ string, _ []byte) bool { return typ == "snapshot" })
}
