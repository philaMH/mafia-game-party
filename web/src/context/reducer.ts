import type {
  AnnounceMsg,
  EventMsg,
  EventPayload,
  IncomingMsg,
  Player,
  PlayerID,
  Severity,
  State,
  YourInfo,
} from "../types/wire";
import { defaultOptions, teamOf } from "../types/wire";

export type ConnectionStatus = "connecting" | "connected" | "reconnecting" | "closed";

export interface PoliceResultEntry {
  target: PlayerID;
  team: "MAFIA" | "CITIZEN";
  receivedAt: number;
}

export interface VoteTallyEntry {
  counts: Record<PlayerID, number>;
  eliminated?: PlayerID;
  recount: boolean;
  receivedAt: number;
}

// AudioCueQueueEntry records one announce-sourced audio request and the
// event kind in effect when the announce arrived. Effect drains by seq so
// no cue is lost when React batches multiple WS frames into one render
// (which silently dropped phase.night, game.started, and intro.speaker
// before this change because the effect only saw `lastAnnounce` and
// always observed the *last* announcement in the batch).
export interface AudioCueQueueEntry {
  audioId: string;
  eventKind?: EventPayload["kind"];
  seq: number;
}

const AUDIO_CUE_LOG_CAP = 64;

export interface GameState {
  status: ConnectionStatus;
  clientId?: string;
  playerId?: PlayerID;
  token?: string;
  isHost: boolean;
  state?: State;
  your: YourInfo;
  lastAnnounce?: {
    subtitle: string;
    audioId?: string;
    severity: Severity;
    receivedAt: number;
  };
  lastEventKind?: EventPayload["kind"];
  lastPoliceResult?: PoliceResultEntry;
  lastVoteTally?: VoteTallyEntry;
  errors: { code: string; message: string; addedAt: number }[];
  voiceOn: boolean;
  // Iter7 — true when the host PublicView can play <audio> elements.
  // Previously named ttsAvailable when we relied on Web Speech; the
  // browser support surface is broader for HTMLAudioElement so this
  // flag is effectively always true on real browsers and only gated
  // for SSR / test environments without a `window`.
  audioAvailable: boolean;
  // Iteration 2 — GM seat & room lifecycle gating.
  hostToken?: string;
  roomOpened: boolean;
  hostOccupied: boolean;
  roomOptions?: import("../types/wire").Options;
  // Monotonic counter incremented on every room:closed message. Used by
  // GameContext to fire side effects (e.g., clearing the saved player
  // token from localStorage) without coupling the reducer to storage.
  roomClosedSeq: number;
  // FIFO log of audio cue requests, capped at AUDIO_CUE_LOG_CAP. Each
  // announce with a non-empty audioId appends one entry. GameContext
  // tracks the last seq it dispatched via a ref so every cue plays
  // exactly once even when multiple WS frames batch into one render.
  audioCues: AudioCueQueueEntry[];
  // Monotonic seq for audioCues. Preserved across logout / room:closed
  // so the GameContext ref-based watermark stays valid across resets.
  audioCueSeq: number;
}

export type GameAction =
  | { type: "ws_connecting" }
  | { type: "ws_open" }
  | { type: "ws_message"; msg: IncomingMsg }
  | { type: "ws_reconnecting" }
  | { type: "ws_closed" }
  | { type: "set_voice"; on: boolean }
  | { type: "audio_unavailable" }
  | { type: "ack_error"; addedAt: number }
  | { type: "logout" };

export const initialState: GameState = {
  status: "connecting",
  isHost: false,
  your: {},
  errors: [],
  voiceOn: true,
  audioAvailable:
    typeof window !== "undefined" && typeof Audio !== "undefined",
  roomOpened: false,
  hostOccupied: false,
  roomClosedSeq: 0,
  audioCues: [],
  audioCueSeq: 0,
};

export function gameReducer(state: GameState, action: GameAction): GameState {
  switch (action.type) {
    case "ws_connecting":
      return { ...state, status: "connecting" };
    case "ws_open":
      return { ...state, status: "connected" };
    case "ws_reconnecting":
      return { ...state, status: "reconnecting" };
    case "ws_closed":
      return { ...state, status: "closed" };
    case "ws_message":
      return applyIncoming(state, action.msg);
    case "set_voice":
      return { ...state, voiceOn: action.on };
    case "audio_unavailable":
      return { ...state, audioAvailable: false };
    case "ack_error":
      return { ...state, errors: state.errors.filter((e) => e.addedAt !== action.addedAt) };
    case "logout":
      return {
        ...initialState,
        status: state.status,
        audioAvailable: state.audioAvailable,
        voiceOn: state.voiceOn,
        // Preserve audioCueSeq so the GameContext ref-based watermark
        // continues monotonically; resetting to 0 would let stale seqs
        // shadow new cues until the watermark caught up again.
        audioCueSeq: state.audioCueSeq,
      };
    default:
      return state;
  }
}

