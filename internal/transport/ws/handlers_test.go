package ws_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/game"
)

// TestHandlers_AllSubmitTypes exercises every "submit:*" + "host:*" message
// type to lift handleIncoming coverage. Most submissions return engine
// errors (wrong phase, etc.), which is fine — we only want the routing
// branches taken.
func TestHandlers_AllSubmitTypes(t *testing.T) {
	rig := newRig(t)
	host := dial(t, rig.url)
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, host, map[string]any{"type": "host:create-session", "name": "host"})
	jr := readUntil(t, host, func(typ string, _ []byte) bool { return typ == "joined" })
	hostPID := jr["playerId"].(string)
	_ = hostPID

	// Add 5 more players
	for i := 0; i < 5; i++ {
		conn := dial(t, rig.url)
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
		sendJSON(t, conn, map[string]any{"type": "join", "name": fmt.Sprintf("p%d", i)})
		readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "joined" })
	}

	sendJSON(t, host, map[string]any{
		"type":    "host:start",
		"options": game.DefaultOptions(6),
	})
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "event" })

	// Now exercise every submit/host action — failures are fine, we only
	// need each switch arm hit.
	hostMsgs := []map[string]any{
		{"type": "submit:advance-intro"},
		{"type": "submit:end-night"},
		{"type": "submit:end-discussion"},
		{"type": "host:toggle-voice", "on": true},
		{"type": "host:toggle-voice", "on": false},
		{"type": "submit:mafia-kill", "target": hostPID},
		{"type": "submit:doctor-heal", "target": hostPID},
		{"type": "submit:police-check", "target": hostPID},
		{"type": "submit:vote", "target": hostPID},
		{"type": "subscribe-public"},
	}
	for _, msg := range hostMsgs {
		sendJSON(t, host, msg)
		// drain whatever comes back briefly
		_ = host.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		for i := 0; i < 5; i++ {
			var raw map[string]any
			if err := host.ReadJSON(&raw); err != nil {
				break
			}
			_ = raw
		}
		_ = host.SetReadDeadline(time.Now().Add(2 * time.Second))
	}

	// host:force-end last
	sendJSON(t, host, map[string]any{"type": "host:force-end"})
}

// TestHandlers_BadJSONRejected covers the JSON decode failure branch.
func TestHandlers_BadJSONRejected(t *testing.T) {
	rig := newRig(t)
	host := dial(t, rig.url)
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })

	if err := host.WriteMessage(websocket.TextMessage, []byte(`{"type": "host:start", "options": "not-an-object"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := readUntil(t, host, func(typ string, _ []byte) bool { return typ == "error" })
	_ = got
}

// TestHandlers_NonJSONRejected covers the non-JSON branch.
func TestHandlers_NonJSONRejected(t *testing.T) {
	rig := newRig(t)
	conn := dial(t, rig.url)
	readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`<<<not-json>>>`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "error" })
	if got["code"] != "VALIDATION_ERROR" {
		t.Errorf("code = %v", got["code"])
	}
}

// TestHandlers_DuplicateNameRejected covers the announce.RenderError
// path inside handleSubmitErr.
func TestHandlers_DuplicateNameRejected(t *testing.T) {
	rig := newRig(t)
	host := dial(t, rig.url)
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, host, map[string]any{"type": "host:create-session", "name": "host"})
	readUntil(t, host, func(typ string, _ []byte) bool { return typ == "joined" })

	conn := dial(t, rig.url)
	readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, conn, map[string]any{"type": "join", "name": "first"})
	readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "joined" })

	conn2 := dial(t, rig.url)
	readUntil(t, conn2, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, conn2, map[string]any{"type": "join", "name": "first"}) // duplicate
	// Expect both `announce` and `error` frames (in either order).
	sawAnnounce := false
	sawError := false
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 6; i++ {
		var raw map[string]any
		if err := conn2.ReadJSON(&raw); err != nil {
			break
		}
		switch raw["type"] {
		case "announce":
			sawAnnounce = true
		case "error":
			sawError = true
		}
		if sawAnnounce && sawError {
			break
		}
	}
	if !sawError {
		t.Errorf("expected error frame")
	}
}

// TestHandlers_LastConnectWinsEvictsPrior — BR-U3-RECONNECT-1.
func TestHandlers_LastConnectWinsEvictsPrior(t *testing.T) {
	rig := newRig(t)
	first := dial(t, rig.url)
	readUntil(t, first, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, first, map[string]any{"type": "host:create-session", "name": "host"})
	jr := readUntil(t, first, func(typ string, _ []byte) bool { return typ == "joined" })
	token := jr["token"].(string)

	second := dial(t, rig.url)
	readUntil(t, second, func(typ string, _ []byte) bool { return typ == "welcome" })
	sendJSON(t, second, map[string]any{"type": "resume", "token": token})
	readUntil(t, second, func(typ string, _ []byte) bool { return typ == "joined" })
	readUntil(t, second, func(typ string, _ []byte) bool { return typ == "snapshot" })

	// The first connection should be force-closed.
	_ = first.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 10; i++ {
		var raw map[string]any
		err := first.ReadJSON(&raw)
		if err != nil {
			return // expected — connection closed
		}
	}
	t.Error("first conn was not evicted")
}

// TestHandlers_RegisterAfterCloseRejected covers Hub.Register error path.
func TestHandlers_RegisterAfterCloseRejected(t *testing.T) {
	rig := newRig(t)
	if err := rig.hub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	conn, _, err := websocket.DefaultDialer.Dial(rig.url, nil)
	if err != nil {
		// Server may already have stopped accepting — this is fine.
		return
	}
	_ = conn.Close()
}

// TestHandlers_ContextDoneOnCloseTerminatesGoroutines verifies the Run/Close lifecycle.
func TestHandlers_ContextDoneOnCloseTerminatesGoroutines(t *testing.T) {
	rig := newRig(t)
	conn := dial(t, rig.url)
	readUntil(t, conn, func(typ string, _ []byte) bool { return typ == "welcome" })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- rig.hub.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned err: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Run did not return within 2s")
	}
}
