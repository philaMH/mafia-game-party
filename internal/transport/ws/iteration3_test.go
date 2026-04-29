package ws_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/game"
)

// readNextType reads the next frame and returns its type + raw map. Used
// when we need to assert exact send order rather than skipping until a
// matching type appears (which is what readUntil/readType do).
func readNextType(t *testing.T, conn *websocket.Conn) (string, map[string]any) {
	t.Helper()
	var raw json.RawMessage
	if err := conn.ReadJSON(&raw); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	var env struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return env.Type, out
}

// expectNoFrameWithin asserts that no further frame arrives within d. Used
// to verify that the server did NOT push extra messages after welcome
// when the room is closed.
func expectNoFrameWithin(t *testing.T, conn *websocket.Conn, d time.Duration) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(d))
	var raw json.RawMessage
	err := conn.ReadJSON(&raw)
	if err == nil {
		t.Fatalf("expected no frame within %v, got %s", d, string(raw))
	}
	// Reset deadline for any later reads.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
}

func TestIter3_Register_BeforeOpenRoom_NoExtraMessages(t *testing.T) {
	rig := newRig(t)
	conn := dial(t, rig.url)

	typ, _ := readNextType(t, conn)
	if typ != "welcome" {
		t.Fatalf("first frame = %q, want welcome", typ)
	}
	expectNoFrameWithin(t, conn, 200*time.Millisecond)
}

func TestIter3_Register_AfterClaimBeforeOpen_PushesHostOccupied(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	if typ, _ := readNextType(t, host); typ != "welcome" {
		t.Fatalf("host first frame not welcome")
	}
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	if typ, _ := readNextType(t, host); typ != "host-token" {
		t.Fatalf("host did not receive host-token")
	}

	other := dial(t, rig.url)
	typ, _ := readNextType(t, other)
	if typ != "welcome" {
		t.Fatalf("other first frame = %q, want welcome", typ)
	}
	typ, _ = readNextType(t, other)
	if typ != "room:host-occupied" {
		t.Fatalf("late-joiner did not get room:host-occupied; got %q", typ)
	}
	expectNoFrameWithin(t, other, 200*time.Millisecond)
}

func TestIter3_Register_AfterOpenRoom_PushesRoomOpened(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	_ = readType(t, host, "host-token")
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	sendJSON(t, host, map[string]any{"type": "host:open-room", "options": opts})
	_ = readType(t, host, "room:opened")

	other := dial(t, rig.url)
	if typ, _ := readNextType(t, other); typ != "welcome" {
		t.Fatalf("other first frame not welcome")
	}
	typ, payload := readNextType(t, other)
	if typ != "room:opened" {
		t.Fatalf("late-joiner did not get room:opened; got %q", typ)
	}
	gotOpts, _ := payload["options"].(map[string]any)
	if gotOpts == nil {
		t.Fatalf("room:opened missing options: %v", payload)
	}
	if int(gotOpts["mafiaCount"].(float64)) != opts.MafiaCount {
		t.Errorf("late-joiner room:opened mafiaCount=%v, want %d", gotOpts["mafiaCount"], opts.MafiaCount)
	}
	// LOBBY snapshot is pushed alongside room:opened so a refreshing
	// host sees the (empty) roster and start button.
	typ, snapPayload := readNextType(t, other)
	if typ != "snapshot" {
		t.Fatalf("expected snapshot after room:opened; got %q", typ)
	}
	st, _ := snapPayload["state"].(map[string]any)
	if st == nil {
		t.Fatalf("snapshot missing state: %v", snapPayload)
	}
	if st["phase"] != "LOBBY" {
		t.Errorf("snapshot.state.phase = %v, want LOBBY", st["phase"])
	}
	typ, _ = readNextType(t, other)
	if typ != "room:host-occupied" {
		t.Fatalf("expected room:host-occupied after snapshot; got %q", typ)
	}
	expectNoFrameWithin(t, other, 200*time.Millisecond)
}