function applyIncoming(state: GameState, msg: IncomingMsg): GameState {
  switch (msg.type) {
    case "welcome":
      return { ...state, clientId: msg.clientId };
    case "joined":
      return {
        ...state,
        playerId: msg.playerId,
        token: msg.token,
        isHost: msg.isHost,
      };
    case "snapshot":
      return {
        ...state,
        state: msg.state,
        your: msg.your,
        isHost: msg.isHost,
      };
    case "event":
      return applyEvent(state, msg);
    case "announce":
      return applyAnnounce(state, msg);
    case "error":
      return {
        ...state,
        errors: [
          ...state.errors,
          { code: msg.code, message: msg.message, addedAt: Date.now() },
        ],
      };
    case "host-token":
      // Receiving a host token proves we ARE the host, so clear any
      // hostOccupied flag that pushRoomState may have raced ahead of us
      // (e.g., during a host browser refresh where the prior conn's
      // ReleaseHost defer hadn't run yet when we registered).
      return { ...state, hostToken: msg.token, isHost: true, hostOccupied: false };
    case "room:opened":
      return { ...state, roomOpened: true, roomOptions: msg.options };
    case "room:host-occupied":
      return { ...state, hostOccupied: true };
    case "room:closed":
      // Drop player-scoped state (token, playerId, role/keyword, last
      // game state) so the next round starts clean. The host keeps
      // hostToken / isHost so they stay on the GM seat and return to
      // the OpenRoom configuration screen. audioAvailable / voiceOn are
      // device-level preferences and are preserved. roomClosedSeq is
      // bumped so a GameContext effect can clear localStorage.
      return {
        ...initialState,
        status: state.status,
        audioAvailable: state.audioAvailable,
        voiceOn: state.voiceOn,
        clientId: state.clientId,
        hostToken: state.hostToken,
        isHost: state.hostToken !== undefined,
        roomClosedSeq: state.roomClosedSeq + 1,
        // Preserve seq so the GameContext watermark keeps advancing —
        // see the same comment on the `logout` case.
        audioCueSeq: state.audioCueSeq,
      };
  }
}

function applyAnnounce(state: GameState, msg: AnnounceMsg): GameState {
  const lastAnnounce = {
    subtitle: msg.subtitle,
    audioId: msg.audioId,
    severity: msg.severity,
    receivedAt: Date.now(),
  };
  if (!msg.audioId) {
    return { ...state, lastAnnounce };
  }
  const seq = state.audioCueSeq + 1;
  const entry: AudioCueQueueEntry = {
    audioId: msg.audioId,
    eventKind: state.lastEventKind,
    seq,
  };
  const audioCues =
    state.audioCues.length >= AUDIO_CUE_LOG_CAP
      ? [...state.audioCues.slice(state.audioCues.length - AUDIO_CUE_LOG_CAP + 1), entry]
      : [...state.audioCues, entry];
  return {
    ...state,
    lastAnnounce,
    audioCues,
    audioCueSeq: seq,
  };
}

