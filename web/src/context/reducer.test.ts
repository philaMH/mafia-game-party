import { describe, expect, it } from "vitest";

import { gameReducer, initialState, type GameAction } from "./reducer";
import type { State } from "../types/wire";

const baseState: State = {
  gameId: "g",
  phase: "LOBBY",
  day: 0,
  hostId: "h",
  players: [
    { id: "h", name: "host", alive: true },
    { id: "p1", name: "철수", alive: true },
    { id: "p2", name: "영희", alive: true },
  ],
  settings: {
    mafiaCount: 1,
    maxPlayers: 6,
    introSecondsPerPlayer: 20,
    discussionSeconds: 180,
    nightMafiaSeconds: 30,
    nightPoliceSeconds: 10,
    nightDoctorSeconds: 10,
    doctorSelfHealAllowed: true,
    announcementVoiceOn: true,
  },
  votes: {},
};

function reduce(action: GameAction) {
  return gameReducer({ ...initialState, state: baseState }, action);
}

describe("gameReducer", () => {
  it("ws_connecting/open/reconnecting/closed updates status", () => {
    expect(gameReducer(initialState, { type: "ws_open" }).status).toBe("connected");
    expect(gameReducer(initialState, { type: "ws_reconnecting" }).status).toBe("reconnecting");
    expect(gameReducer(initialState, { type: "ws_closed" }).status).toBe("closed");
    expect(gameReducer(initialState, { type: "ws_connecting" }).status).toBe("connecting");
  });

  it("welcome message sets clientId", () => {
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "welcome", clientId: "abc", kind: "PUBLIC", protocolVersion: "v1" },
    });
    expect(next.clientId).toBe("abc");
  });

  it("joined message sets playerId/token/isHost", () => {
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "joined", playerId: "p1", token: "tok", isHost: true },
    });
    expect(next.playerId).toBe("p1");
    expect(next.token).toBe("tok");
    expect(next.isHost).toBe(true);
  });

  it("snapshot replaces state and your", () => {
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: {
        type: "snapshot",
        state: baseState,
        your: { role: "DOCTOR", keyword: "신뢰", team: "CITIZEN" },
        isHost: false,
      },
    });
    expect(next.state?.phase).toBe("LOBBY");
    expect(next.your.role).toBe("DOCTOR");
  });

  it("PhaseChanged updates phase/day/deadline", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PhaseChanged", phase: "DAY", day: 2, deadlineMs: 1714000000000 },
      },
    });
    expect(next.state?.phase).toBe("DAY");
    expect(next.state?.day).toBe(2);
    expect(next.state?.deadline).toBeDefined();
  });

  it("RoleRevealedToPlayer updates your", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PLAYER",
        event: { kind: "RoleRevealedToPlayer", playerId: "p1", role: "MAFIA", keyword: "k" },
      },
    });
    expect(next.your.role).toBe("MAFIA");
    expect(next.your.team).toBe("MAFIA");
    expect(next.your.keyword).toBe("k");
  });

  it("MafiaCohortRevealed updates cohort and representative", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "ROLE_MAFIA",
        event: {
          kind: "MafiaCohortRevealed",
          mafiaIds: ["p1", "p2"],
          representativeId: "p1",
        },
      },
    });
    expect(next.your.mafiaCohort).toEqual(["p1", "p2"]);
    expect(next.state?.mafiaRepresentativeId).toBe("p1");
  });

  it("MafiaTargetSelected updates pendingMafiaTarget", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "ROLE_MAFIA",
        event: { kind: "MafiaTargetSelected", representativeId: "p1", target: "p2" },
      },
    });
    expect(next.state?.pendingMafiaTarget).toBe("p2");
  });

  it("PoliceResult sets lastPoliceResult and policeCheckedThisNight", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PLAYER",
        event: { kind: "PoliceResult", police: "p1", target: "p2", team: "MAFIA" },
      },
    });
    expect(next.lastPoliceResult?.target).toBe("p2");
    expect(next.lastPoliceResult?.team).toBe("MAFIA");
    expect(next.state?.policeCheckedThisNight).toBe(true);
  });

  it("DeathAnnounced flips alive=false", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "DeathAnnounced", victim: "p1" },
      },
    });
    const p1 = next.state?.players.find((p) => p.id === "p1");
    expect(p1?.alive).toBe(false);
  });

  it("Eliminated flips alive=false and assigns role", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "Eliminated", playerId: "p2", role: "MAFIA" },
      },
    });
    const p2 = next.state?.players.find((p) => p.id === "p2");
    expect(p2?.alive).toBe(false);
    expect(p2?.role).toBe("MAFIA");
  });

  it("VoteTallied stores tally", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "VoteTallied", counts: { p1: 2, p2: 1 }, recount: false, eliminated: "p1" },
      },
    });
    expect(next.lastVoteTally?.eliminated).toBe("p1");
    expect(next.lastVoteTally?.counts.p1).toBe(2);
  });

  it("MafiaRepresentativeReassigned updates representative", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "ROLE_MAFIA",
        event: { kind: "MafiaRepresentativeReassigned", oldId: "p1", newId: "p2" },
      },
    });
    expect(next.state?.mafiaRepresentativeId).toBe("p2");
  });

  it("GameEnded transitions to END and reveals players", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: {
          kind: "GameEnded",
          winner: "CITIZEN",
          endReason: "CITIZEN_WIN",
          reveal: [{ id: "p1", name: "철수", alive: false, role: "MAFIA" }],
        },
      },
    });
    expect(next.state?.phase).toBe("END");
    expect(next.state?.winner).toBe("CITIZEN");
    expect(next.state?.players[0]?.role).toBe("MAFIA");
  });

  it("PlayerJoined appends to existing lobby roster", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PlayerJoined", playerId: "p3", name: "민지" },
      },
    });
    expect(next.state?.players).toHaveLength(4);
    expect(next.state?.players[3]?.id).toBe("p3");
    expect(next.state?.players[3]?.name).toBe("민지");
    expect(next.state?.players[3]?.alive).toBe(true);
    expect(next.state?.players[3]?.role).toBeUndefined();
  });

  it("PlayerJoined initializes LOBBY state for fresh PUBLIC viewer", () => {
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PlayerJoined", playerId: "host", name: "호스트" },
      },
    });
    expect(next.state).toBeDefined();
    expect(next.state?.phase).toBe("LOBBY");
    expect(next.state?.players).toHaveLength(1);
    expect(next.state?.players[0]?.name).toBe("호스트");
  });

  it("PlayerJoined is idempotent on duplicate id", () => {
    const once = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PlayerJoined", playerId: "p1", name: "철수" },
      },
    });
    expect(once.state?.players).toHaveLength(3);
  });

  it("VoiceToggled mirrors voiceOn", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "VoiceToggled", on: false },
      },
    });
    expect(next.voiceOn).toBe(false);
  });

  it("announce updates lastAnnounce with audioId", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "테스트 안내",
        audioId: "phase.night",
        severity: "EMPHASIS",
      },
    });
    expect(next.lastAnnounce?.subtitle).toBe("테스트 안내");
    expect(next.lastAnnounce?.audioId).toBe("phase.night");
    expect(next.lastAnnounce?.severity).toBe("EMPHASIS");
  });

  it("announce without audioId leaves it undefined (graceful skip)", () => {
    const next = reduce({
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "자막 전용",
        severity: "INFO",
      },
    });
    expect(next.lastAnnounce?.subtitle).toBe("자막 전용");
    expect(next.lastAnnounce?.audioId).toBeUndefined();
    // Audio cue log must NOT grow when audioId is empty.
    expect(next.audioCues).toHaveLength(0);
    expect(next.audioCueSeq).toBe(0);
  });

  it("announce with audioId appends a unique-seq audioCue tagged with current lastEventKind", () => {
    // Simulate the server frame order: PhaseChanged event arrives first,
    // then the corresponding announce. The cue should record
    // eventKind=PhaseChanged so the URGENT branch fires downstream.
    const afterPhase = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PhaseChanged", phase: "NIGHT", day: 2, deadlineMs: 0 },
      },
    });
    const afterAnnounce = gameReducer(afterPhase, {
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "이제 밤이 깊어졌습니다.",
        audioId: "phase.night",
        severity: "EMPHASIS",
      },
    });
    expect(afterAnnounce.audioCues).toHaveLength(1);
    expect(afterAnnounce.audioCues[0]).toMatchObject({
      audioId: "phase.night",
      eventKind: "PhaseChanged",
      seq: 1,
    });
    expect(afterAnnounce.audioCueSeq).toBe(1);
  });

  it("audioCues preserves every announce when batched between PhaseChanged and NightStepChanged (regression)", () => {
    // Server emits PhaseChanged(NIGHT) → announce phase.night →
    // NightStepChanged(MAFIA) → announce night.mafia. Even when React
    // batches all four into a single render, every cue must remain
    // recorded in the queue with its own seq and eventKind.
    let s = reduce({
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PhaseChanged", phase: "NIGHT", day: 2, deadlineMs: 0 },
      },
    });
    s = gameReducer(s, {
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "이제 밤이 깊어졌습니다.",
        audioId: "phase.night",
        severity: "EMPHASIS",
      },
    });
    s = gameReducer(s, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "NightStepChanged", step: "MAFIA", day: 2 },
      },
    });
    s = gameReducer(s, {
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "마피아는 눈을 뜨고…",
        audioId: "night.mafia",
        severity: "EMPHASIS",
      },
    });
    expect(s.audioCues).toHaveLength(2);
    expect(s.audioCues[0]).toMatchObject({
      audioId: "phase.night",
      eventKind: "PhaseChanged",
      seq: 1,
    });
    expect(s.audioCues[1]).toMatchObject({
      audioId: "night.mafia",
      eventKind: "NightStepChanged",
      seq: 2,
    });
  });

  it("room:closed clears audioCues but preserves audioCueSeq watermark", () => {
    let s = reduce({
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "마피아 게임이 시작됩니다.",
        audioId: "game.started",
        severity: "EMPHASIS",
      },
    });
    expect(s.audioCueSeq).toBe(1);
    s = gameReducer(
      { ...s, hostToken: "h-tok" },
      { type: "ws_message", msg: { type: "room:closed" } },
    );
    expect(s.audioCues).toHaveLength(0);
    // Preserved so the GameContext watermark keeps advancing past the
    // last cue from the previous game.
    expect(s.audioCueSeq).toBe(1);
    // Host stays seated, so a fresh game-start announce continues from seq 2.
    s = gameReducer(s, {
      type: "ws_message",
      msg: {
        type: "announce",
        subtitle: "마피아 게임이 시작됩니다.",
        audioId: "game.started",
        severity: "EMPHASIS",
      },
    });
    expect(s.audioCues).toEqual([
      expect.objectContaining({ audioId: "game.started", seq: 2 }),
    ]);
  });

  it("error appends to errors and ack_error removes by addedAt", () => {
    const added = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "error", code: "VALIDATION_ERROR", message: "bad" },
    });
    expect(added.errors).toHaveLength(1);
    const addedAt = added.errors[0]!.addedAt;
    const next = gameReducer(added, { type: "ack_error", addedAt });
    expect(next.errors).toHaveLength(0);
  });

  it("logout resets identity but preserves voice/tts", () => {
    const seeded = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "joined", playerId: "p1", token: "tok", isHost: false },
    });
    const next = gameReducer(seeded, { type: "logout" });
    expect(next.playerId).toBeUndefined();
    expect(next.token).toBeUndefined();
    expect(next.voiceOn).toBe(seeded.voiceOn);
  });

  it("host-token sets hostToken and isHost (Iteration 2)", () => {
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "host-token", token: "abc" },
    });
    expect(next.hostToken).toBe("abc");
    expect(next.isHost).toBe(true);
  });

  it("room:opened sets roomOpened and roomOptions (Iteration 2)", () => {
    const opts = {
      mafiaCount: 2,
      maxPlayers: 8,
      introSecondsPerPlayer: 20,
      discussionSeconds: 180,
      nightMafiaSeconds: 30,
      nightPoliceSeconds: 10,
      nightDoctorSeconds: 10,
      doctorSelfHealAllowed: true,
      announcementVoiceOn: true,
    };
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "room:opened", options: opts },
    });
    expect(next.roomOpened).toBe(true);
    expect(next.roomOptions?.maxPlayers).toBe(8);
  });

  it("room:host-occupied sets hostOccupied (Iteration 2)", () => {
    const next = gameReducer(initialState, {
      type: "ws_message",
      msg: { type: "room:host-occupied" },
    });
    expect(next.hostOccupied).toBe(true);
  });

  it("NightStepChanged updates state.nightStep (Iteration 4)", () => {
    const seeded = {
      ...initialState,
      state: { ...baseState, phase: "NIGHT" as const, day: 2 },
    };
    const next = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "NightStepChanged", step: "POLICE", day: 2 },
      },
    });
    expect(next.state?.nightStep).toBe("POLICE");
  });

  it("PoliceResult appends to policeHistory (Iteration 4)", () => {
    const seeded = {
      ...initialState,
      state: { ...baseState, phase: "NIGHT" as const, day: 2 },
    };
    const after1 = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PLAYER",
        event: { kind: "PoliceResult", police: "h", target: "p1", team: "MAFIA" },
      },
    });
    expect(after1.state?.policeHistory).toEqual([
      { day: 2, target: "p1", team: "MAFIA" },
    ]);
    expect(after1.state?.policeCheckedThisNight).toBe(true);

    // Next NIGHT (day=3) — phase change clears the per-night flag and
    // a new investigation appends a second history entry.
    const after2 = gameReducer(after1, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PhaseChanged", phase: "NIGHT", day: 3, deadlineMs: 0 },
      },
    });
    expect(after2.state?.policeCheckedThisNight).toBe(false);
    const after3 = gameReducer(after2, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PLAYER",
        event: { kind: "PoliceResult", police: "h", target: "p2", team: "CITIZEN" },
      },
    });
    expect(after3.state?.policeHistory).toHaveLength(2);
    expect(after3.state?.policeHistory?.[1]).toEqual({
      day: 3,
      target: "p2",
      team: "CITIZEN",
    });
  });

  it("PhaseChanged out of NIGHT clears nightStep (Iteration 4)", () => {
    const seeded = {
      ...initialState,
      state: { ...baseState, phase: "NIGHT" as const, day: 2, nightStep: "DOCTOR" as const },
    };
    const next = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PhaseChanged", phase: "DAY", day: 3, deadlineMs: 0 },
      },
    });
    expect(next.state?.nightStep).toBeUndefined();
    expect(next.state?.phase).toBe("DAY");
  });

  it("NightStepChanged with stepDeadlineMs records ISO deadline (Iteration 5)", () => {
    const seeded = {
      ...initialState,
      state: { ...baseState, phase: "NIGHT" as const, day: 2 },
    };
    const ts = 1714000000000;
    const next = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: {
          kind: "NightStepChanged",
          step: "POLICE",
          day: 2,
          stepDeadlineMs: ts,
        },
      },
    });
    expect(next.state?.nightStep).toBe("POLICE");
    expect(next.state?.nightStepDeadline).toBe(new Date(ts).toISOString());
  });

  it("GamePaused / GameResumed toggle state.paused (Iteration 5)", () => {
    const seeded = {
      ...initialState,
      state: { ...baseState, phase: "NIGHT" as const, day: 2 },
    };
    const paused = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "GamePaused", phase: "NIGHT" },
      },
    });
    expect(paused.state?.paused).toBe(true);

    const ts = 1714000123000;
    const resumed = gameReducer(paused, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "GameResumed", phase: "NIGHT", deadlineMs: ts },
      },
    });
    expect(resumed.state?.paused).toBe(false);
    expect(resumed.state?.nightStepDeadline).toBe(new Date(ts).toISOString());
  });

  it("GameResumed during DAY shifts the discussion deadline (Iteration 5)", () => {
    const seeded = {
      ...initialState,
      state: {
        ...baseState,
        phase: "DAY" as const,
        day: 2,
        deadline: new Date(1700000000000).toISOString(),
        paused: true,
      },
    };
    const ts = 1700000060000;
    const next = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "GameResumed", phase: "DAY", deadlineMs: ts },
      },
    });
    expect(next.state?.paused).toBe(false);
    expect(next.state?.deadline).toBe(new Date(ts).toISOString());
  });

  it("PhaseChanged out of NIGHT also clears nightStepDeadline (Iteration 5)", () => {
    const seeded = {
      ...initialState,
      state: {
        ...baseState,
        phase: "NIGHT" as const,
        day: 2,
        nightStep: "DOCTOR" as const,
        nightStepDeadline: new Date(1700000000000).toISOString(),
      },
    };
    const next = gameReducer(seeded, {
      type: "ws_message",
      msg: {
        type: "event",
        visibility: "PUBLIC",
        event: { kind: "PhaseChanged", phase: "DAY", day: 3, deadlineMs: 0 },
      },
    });
    expect(next.state?.nightStepDeadline).toBeUndefined();
  });
});
