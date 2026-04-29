package game

import (
	"encoding/json"
	"testing"
	"time"
)

// Iteration 9 — Fix · 최종 결과 발표 → 승리 화면 전환.
// VOTE/NIGHT 결판 직후 GameEnded emission 을 defaultFinalResultBufferSeconds
// 만큼 지연시키는 흐름의 회귀 테스트. 자세한 설계는
// `aidlc-docs/construction/u1-game-core/functional-design/iteration9-patch.md`
// 참고.

// runToVoteEndCitizenWin advances a 6-player game (1 mafia + 5 citizen-side)
// to a vote-end where all surviving citizens vote out the lone mafia,
// triggering CITIZEN_WIN through applyElimination. Returns the engine, fake
// clock, and the events emitted by the final SubmitVote (which contains the
// tally-driven Eliminated batch but NOT GameEnded — that one is deferred).
func runToVoteEndCitizenWin(t *testing.T, seed int64) (Engine, *FakeClock, []EventEnvelope) {
	t.Helper()
	e, clock := newTestEngine(t, seed)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	advanceToNight(t, e)

	// Night 1: skip the kill via EndNightEarly so all 6 stay alive — Day 2
	// will then vote out the mafia and trigger CITIZEN_WIN.
	if _, _, err := e.Apply(EndNightEarly{HostID: "p1"}); err != nil {
		t.Fatalf("EndNightEarly: %v", err)
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatalf("EndDiscussionEarly Day 2: %v", err)
	}
	state := e.Snapshot()
	if state.Phase != PhaseVote {
		t.Fatalf("expected VOTE on Day 2, got %s", state.Phase)
	}
	mafia, ok := findRole(state, RoleMafia)
	if !ok {
		t.Fatalf("no mafia in roster")
	}

	var lastEvs []EventEnvelope
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		_, evs, err := e.Apply(SubmitVote{Voter: p.ID, Target: mafia})
		if err != nil {
			t.Fatalf("SubmitVote: %v", err)
		}
		lastEvs = evs
	}
	return e, clock, lastEvs
}

// runToNightEndMafiaWin advances a 6-player / 2-mafia game so that Night 2's
// resolveNight closes the game with MAFIA_WIN. Returns the engine and clock.
// At return the engine is in PhaseDay with State.PendingGameEnd set; the
// final GameEnded has not yet been fired.
func runToNightEndMafiaWin(t *testing.T, seed int64) (Engine, *FakeClock) {
	t.Helper()
	e, clock := newTestEngine(t, seed)
	opts := DefaultOptions(6)
	opts.MafiaCount = 2 // 2 mafia + 1 doctor + 1 police + 2 citizens
	mustStart(t, e, playerSet(6), "p1", opts)

	// INTRO -> Day 1 -> abstain vote -> NIGHT 1 (MAFIA step).
	state := advanceToNight(t, e)
	mafiaIDs, _, _, _ := allRoles(state)
	if len(mafiaIDs) != 2 {
		t.Fatalf("want 2 mafia, got %d", len(mafiaIDs))
	}
	rep := state.MafiaRepresentativeID

	// Night 1 kill: pick any living non-mafia.
	target := firstLivingNonMafia(state)
	if target == "" {
		t.Fatalf("Night 1: no kill target found")
	}
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: rep, Target: target}); err != nil {
		t.Fatalf("Night1 SubmitMafiaKill: %v", err)
	}
	advanceToDay(t, e, clock)

	// Day 2: 2M + 3C. Abstain so no elimination occurs; we need Night 2 to
	// be the closing event (MAFIA_WIN at resolveNight).
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: "p1"}); err != nil {
		t.Fatalf("Day2 EndDiscussionEarly: %v", err)
	}
	state = e.Snapshot()
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		if _, _, err := e.Apply(SubmitVote{Voter: p.ID, Target: ""}); err != nil {
			t.Fatalf("Day2 SubmitVote abstain: %v", err)
		}
	}
	state = e.Snapshot()
	if state.Phase != PhaseNight {
		t.Fatalf("expected NIGHT after Day 2 abstain, got %s", state.Phase)
	}
	if state.PendingGameEnd != nil {
		t.Fatalf("Day 2 abstain unexpectedly scheduled an end: %+v", state.PendingGameEnd)
	}
	drainNightIntro(t, e)

	// Night 2 kill: pick any living non-mafia (should leave 2M + 2C → MAFIA_WIN).
	state = e.Snapshot()
	rep2 := state.MafiaRepresentativeID
	target2 := firstLivingNonMafia(state)
	if target2 == "" {
		t.Fatalf("Night 2: no kill target found")
	}
	if _, _, err := e.Apply(SubmitMafiaKill{Mafia: rep2, Target: target2}); err != nil {
		t.Fatalf("Night2 SubmitMafiaKill: %v", err)
	}
	advanceToDay(t, e, clock)

	state = e.Snapshot()
	if state.Phase != PhaseDay {
		t.Fatalf("expected DAY 3 after Night 2 resolve, got %s", state.Phase)
	}
	if state.PendingGameEnd == nil {
		t.Fatalf("expected PendingGameEnd set after night-end MAFIA_WIN, got nil (mafia=%d citizens=%d)",
			state.LiveMafiaCount(), state.LiveCitizenSideCount())
	}
	return e, clock
}

