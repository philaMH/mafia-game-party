package ws_test

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/game"
)

// readType drains messages until one of the given types arrives, or the
// connection's read deadline fires.
func readType(t *testing.T, conn *websocket.Conn, want ...string) map[string]any {
	t.Helper()
	allowed := make(map[string]bool, len(want))
	for _, s := range want {
		allowed[s] = true
	}
	return readUntil(t, conn, func(typ string, _ []byte) bool { return allowed[typ] })
}

func TestIter2_HostClaim_FirstSucceedsSecondRejected(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	defer host.Close()
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	tok := readType(t, host, "host-token", "room:host-occupied")
	if tok["type"] != "host-token" {
		t.Fatalf("first claim got %v, want host-token", tok["type"])
	}
	if s, _ := tok["token"].(string); s == "" {
		t.Errorf("host-token should carry non-empty token, got %v", tok)
	}

	other := dial(t, rig.url)
	defer other.Close()
	_ = readType(t, other, "welcome")
	sendJSON(t, other, map[string]any{"type": "host:claim"})
	resp := readType(t, other, "host-token", "room:host-occupied")
	if resp["type"] != "room:host-occupied" {
		t.Errorf("second claim got %v, want room:host-occupied", resp["type"])
	}
}

func TestIter2_HostOpenRoom_BroadcastsRoomOpened(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	defer host.Close()
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	_ = readType(t, host, "host-token")

	// Other client (player view), already connected before open-room.
	other := dial(t, rig.url)
	defer other.Close()
	_ = readType(t, other, "welcome")

	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	sendJSON(t, host, map[string]any{
		"type":    "host:open-room",
		"options": opts,
	})

	// Both clients should receive room:opened.
	gotHost := readType(t, host, "room:opened")
	if gotHost["type"] != "room:opened" {
		t.Errorf("host did not receive room:opened, got %v", gotHost)
	}
	_ = other.SetReadDeadline(time.Now().Add(2 * time.Second))
	gotOther := readType(t, other, "room:opened")
	if gotOther["type"] != "room:opened" {
		t.Errorf("other did not receive room:opened, got %v", gotOther)
	}
}

func TestIter2_HostReleaseOnDisconnect(t *testing.T) {
	rig := newRig(t)

	host1 := dial(t, rig.url)
	_ = readType(t, host1, "welcome")
	sendJSON(t, host1, map[string]any{"type": "host:claim"})
	resp := readType(t, host1, "host-token")
	if resp["type"] != "host-token" {
		t.Fatalf("first claim failed: %v", resp)
	}
	_ = host1.Close()

	// Wait briefly for the server to process the disconnect.
	time.Sleep(150 * time.Millisecond)

	host2 := dial(t, rig.url)
	defer host2.Close()
	_ = readType(t, host2, "welcome")
	sendJSON(t, host2, map[string]any{"type": "host:claim"})
	got := readType(t, host2, "host-token", "room:host-occupied")
	if got["type"] != "host-token" {
		t.Errorf("after disconnect, second host should get host-token; got %v", got["type"])
	}
}
