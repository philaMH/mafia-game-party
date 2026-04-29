// Wire protocol types — manual mirror of internal/transport/ws/protocol.go
// (Q-FD-U5-13=A). Keep this file in lockstep with backend changes; it is
// the single source of truth for U5's typing of incoming/outgoing JSON.

export type PlayerID = string;
export type Role = "MAFIA" | "CITIZEN" | "DOCTOR" | "POLICE";
export type Team = "MAFIA" | "CITIZEN";
export type Phase = "LOBBY" | "INTRO" | "NIGHT" | "DAY" | "VOTE" | "RECOUNT" | "END";
export type NightStep = "MAFIA" | "POLICE" | "DOCTOR" | "RESOLVED";
export type EndReason = "MAFIA_WIN" | "CITIZEN_WIN" | "HOST_FORCE_END";

export interface PoliceCheckRecord {
  day: number;
  target: PlayerID;
  team: Team;
}
export type Severity = "INFO" | "EMPHASIS" | "WARN";

export interface Player {
  id: PlayerID;
  name: string;
  alive: boolean;
  // Role and keyword are masked in PublicView and other-player rows; only
  // populated for the viewer's own player or after PhaseEnd reveal.
  role?: Role;
  keyword?: string;
}

export interface Options {
  mafiaCount: number;
  maxPlayers: number;
  introSecondsPerPlayer: number;
  discussionSeconds: number;
  // Iteration 5 R6 — fixed wall-clock duration for each NightStep.
  nightMafiaSeconds: number;
  nightPoliceSeconds: number;
  nightDoctorSeconds: number;
  doctorSelfHealAllowed: boolean;
  announcementVoiceOn: boolean;
}

export interface State {
  gameId: string;
  phase: Phase;
  day: number;
  players: Player[];
  hostId: PlayerID;
  settings: Options;
  startedAt?: string;
  deadline?: string;
  introSpeakerIdx?: number;
  introSpeakerStartedAt?: string;
  mafiaRepresentativeId?: PlayerID;
  pendingMafiaTarget?: PlayerID;
  pendingDoctorTarget?: PlayerID;
  pendingPoliceTarget?: PlayerID;
  policeCheckedThisNight?: boolean;
  votes?: Record<PlayerID, PlayerID>;
  voteRound?: number;
  voteCandidates?: PlayerID[];
  nightStep?: NightStep;
  // Iteration 5 — wall-clock instant when the current NightStep
  // auto-advances. ISO string when present.
  nightStepDeadline?: string;
  // Iteration 5 — host pause flag. When true the public TimerBar
  // freezes its countdown and the host UI shows the Resume button.
  paused?: boolean;
  pausedAt?: string;
  policeHistory?: PoliceCheckRecord[];
  winner?: Team;
  endReason?: EndReason;
  lastTickAt?: string;
}

export interface YourInfo {
  role?: Role;
  keyword?: string;
  team?: Team;
  mafiaCohort?: PlayerID[];
}

// ---------- Incoming (server -> client) ----------

export type IncomingMsg =
  | WelcomeMsg
  | JoinedMsg
  | SnapshotMsg
  | EventMsg
  | AnnounceMsg
  | ErrorMsg
  | HostTokenMsg
  | RoomOpenedMsg
  | RoomHostOccupiedMsg
  | RoomClosedMsg;

export interface HostTokenMsg {
  type: "host-token";
  token: string;
}

export interface RoomOpenedMsg {
  type: "room:opened";
  options: Options;
}

export interface RoomHostOccupiedMsg {
  type: "room:host-occupied";
}

export interface RoomClosedMsg {
  type: "room:closed";
}

export interface WelcomeMsg {
  type: "welcome";
  clientId: string;
  kind: "PUBLIC" | "PLAYER";
  protocolVersion: string;
}

export interface JoinedMsg {
  type: "joined";
  playerId: PlayerID;
  token: string;
  isHost: boolean;
}

export interface SnapshotMsg {
  type: "snapshot";
  state: State;
  your: YourInfo;
  isHost: boolean;
}