func TestIter3_Register_AfterHostStartGame_PushesSnapshot(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	_ = readType(t, host, "host-token")
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	sendJSON(t, host, map[string]any{"type": "host:open-room", "options": opts})
	_ = readType(t, host, "room:opened")

	// Six players join so the host can start.
	for i := range 6 {
		conn := dial(t, rig.url)
		_ = readType(t, conn, "welcome")
		_ = readType(t, conn, "room:opened")
		_ = readType(t, conn, "room:host-occupied")
		sendJSON(t, conn, map[string]any{"type": "join", "name": fmt.Sprintf("p%d", i)})
		_ = readType(t, conn, "joined")
	}

	// Drain until host start can fire — host:start-room dispatches events.
	sendJSON(t, host, map[string]any{"type": "host:start-room"})
	deadline := time.Now().Add(3 * time.Second)
	gotIntro := false
	for time.Now().Before(deadline) && !gotIntro {
		_ = host.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var raw map[string]any
		if err := host.ReadJSON(&raw); err != nil {
			break
		}
		if raw["type"] == "event" {
			if ev, _ := raw["event"].(map[string]any); ev != nil && ev["kind"] == "PhaseChanged" && ev["phase"] == "INTRO" {
				gotIntro = true
			}
		}
	}
	if !gotIntro {
		t.Fatalf("host did not observe INTRO PhaseChanged before deadline")
	}

	// Now register a fresh client; expect welcome + room:opened + snapshot
	// + room:host-occupied. State.Phase must be INTRO.
	other := dial(t, rig.url)
	if typ, _ := readNextType(t, other); typ != "welcome" {
		t.Fatalf("other first frame not welcome")
	}
	if typ, _ := readNextType(t, other); typ != "room:opened" {
		t.Fatalf("expected room:opened, got %q", typ)
	}
	typ, payload := readNextType(t, other)
	if typ != "snapshot" {
		t.Fatalf("expected snapshot, got %q", typ)
	}
	state, _ := payload["state"].(map[string]any)
	if state == nil {
		t.Fatalf("snapshot missing state field: %v", payload)
	}
	if state["phase"] != "INTRO" {
		t.Errorf("snapshot.state.phase = %v, want INTRO", state["phase"])
	}
	your, _ := payload["your"].(map[string]any)
	if your == nil {
		t.Fatalf("snapshot missing your field")
	}
	// All yourInfo fields use omitempty, so an empty struct serializes to {}.
	// A late-joiner must NOT see role/keyword/team/mafiaCohort populated.
	for _, k := range []string{"role", "keyword", "team", "mafiaCohort"} {
		if v, ok := your[k]; ok && v != "" && v != nil {
			t.Errorf("late-joiner snapshot.your.%s = %v, want empty/absent", k, v)
		}
	}
	if payload["isHost"] != false {
		t.Errorf("late-joiner snapshot.isHost = %v, want false", payload["isHost"])
	}
	if typ, _ := readNextType(t, other); typ != "room:host-occupied" {
		t.Fatalf("expected room:host-occupied after snapshot, got %q", typ)
	}
}

func TestIter3_Register_PushOrder(t *testing.T) {
	rig := newRig(t)
	ctx := context.Background()

	// Drive state at the manager level (faster + deterministic) instead of
	// going through wire — but rig only exposes mgr, which is enough.
	tok, err := rig.mgr.ClaimHost(ctx)
	if err != nil {
		t.Fatalf("ClaimHost: %v", err)
	}
	opts := game.DefaultOptions(8)
	opts.MaxPlayers = 8
	if _, err := rig.mgr.OpenRoom(ctx, tok, opts); err != nil {
		t.Fatalf("OpenRoom: %v", err)
	}
	for i := range 6 {
		if _, err := rig.mgr.JoinPlayer(ctx, fmt.Sprintf("p%d", i)); err != nil {
			t.Fatalf("JoinPlayer #%d: %v", i, err)
		}
	}
	if _, err := rig.mgr.HostStartGame(ctx, tok); err != nil {
		t.Fatalf("HostStartGame: %v", err)
	}

	conn := dial(t, rig.url)
	expected := []string{"welcome", "room:opened", "snapshot", "room:host-occupied"}
	for _, want := range expected {
		got, _ := readNextType(t, conn)
		if got != want {
			t.Fatalf("frame order mismatch: got %q, want %q (sequence so far: %v)", got, want, expected)
		}
	}
}
