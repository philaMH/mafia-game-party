package game

import (
	"errors"
	"testing"
	"time"
)

// Iteration 8 — Fix · 밤 진입 안내. NightStepIntro buffer at NIGHT entry +
// defaultDayIntroSeconds buffer at NIGHT->DAY transition + Pause guard for
// the INTRO step. See iteration8-fix-vote-result-requirements.md.

// reachVoteAbstain runs the engine through INTRO -> DAY 1 -> VOTE -> all
// abstain, leaving the engine in NIGHT 1 right after the vote tally fires
// enterNight(). It does NOT drain the INTRO buffer (unlike advanceToNight).
func reachVoteAbstain(t *testing.T, e Engine) State {
	t.Helper()
	state := e.Snapshot()
	for state.Phase == PhaseIntro {
		if _, _, err := e.Apply(AdvanceIntro{HostID: state.HostID}); err != nil {
			t.Fatalf("AdvanceIntro: %v", err)
		}
		state = e.Snapshot()
	}
	if _, _, err := e.Apply(EndDiscussionEarly{HostID: state.HostID}); err != nil {
		t.Fatalf("EndDiscussionEarly: %v", err)
	}
	state = e.Snapshot()
	for _, p := range state.Players {
		if !p.Alive {
			continue
		}
		if _, _, err := e.Apply(SubmitVote{Voter: p.ID, Target: ""}); err != nil {
			t.Fatalf("SubmitVote abstain: %v", err)
		}
	}
	state = e.Snapshot()
	if state.Phase != PhaseNight {
		t.Fatalf("expected NIGHT, got %s", state.Phase)
	}
	return state
}

// I8-T1 — NIGHT 진입 직후 NightStep=INTRO, Deadline = now + 5s.
func TestI8_NightStepIntroOnEntry(t *testing.T) {
	e, clock := newTestEngine(t, 8001)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state := reachVoteAbstain(t, e)

	if state.NightStep != NightStepIntro {
		t.Fatalf("NightStep=%q, want INTRO", state.NightStep)
	}
	got := state.NightStepDeadline.Sub(clock.Now())
	want := time.Duration(defaultNightIntroSeconds) * time.Second
	if got != want {
		t.Errorf("INTRO deadline offset=%s, want %s", got, want)
	}
}