// firstLivingNonMafia returns the first living non-mafia player id, or "".
func firstLivingNonMafia(s State) PlayerID {
	for _, p := range s.Players {
		if p.Alive && p.Role != RoleMafia {
			return p.ID
		}
	}
	return ""
}

// advanceToDay drains a NIGHT phase by ticking past each NightStep deadline
// until resolveNight has fired (Phase becomes DAY/END). Caps at 5 ticks
// to surface infinite loops.
func advanceToDay(t *testing.T, e Engine, clock *FakeClock) {
	t.Helper()
	for range 5 {
		state := e.Snapshot()
		if state.Phase != PhaseNight {
			return
		}
		if state.NightStepDeadline.IsZero() {
			t.Fatalf("advanceToDay: NightStepDeadline=zero (step=%q)", state.NightStep)
		}
		clock.T = state.NightStepDeadline.Add(time.Millisecond)
		if _, _, err := e.Tick(clock.Now()); err != nil {
			t.Fatalf("advanceToDay Tick: %v", err)
		}
	}
	t.Fatalf("advanceToDay: did not exit NIGHT after 5 ticks")
}

// I9-T1 — Vote-end: tally directly schedules GameEnded and does NOT emit
// it inside the same batch. Phase remains VOTE, PendingGameEnd is set with
// Deadline = now + 5s.
func TestI9_VoteEndSchedulesPendingGameEnd(t *testing.T) {
	e, clock, evs := runToVoteEndCitizenWin(t, 9001)

	// GameEnded must NOT be in the tally event batch.
	for _, ev := range evs {
		if _, ok := ev.Event.(GameEnded); ok {
			t.Fatalf("GameEnded emitted inside vote-end batch; want deferred")
		}
	}
	state := e.Snapshot()
	if state.Phase != PhaseVote {
		t.Errorf("Phase=%s, want VOTE while pending end", state.Phase)
	}
	if state.PendingGameEnd == nil {
		t.Fatalf("PendingGameEnd is nil after vote-end tally")
	}
	if state.PendingGameEnd.Reason != EndCitizenWin {
		t.Errorf("PendingGameEnd.Reason=%q, want CITIZEN_WIN", state.PendingGameEnd.Reason)
	}
	if state.PendingGameEnd.Winner == nil || *state.PendingGameEnd.Winner != TeamCitizen {
		t.Errorf("PendingGameEnd.Winner=%v, want CITIZEN", state.PendingGameEnd.Winner)
	}
	want := clock.Now().Add(time.Duration(defaultFinalResultBufferSeconds) * time.Second)
	if got := state.PendingGameEnd.Deadline; !got.Equal(want) {
		t.Errorf("PendingGameEnd.Deadline=%v, want %v", got, want)
	}
}

