package ws_test

import (
	"strings"
	"testing"
	"time"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

// waitForSavedHostOptions polls SessionManager.SavedHostOptions for up to
// 1 second so tests can synchronize with the asynchronous WS dispatch
// goroutine.
func waitForSavedHostOptions(t *testing.T, mgr session.SessionManager) (game.Options, bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got, ok := mgr.SavedHostOptions(); ok {
			return got, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return game.Options{}, false
}

// validHostOptions returns an Options value that satisfies the U2 shape
// validator (mirrors the front-end's defaultOptions(8)).
func validHostOptions() game.Options {
	return game.Options{
		MafiaCount:            2,
		MaxPlayers:            8,
		IntroSecondsPerPlayer: 20,
		DiscussionSeconds:     180,
		NightMafiaSeconds:     30,
		NightPoliceSeconds:    10,
		NightDoctorSeconds:    10,
		DoctorSelfHealAllowed: true,
		AnnouncementVoiceOn:   true,
	}
}

// TestIter7_HostSaveOptions_HappyPath verifies that a valid host:save-options
// frame from the GM-seat holder is forwarded to U2 and the saved value is
// reflected in the SessionManager state.
func TestIter7_HostSaveOptions_HappyPath(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	defer host.Close()
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	_ = readType(t, host, "host-token")

	opts := validHostOptions()
	opts.DiscussionSeconds = 240 // distinguishable value
	sendJSON(t, host, map[string]any{
		"type":    "host:save-options",
		"options": opts,
	})

	// Server emits no outgoing on success; the WS readLoop dispatches
	// asynchronously, so poll until the saved value lands.
	got, ok := waitForSavedHostOptions(t, rig.mgr)
	if !ok {
		t.Fatalf("hasSavedHostOptions=false after host:save-options happy path")
	}
	if got.DiscussionSeconds != 240 {
		t.Errorf("DiscussionSeconds=%d, want 240", got.DiscussionSeconds)
	}
}

// TestIter7_HostSaveOptions_NonHost confirms that a client without the
// GM-seat token receives a permission error.
func TestIter7_HostSaveOptions_NonHost(t *testing.T) {
	rig := newRig(t)

	stranger := dial(t, rig.url)
	defer stranger.Close()
	_ = readType(t, stranger, "welcome")

	sendJSON(t, stranger, map[string]any{
		"type":    "host:save-options",
		"options": validHostOptions(),
	})

	frame := readType(t, stranger, "error", "announce")
	// announce may precede error; loop one more time if needed.
	if frame["type"] == "announce" {
		frame = readType(t, stranger, "error")
	}
	if frame["type"] != "error" {
		t.Fatalf("got %v, want error frame", frame["type"])
	}
	code, _ := frame["code"].(string)
	if !strings.Contains(strings.ToUpper(code), "PERMISSION") {
		t.Errorf("error code=%q, want PERMISSION_DENIED", code)
	}
	if _, ok := rig.mgr.SavedHostOptions(); ok {
		t.Errorf("non-host save should not flip hasSavedHostOptions")
	}
}

// TestIter7_HostSaveOptions_Validation confirms that a host claim followed
// by a malformed options shape is rejected by the U2 validator.
func TestIter7_HostSaveOptions_Validation(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	defer host.Close()
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	_ = readType(t, host, "host-token")

	bad := validHostOptions()
	bad.MaxPlayers = 5 // out of [6,12]
	sendJSON(t, host, map[string]any{
		"type":    "host:save-options",
		"options": bad,
	})

	frame := readType(t, host, "error", "announce")
	if frame["type"] == "announce" {
		frame = readType(t, host, "error")
	}
	if frame["type"] != "error" {
		t.Fatalf("got %v, want error frame", frame["type"])
	}
	code, _ := frame["code"].(string)
	if !strings.Contains(strings.ToUpper(code), "VALID") {
		t.Errorf("error code=%q, want a *VALIDATION* code", code)
	}
	if _, ok := rig.mgr.SavedHostOptions(); ok {
		t.Errorf("invalid save should not flip hasSavedHostOptions")
	}
}

// TestIter7_HostSaveOptions_BadJSON sends a payload whose `options` field
// is the wrong JSON shape; the dispatcher must respond with a
// VALIDATION_ERROR error frame.
func TestIter7_HostSaveOptions_BadJSON(t *testing.T) {
	rig := newRig(t)

	host := dial(t, rig.url)
	defer host.Close()
	_ = readType(t, host, "welcome")
	sendJSON(t, host, map[string]any{"type": "host:claim"})
	_ = readType(t, host, "host-token")

	// Send a hand-crafted frame so `options` is a string instead of an
	// object — json.Unmarshal into hostSaveOptionsPayload will fail.
	if err := host.WriteMessage(1 /* TextMessage */, []byte(`{"type":"host:save-options","options":"oops"}`)); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	frame := readType(t, host, "error")
	if frame["type"] != "error" {
		t.Fatalf("got %v, want error frame", frame["type"])
	}
	code, _ := frame["code"].(string)
	if code != "VALIDATION_ERROR" {
		t.Errorf("error code=%q, want VALIDATION_ERROR", code)
	}
}