export interface EventMsg {
  type: "event";
  visibility: "PUBLIC" | "PLAYER" | "ROLE_MAFIA";
  event: EventPayload;
}

export interface AnnounceMsg {
  type: "announce";
  subtitle: string;
  // Iter7 FR-8.9 — stable cue id mapping to /audio/<audioId>.mp3 on the
  // host PublicView. Empty when the catalog produced no audio (graceful
  // skip — Iter7 FR-8.8) or for legacy events without a recording yet.
  audioId?: string;
  severity: Severity;
}

export interface ErrorMsg {
  type: "error";
  code: string;
  message: string;
}

// ---------- Event payloads (15 kinds, mirror of buildEventPayload) ----------

export type EventPayload =
  | { kind: "PlayerJoined"; playerId: PlayerID; name: string }
  | { kind: "GameStarted" }
  | { kind: "PhaseChanged"; phase: Phase; day: number; deadlineMs: number }
  | { kind: "RoleRevealedToPlayer"; playerId: PlayerID; role: Role; keyword: string }
  | { kind: "MafiaCohortRevealed"; mafiaIds: PlayerID[]; representativeId: PlayerID }
  | { kind: "IntroSpeakerChanged"; playerId: PlayerID; secondsLeft: number }
  | { kind: "MafiaTargetSelected"; representativeId: PlayerID; target: PlayerID }
  | { kind: "PoliceResult"; police: PlayerID; target: PlayerID; team: Team }
  | { kind: "DeathAnnounced"; victim: PlayerID }
  | { kind: "PeacefulNight" }
  | { kind: "DiscussionTimerTick"; secondsLeft: number }
  | {
      kind: "VoteTallied";
      counts: Record<PlayerID, number>;
      eliminated?: PlayerID;
      recount: boolean;
    }
  | { kind: "Eliminated"; playerId: PlayerID; role: Role }
  | { kind: "MafiaRepresentativeReassigned"; oldId: PlayerID; newId: PlayerID }
  | { kind: "GameEnded"; winner?: Team; endReason: EndReason; reveal: Player[] }
  | { kind: "VoiceToggled"; on: boolean }
  | { kind: "NightStepChanged"; step: NightStep; day: number; stepDeadlineMs?: number }
  | { kind: "GamePaused"; phase: Phase }
  | { kind: "GameResumed"; phase: Phase; deadlineMs?: number };

// ---------- Outgoing (client -> server) ----------

export type OutgoingMsg =
  | { type: "host:create-session"; name: string }
  | { type: "join"; name: string }
  | { type: "resume"; token: string }
  | { type: "host:start"; options: Options }
  | { type: "submit:advance-intro" }
  | { type: "submit:mafia-kill"; target: PlayerID }
  | { type: "submit:doctor-heal"; target: PlayerID }
  | { type: "submit:police-check"; target: PlayerID }
  | { type: "submit:end-night" }
  | { type: "submit:end-discussion" }
  | { type: "submit:vote"; target: PlayerID }
  | { type: "host:toggle-voice"; on: boolean }
  | { type: "host:force-end" }
  | { type: "subscribe-public" }
  | { type: "host:claim" }
  | { type: "host:open-room"; options: Options }
  | { type: "host:start-room" }
  | { type: "host:terminate-room" }
  | { type: "host:close-room" }
  | { type: "player:end-self-intro" }
  | { type: "host:pause" }
  | { type: "host:resume" };

// ---------- Default options helper ----------

export function defaultOptions(playerCount: number): Options {
  return {
    mafiaCount: playerCount <= 6 ? 1 : playerCount <= 9 ? 2 : 3,
    maxPlayers: Math.max(6, Math.min(12, playerCount || 8)),
    introSecondsPerPlayer: 20,
    discussionSeconds: 180,
    nightMafiaSeconds: 30,
    nightPoliceSeconds: 10,
    nightDoctorSeconds: 10,
    doctorSelfHealAllowed: true,
    announcementVoiceOn: true,
  };
}

export function teamOf(role: Role): Team {
  return role === "MAFIA" ? "MAFIA" : "CITIZEN";
}