function applyEvent(state: GameState, msg: EventMsg): GameState {
  const ev = msg.event;
  const next: GameState = { ...state, lastEventKind: ev.kind };

  switch (ev.kind) {
    case "PlayerJoined":
      return applyPlayerJoined(next, ev.playerId, ev.name);
    case "GameStarted":
      return next;
    case "PhaseChanged":
      if (!state.state) return next;
      return {
        ...next,
        state: {
          ...state.state,
          phase: ev.phase,
          day: ev.day,
          deadline: ev.deadlineMs > 0 ? new Date(ev.deadlineMs).toISOString() : undefined,
          // NightStep is only meaningful inside PhaseNight; clear it on
          // every other transition so the player UI doesn't render a
          // stale "마피아가 행동 중" overlay during DAY/VOTE. The server
          // re-emits NightStepChanged when entering NIGHT.
          nightStep: undefined,
          // Drop NightStep deadline when leaving NIGHT so TimerBar
          // doesn't show a stale negative countdown.
          nightStepDeadline: undefined,
          // policeCheckedThisNight is per-NIGHT; every PhaseChanged
          // crosses a night boundary one way or another, so reset.
          policeCheckedThisNight: false,
        },
      };
    case "RoleRevealedToPlayer":
      return {
        ...next,
        your: {
          ...state.your,
          role: ev.role,
          keyword: ev.keyword,
          team: teamOf(ev.role),
        },
      };
    case "MafiaCohortRevealed":
      return {
        ...next,
        your: { ...state.your, mafiaCohort: ev.mafiaIds },
        state: state.state
          ? { ...state.state, mafiaRepresentativeId: ev.representativeId }
          : state.state,
      };
    case "IntroSpeakerChanged":
      if (!state.state) return next;
      return {
        ...next,
        state: {
          ...state.state,
          introSpeakerIdx: state.state.players.findIndex((p) => p.id === ev.playerId),
        },
      };
    case "MafiaTargetSelected":
      if (!state.state) return next;
      return {
        ...next,
        state: { ...state.state, pendingMafiaTarget: ev.target },
      };
    case "PoliceResult": {
      const nextState = state.state
        ? {
            ...state.state,
            policeCheckedThisNight: true,
            policeHistory: [
              ...(state.state.policeHistory ?? []),
              {
                day: state.state.day,
                target: ev.target,
                team: ev.team,
              },
            ],
          }
        : state.state;
      return {
        ...next,
        lastPoliceResult: {
          target: ev.target,
          team: ev.team,
          receivedAt: Date.now(),
        },
        state: nextState,
      };
    }
    case "NightStepChanged":
      if (!state.state) return next;
      return {
        ...next,
        state: {
          ...state.state,
          nightStep: ev.step,
          nightStepDeadline:
            ev.stepDeadlineMs && ev.stepDeadlineMs > 0
              ? new Date(ev.stepDeadlineMs).toISOString()
              : undefined,
        },
      };
    case "GamePaused":
      if (!state.state) return next;
      return {
        ...next,
        state: { ...state.state, paused: true },
      };
    case "GameResumed":
      if (!state.state) return next;
      return {
        ...next,
        state: {
          ...state.state,
          paused: false,
          // The server has already shifted the active timer's deadline
          // forward; reflect it here so TimerBar resumes from the
          // correct value. Phase tells us which deadline field to
          // update.
          ...(ev.phase === "DAY" && ev.deadlineMs && ev.deadlineMs > 0
            ? { deadline: new Date(ev.deadlineMs).toISOString() }
            : {}),
          ...(ev.phase === "NIGHT" && ev.deadlineMs && ev.deadlineMs > 0
            ? { nightStepDeadline: new Date(ev.deadlineMs).toISOString() }
            : {}),
        },
      };
    case "DeathAnnounced":
      return updateAlive(next, ev.victim, false);
    case "PeacefulNight":
      return next;
    case "DiscussionTimerTick":
      return next;
    case "VoteTallied":
      return {
        ...next,
        lastVoteTally: {
          counts: ev.counts,
          eliminated: ev.eliminated,
          recount: ev.recount,
          receivedAt: Date.now(),
        },
      };
    case "Eliminated":
      return updateAliveWithRole(next, ev.playerId, false, ev.role);
    case "MafiaRepresentativeReassigned":
      if (!state.state) return next;
      return {
        ...next,
        state: { ...state.state, mafiaRepresentativeId: ev.newId },
      };
    case "GameEnded":
      if (!state.state) return next;
      return {
        ...next,
        state: {
          ...state.state,
          phase: "END",
          winner: ev.winner,
          endReason: ev.endReason,
          players: ev.reveal,
        },
      };
    case "VoiceToggled":
      return { ...next, voiceOn: ev.on };
  }
  return next;
}

function applyPlayerJoined(state: GameState, id: PlayerID, name: string): GameState {
  const newPlayer: Player = { id, name, alive: true };
  if (!state.state) {
    // First PlayerJoined for a fresh PUBLIC viewer or freshly-joined client.
    // Initialize a stub LOBBY state so the host PC and player tabs render
    // the roster instead of the "waiting" placeholder.
    return {
      ...state,
      state: {
        gameId: "",
        phase: "LOBBY",
        day: 0,
        hostId: "",
        players: [newPlayer],
        settings: defaultOptions(1),
      },
    };
  }
  if (state.state.players.some((p) => p.id === id)) {
    return state;
  }
  return {
    ...state,
    state: {
      ...state.state,
      players: [...state.state.players, newPlayer],
    },
  };
}

function updateAlive(state: GameState, id: PlayerID, alive: boolean): GameState {
  if (!state.state) return state;
  const players: Player[] = state.state.players.map((p) =>
    p.id === id ? { ...p, alive } : p,
  );
  return { ...state, state: { ...state.state, players } };
}

function updateAliveWithRole(
  state: GameState,
  id: PlayerID,
  alive: boolean,
  role: Player["role"],
): GameState {
  if (!state.state) return state;
  const players: Player[] = state.state.players.map((p) =>
    p.id === id ? { ...p, alive, role } : p,
  );
  return { ...state, state: { ...state.state, players } };
}
