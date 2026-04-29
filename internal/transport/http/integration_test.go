package httpx

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
	"github.com/saltware/mafia-game/internal/transport/ws"
)

// TestIntegration_ListenAndShutdown spins up the actual httpx.Server on
// a loopback listener, hits /healthz, then verifies graceful Shutdown
// completes within the 5s budget (NFR-U4-R1 fragment).
func TestIntegration_ListenAndShutdown(t *testing.T) {
	dir := t.TempDir()
	store, err := persistence.OpenSqlite(context.Background(), filepath.Join(dir, "u4.db"))
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	clock := &game.FakeClock{T: time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)}
	engine := game.New(game.NewAssigner(game.NewDefaultKeywordPool()), clock, rand.Reader)
	mgr, err := session.New(store, announce.NewDefaultCatalog(), engine, clock, rand.Reader,
		session.SessionOpts{TickInterval: time.Hour})
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })

	hub := ws.New(websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}, mgr, nil)
	t.Cleanup(func() { _ = hub.Close() })

	// Find a free port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	srv, err := New(Config{
		Addr:   addr,
		Hub:    hub,
		Store:  store,
		Assets: testAssets(),
		Logger: slog.New(slog.NewTextHandler(noopWriter{}, nil)),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	// Wait briefly for ListenAndServe to bind.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Hit /healthz to confirm the server is live.
	resp, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}

	// Now Shutdown — must finish within 5s.
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Shutdown took %v", elapsed)
	}

	// ListenAndServe goroutine should have returned http.ErrServerClosed.
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("ListenAndServe err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("ListenAndServe did not return after Shutdown")
	}
}