// I9-T2 — Vote-end + 5s Tick: GameEnded fires, Phase transitions to END,
// PendingGameEnd is cleared, Winner/EndReason populated.
func TestI9_VoteEndTickFiresGameEnded(t *testing.T) {
	e, clock, _ := runToVoteEndCitizenWin(t, 9002)
	evs := runPendingEndTick(t, e, clock)

	var ge *GameEnded
	for _, ev := range evs {
		if g, ok := ev.Event.(GameEnded); ok {
			ge = &g
			break
		}
	}
	if ge == nil {
		t.Fatalf("GameEnded not emitted by pending-end Tick; events=%+v", evs)
	}
	if ge.EndReason != EndCitizenWin {
		t.Errorf("GameEnded.EndReason=%q, want CITIZEN_WIN", ge.EndReason)
	}
	if ge.Winner == nil || *ge.Winner != TeamCitizen {
		t.Errorf("GameEnded.Winner=%v, want CITIZEN", ge.Winner)
	}
	state := e.Snapshot()
	if state.Phase != PhaseEnd {
		t.Errorf("Phase=%s, want END after firePendingEnd", state.Phase)
	}
	if state.PendingGameEnd != nil {
		t.Errorf("PendingGameEnd still set after fire: %+v", state.PendingGameEnd)
	}
	if state.Winner == nil || *state.Winner != TeamCitizen {
		t.Errorf("State.Winner=%v, want CITIZEN", state.Winner)
	}
}

// I9-T3 — Night-end: resolveNight emits PhaseChanged{DAY} +
// DeathAnnounced and schedules GameEnded; no GameEnded inside the batch.
// State.Phase remains DAY during the buffer.
func TestI9_NightEndSchedulesPendingGameEnd(t *testing.T) {
	e, _ := runToNightEndMafiaWin(t, 9003)
	state := e.Snapshot()
	if state.PendingGameEnd == nil {
		t.Fatalf("PendingGameEnd is nil after night-end resolve (phase=%s)", state.Phase)
	}
	if state.PendingGameEnd.Reason != EndMafiaWin {
		t.Errorf("PendingGameEnd.Reason=%q, want MAFIA_WIN", state.PendingGameEnd.Reason)
	}
	if state.PendingGameEnd.Winner == nil || *state.PendingGameEnd.Winner != TeamMafia {
		t.Errorf("PendingGameEnd.Winner=%v, want MAFIA", state.PendingGameEnd.Winner)
	}
	if state.Phase != PhaseDay {
		t.Errorf("Phase=%s, want DAY during buffer", state.Phase)
	}
}

// I9-T4 — Night-end + 5s Tick: GameEnded fires, Phase=END, MAFIA_WIN.
func TestI9_NightEndTickFiresGameEnded(t *testing.T) {
	e, clock := runToNightEndMafiaWin(t, 9004)
	evs := runPendingEndTick(t, e, clock)
	var ge *GameEnded
	for _, ev := range evs {
		if g, ok := ev.Event.(GameEnded); ok {
			ge = &g
		}
	}
	if ge == nil {
		t.Fatalf("GameEnded not emitted by pending-end Tick; events=%+v", evs)
	}
	if ge.EndReason != EndMafiaWin {
		t.Errorf("EndReason=%q, want MAFIA_WIN", ge.EndReason)
	}
	if ge.Winner == nil || *ge.Winner != TeamMafia {
		t.Errorf("Winner=%v, want MAFIA", ge.Winner)
	}
	if e.Snapshot().Phase != PhaseEnd {
		t.Errorf("Phase=%s, want END after fire", e.Snapshot().Phase)
	}
}

// I9-T5 — Pause/Resume mid-buffer: PendingGameEnd.Deadline is shifted
// forward by the elapsed pause duration so the buffer's remaining time
// is preserved.
func TestI9_PauseResumeShiftsPendingGameEnd(t *testing.T) {
	e, clock, _ := runToVoteEndCitizenWin(t, 9005)
	state := e.Snapshot()
	originalDeadline := state.PendingGameEnd.Deadline

	// 1 second into the buffer, pause.
	clock.Advance(1 * time.Second)
	if _, _, err := e.Apply(PauseGame{HostID: state.HostID}); err != nil {
		t.Fatalf("PauseGame mid-buffer (phase=%s): %v", state.Phase, err)
	}
	if !e.Snapshot().Paused {
		t.Fatalf("Paused=false after PauseGame")
	}

	// 30s of paused wall-clock time. Tick stays a no-op.
	clock.Advance(30 * time.Second)
	if _, evs, err := e.Tick(clock.Now()); err != nil {
		t.Fatal(err)
	} else if len(evs) != 0 {
		t.Errorf("Tick during pause emitted %d events; want 0", len(evs))
	}
	if e.Snapshot().Phase == PhaseEnd {
		t.Fatalf("Phase=END during pause; pending end fired prematurely")
	}

	if _, _, err := e.Apply(ResumeGame{HostID: state.HostID}); err != nil {
		t.Fatalf("ResumeGame: %v", err)
	}
	got := e.Snapshot().PendingGameEnd.Deadline
	want := originalDeadline.Add(30 * time.Second)
	if !got.Equal(want) {
		t.Errorf("PendingGameEnd.Deadline=%v, want %v (= original + 30s pause)", got, want)
	}

	// Remaining 4s of buffer + 1ms.
	clock.Advance(4*time.Second + time.Millisecond)
	_, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	var sawEnd bool
	for _, ev := range evs {
		if _, ok := ev.Event.(GameEnded); ok {
			sawEnd = true
			break
		}
	}
	if !sawEnd {
		t.Errorf("GameEnded not emitted after pause+resume+remaining buffer; events=%+v", evs)
	}
	if e.Snapshot().Phase != PhaseEnd {
		t.Errorf("Phase=%s after final tick, want END", e.Snapshot().Phase)
	}
}