// I8-T2 — INTRO 만료 후 Tick 으로 MAFIA 자동 전이; mafia deadline 은
// introDeadline + NightMafiaSeconds 와 정확히 일치.
func TestI8_IntroExpiresToMafia(t *testing.T) {
	e, clock := newTestEngine(t, 8002)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state := reachVoteAbstain(t, e)
	introDeadline := state.NightStepDeadline

	// 5s 미만 -> INTRO 유지.
	clock.T = introDeadline.Add(-1 * time.Millisecond)
	if _, _, err := e.Tick(clock.Now()); err != nil {
		t.Fatal(err)
	}
	if e.Snapshot().NightStep != NightStepIntro {
		t.Fatalf("INTRO advanced before deadline: %q", e.Snapshot().NightStep)
	}

	// 5s + 1ms -> MAFIA. NightStepChanged{MAFIA} 도 emit.
	clock.T = introDeadline.Add(time.Millisecond)
	_, evs, err := e.Tick(clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	snap := e.Snapshot()
	if snap.NightStep != NightStepMafia {
		t.Fatalf("NightStep=%q, want MAFIA", snap.NightStep)
	}
	mafiaDur := time.Duration(DefaultOptions(8).NightMafiaSeconds) * time.Second
	if got, want := snap.NightStepDeadline, introDeadline.Add(mafiaDur); !got.Equal(want) {
		t.Errorf("MAFIA deadline=%v, want %v", got, want)
	}
	var sawMafia bool
	for _, ev := range evs {
		if ns, ok := ev.Event.(NightStepChanged); ok && ns.Step == NightStepMafia {
			sawMafia = true
			break
		}
	}
	if !sawMafia {
		t.Errorf("Tick did not emit NightStepChanged{MAFIA}; events=%+v", evs)
	}
}

// I8-T3 — nightStepSeconds(opts, NightStepIntro) is the constant regardless
// of Options, so the host cannot accidentally widen or shrink the buffer.
func TestI8_NightStepSecondsIntroFixed(t *testing.T) {
	cases := []Options{
		{},
		{NightMafiaSeconds: 60, NightPoliceSeconds: 60, NightDoctorSeconds: 60},
		{NightMafiaSeconds: 1, NightPoliceSeconds: 1, NightDoctorSeconds: 1},
		DefaultOptions(8),
	}
	for _, opts := range cases {
		if got := nightStepSeconds(opts, NightStepIntro); got != defaultNightIntroSeconds {
			t.Errorf("opts=%+v: nightStepSeconds(INTRO)=%d, want %d",
				opts, got, defaultNightIntroSeconds)
		}
	}
}

// I8-T4 — resolveNight() 후 Day Deadline = now + (defaultDayIntroSeconds +
// DiscussionSeconds). 회귀: Iteration 5 의 자동 step 진행과 결합 검증.
func TestI8_ResolveNightAddsDayIntroBuffer(t *testing.T) {
	e, clock := newTestEngine(t, 8004)
	opts := DefaultOptions(8)
	mustStart(t, e, playerSet(8), "p1", opts)
	advanceToNight(t, e) // INTRO drained, MAFIA active

	// Drain MAFIA -> POLICE -> DOCTOR -> resolveNight.
	advanceNightStep(t, e, clock) // MAFIA -> POLICE
	advanceNightStep(t, e, clock) // POLICE -> DOCTOR
	advanceNightStep(t, e, clock) // DOCTOR -> resolveNight -> DAY

	snap := e.Snapshot()
	if snap.Phase != PhaseDay {
		t.Fatalf("Phase=%s, want DAY after resolveNight", snap.Phase)
	}
	wantOffset := time.Duration(defaultDayIntroSeconds+opts.DiscussionSeconds) * time.Second
	gotOffset := snap.Deadline.Sub(clock.Now())
	if gotOffset != wantOffset {
		t.Errorf("Day Deadline offset=%s, want %s", gotOffset, wantOffset)
	}
}

// I8-T5 — 첫째날 (transitionIntroToDay) 의 Day Deadline 은 버퍼 없음.
func TestI8_FirstDayHasNoDayIntroBuffer(t *testing.T) {
	e, clock := newTestEngine(t, 8005)
	opts := DefaultOptions(8)
	mustStart(t, e, playerSet(8), "p1", opts)

	// Walk through intro speakers to trigger transitionIntroToDay at the end.
	for {
		state := e.Snapshot()
		if state.Phase != PhaseIntro {
			break
		}
		if _, _, err := e.Apply(AdvanceIntro{HostID: state.HostID}); err != nil {
			t.Fatalf("AdvanceIntro: %v", err)
		}
	}
	snap := e.Snapshot()
	if snap.Phase != PhaseDay || snap.Day != 1 {
		t.Fatalf("expected DAY 1 after intro, got Phase=%s Day=%d", snap.Phase, snap.Day)
	}
	wantOffset := time.Duration(opts.DiscussionSeconds) * time.Second
	gotOffset := snap.Deadline.Sub(clock.Now())
	if gotOffset != wantOffset {
		t.Errorf("Day 1 Deadline offset=%s, want %s (no buffer)", gotOffset, wantOffset)
	}
}

// I8-T6 — INTRO 단계에서 PauseGame 거부. Paused 필드는 false 유지.
func TestI8_PauseDuringIntroRejected(t *testing.T) {
	e, _ := newTestEngine(t, 8006)
	state, _ := mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	mid := reachVoteAbstain(t, e)
	if mid.NightStep != NightStepIntro {
		t.Fatalf("precondition failed: NightStep=%q, want INTRO", mid.NightStep)
	}

	_, _, err := e.Apply(PauseGame{HostID: state.HostID})
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("PauseGame during INTRO err=%v, want ErrWrongPhase", err)
	}
	if e.Snapshot().Paused {
		t.Error("Paused=true after rejected PauseGame")
	}
}

// I8-T7 — legacy snapshot with NightStep=MAFIA still works (no INTRO).
// Restore directly into MAFIA, verify Tick advances normally.
func TestI8_LegacySnapshotMafiaStepStillWorks(t *testing.T) {
	e, clock := newTestEngine(t, 8007)
	mustStart(t, e, playerSet(8), "p1", DefaultOptions(8))
	state := advanceToNight(t, e) // ends with NightStep=MAFIA after INTRO drained

	// Simulate legacy snapshot: Restore the same state (NightStep=MAFIA).
	if err := e.Restore(state); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if e.Snapshot().NightStep != NightStepMafia {
		t.Fatalf("Restore changed NightStep: %q", e.Snapshot().NightStep)
	}

	// Mafia rep submits a kill — must succeed (MAFIA accepts SubmitMafiaKill).
	_, _, _, citizens := allRoles(state)
	if _, _, err := e.Apply(SubmitMafiaKill{
		Mafia:  state.MafiaRepresentativeID,
		Target: citizens[0],
	}); err != nil {
		t.Errorf("SubmitMafiaKill on legacy MAFIA snapshot: %v", err)
	}

	// Drain MAFIA -> POLICE; deadline math is independent of the missing INTRO.
	advanceNightStep(t, e, clock)
	if got := e.Snapshot().NightStep; got != NightStepPolice {
		t.Errorf("NightStep=%q after MAFIA tick, want POLICE", got)
	}
}
