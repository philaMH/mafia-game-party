package ws

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/saltware/mafia-game/internal/game"
)

// TestIter5_NightStepChangedCarriesDeadline confirms the Iteration 5 wire
// addition (`stepDeadlineMs`) is populated and serialized correctly so
// the public timer bar can render a synchronized countdown.
func TestIter5_NightStepChangedCarriesDeadline(t *testing.T) {
	deadline := time.Date(2026, 4, 29, 13, 0, 30, 0, time.UTC)
	p := buildEventPayload(game.NightStepChanged{
		Step:     game.NightStepPolice,
		Day:      2,
		Deadline: deadline,
	})
	if p.Kind != "NightStepChanged" {
		t.Errorf("Kind=%q", p.Kind)
	}
	if p.Step != game.NightStepPolice {
		t.Errorf("Step=%q", p.Step)
	}
	if p.StepDeadlineMs != deadline.UnixMilli() {
		t.Errorf("StepDeadlineMs=%d, want %d", p.StepDeadlineMs, deadline.UnixMilli())
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(bytes), `"stepDeadlineMs":`) {
		t.Errorf("wire payload missing stepDeadlineMs field: %s", string(bytes))
	}
}

// TestIter5_GamePausedSerialization checks the GamePaused payload only
// carries the Phase and Kind, while GameResumed carries an optional
// shifted-forward deadline.
func TestIter5_GamePausedSerialization(t *testing.T) {
	p := buildEventPayload(game.GamePaused{Phase: game.PhaseNight})
	if p.Kind != "GamePaused" {
		t.Errorf("Kind=%q", p.Kind)
	}
	if p.Phase != game.PhaseNight {
		t.Errorf("Phase=%q", p.Phase)
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(bytes), `"kind":"GamePaused"`) {
		t.Errorf("missing GamePaused kind in %s", string(bytes))
	}
	if !strings.Contains(string(bytes), `"phase":"NIGHT"`) {
		t.Errorf("missing phase=NIGHT in %s", string(bytes))
	}
}

func TestIter5_GameResumedSerialization(t *testing.T) {
	deadline := time.Date(2026, 4, 29, 13, 5, 0, 0, time.UTC)
	p := buildEventPayload(game.GameResumed{Phase: game.PhaseDay, Deadline: deadline})
	if p.Kind != "GameResumed" {
		t.Errorf("Kind=%q", p.Kind)
	}
	if p.DeadlineMs != deadline.UnixMilli() {
		t.Errorf("DeadlineMs=%d, want %d", p.DeadlineMs, deadline.UnixMilli())
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(bytes), `"deadlineMs":`) {
		t.Errorf("missing deadlineMs in %s", string(bytes))
	}
}

// TestIter5_HostPauseResumeWireConstants confirms the new outgoing types
// are stable wire identifiers.
func TestIter5_HostPauseResumeWireConstants(t *testing.T) {
	if TypeHostPause != "host:pause" {
		t.Errorf("TypeHostPause=%q, want host:pause", TypeHostPause)
	}
	if TypeHostResume != "host:resume" {
		t.Errorf("TypeHostResume=%q, want host:resume", TypeHostResume)
	}
}