// I9-T6 — HOST_FORCE_END mid-buffer: ForceEndGame bypasses the buffer,
// emits GameEnded{HOST_FORCE_END, Winner=nil} immediately, and clears
// PendingGameEnd so a later Tick does not re-emit.
func TestI9_HostForceEndClearsPending(t *testing.T) {
	e, clock, _ := runToVoteEndCitizenWin(t, 9006)
	state := e.Snapshot()
	if state.PendingGameEnd == nil {
		t.Fatalf("setup: PendingGameEnd=nil")
	}

	clock.Advance(2 * time.Second)
	_, evs, err := e.Apply(ForceEndGame{HostID: state.HostID})
	if err != nil {
		t.Fatalf("ForceEndGame: %v", err)
	}
	var ge *GameEnded
	for _, ev := range evs {
		if g, ok := ev.Event.(GameEnded); ok {
			ge = &g
		}
	}
	if ge == nil {
		t.Fatalf("GameEnded not emitted by ForceEndGame; events=%+v", evs)
	}
	if ge.EndReason != EndHostForceEnd {
		t.Errorf("EndReason=%q, want HOST_FORCE_END", ge.EndReason)
	}
	if ge.Winner != nil {
		t.Errorf("Winner=%v, want nil for force end", ge.Winner)
	}
	state = e.Snapshot()
	if state.PendingGameEnd != nil {
		t.Errorf("PendingGameEnd not cleared after ForceEnd: %+v", state.PendingGameEnd)
	}
	if state.Phase != PhaseEnd {
		t.Errorf("Phase=%s, want END", state.Phase)
	}

	// A subsequent Tick beyond the original deadline must be a no-op
	// (game already ended; no double GameEnded).
	clock.Advance(10 * time.Second)
	if _, evs2, err := e.Tick(clock.Now()); err != nil {
		t.Fatal(err)
	} else {
		for _, ev := range evs2 {
			if _, ok := ev.Event.(GameEnded); ok {
				t.Errorf("GameEnded re-emitted after ForceEnd; events=%+v", evs2)
			}
		}
	}
}

// I9-T7 — Snapshot mid-buffer: PendingGameEnd survives JSON round-trip
// (snapshot persistence). After Restore, a Tick past the deadline still
// fires GameEnded with the original Reason/Winner.
func TestI9_SnapshotRoundTripPreservesPendingGameEnd(t *testing.T) {
	e, clock, _ := runToVoteEndCitizenWin(t, 9007)
	state := e.Snapshot()
	originalDeadline := state.PendingGameEnd.Deadline

	// Mid-buffer snapshot via JSON round-trip.
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var restored State
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if restored.PendingGameEnd == nil {
		t.Fatalf("PendingGameEnd lost across JSON round-trip; raw=%s", string(raw))
	}
	if !restored.PendingGameEnd.Deadline.Equal(originalDeadline) {
		t.Errorf("Deadline drift: got=%v want=%v", restored.PendingGameEnd.Deadline, originalDeadline)
	}
	if err := e.Restore(restored); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	clock.Advance(time.Duration(defaultFinalResultBufferSeconds)*time.Second + time.Millisecond)
	_, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	var sawEnd bool
	for _, ev := range evs {
		if g, ok := ev.Event.(GameEnded); ok {
			sawEnd = true
			if g.EndReason != EndCitizenWin {
				t.Errorf("EndReason after restore=%q, want CITIZEN_WIN", g.EndReason)
			}
		}
	}
	if !sawEnd {
		t.Errorf("GameEnded not emitted after Restore + Tick; events=%+v", evs)
	}
}
