package ws

import (
	"encoding/json"

	"github.com/saltware/mafia-game/internal/game"
)

// protocolVersion is informational only (Q-FD-U3-13=B). Server does not
// validate the field on incoming messages; the client may inspect it via
// the welcome message.
const protocolVersion = "v1"

// Wire message type discriminators (BR-U3-WIRE-2).
//
// Incoming (client → server):
const (
	TypeHostCreateSession  = "host:create-session"
	TypeJoin               = "join"
	TypeResume             = "resume"
	TypeHostStart          = "host:start"
	TypeSubmitAdvanceIntro = "submit:advance-intro"
	TypeSubmitMafiaKill    = "submit:mafia-kill"
	TypeSubmitDoctorHeal   = "submit:doctor-heal"
	TypeSubmitPoliceCheck  = "submit:police-check"
	TypeSubmitEndNight     = "submit:end-night"
	TypeSubmitEndDiscuss   = "submit:end-discussion"
	TypeSubmitVote         = "submit:vote"
	TypeHostToggleVoice    = "host:toggle-voice"
	TypeHostForceEnd       = "host:force-end"
	TypeSubscribePublic    = "subscribe-public"

	// Iteration 2 (FR-9 / FR-10 / FR-11 / FR-12): GM-only flow.
	TypeHostClaim         = "host:claim"
	TypeHostOpenRoom      = "host:open-room"
	TypeHostStartRoom     = "host:start-room"
	TypeHostTerminateRoom = "host:terminate-room"
	TypeHostCloseRoom     = "host:close-room"
	TypePlayerEndSelfIntro = "player:end-self-intro"

	// Iteration 5 (R4): host pause/resume controls.
	TypeHostPause  = "host:pause"
	TypeHostResume = "host:resume"
)

// Outgoing (server → client):
const (
	TypeWelcome  = "welcome"
	TypeJoined   = "joined"
	TypeSnapshot = "snapshot"
	TypeEvent    = "event"
	TypeAnnounce = "announce"
	TypeError    = "error"

	// Iteration 2 outbound.
	TypeHostTokenOut    = "host-token"
	TypeRoomOpened      = "room:opened"
	TypeRoomHostOccupied = "room:host-occupied"
	TypeRoomClosed      = "room:closed"
)

// incomingEnvelope is the lightweight wrapper used to peek at the type
// field before unmarshaling the full payload. The full payload is then
// re-decoded into a type-specific struct using json.Unmarshal on the
// original bytes.
type incomingEnvelope struct {
	Type string `json:"type"`
}

// Incoming payloads — only the fields we actually consume are declared.
// Unknown fields are silently ignored by encoding/json default decoder.

type joinPayload struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type resumePayload struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type hostCreateSessionPayload struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type hostStartPayload struct {
	Type    string       `json:"type"`
	Options game.Options `json:"options"`
}

type targetPayload struct {
	Type   string        `json:"type"`
	Target game.PlayerID `json:"target"`
}

type voiceTogglePayload struct {
	Type string `json:"type"`
	On   bool   `json:"on"`
}

// Outgoing messages.

type welcomeMsg struct {
	Type            string   `json:"type"`
	ClientID        ClientID `json:"clientId"`
	Kind            string   `json:"kind"`
	ProtocolVersion string   `json:"protocolVersion"`
}

type joinedMsg struct {
	Type     string        `json:"type"`
	PlayerID game.PlayerID `json:"playerId"`
	Token    string        `json:"token"`
	IsHost   bool          `json:"isHost"`
}

type snapshotMsg struct {
	Type   string     `json:"type"`
	State  game.State `json:"state"`
	Your   yourInfo   `json:"your"`
	IsHost bool       `json:"isHost"`
}

type yourInfo struct {
	Role        game.Role       `json:"role,omitempty"`
	Keyword     string          `json:"keyword,omitempty"`
	Team        game.Team       `json:"team,omitempty"`
	MafiaCohort []game.PlayerID `json:"mafiaCohort,omitempty"`
}

type eventMsg struct {
	Type       string       `json:"type"`
	Visibility string       `json:"visibility"`
	Event      eventPayload `json:"event"`
}

// eventPayload carries one of the 16 game.Event variants. `Kind` selects
// which fields are populated; non-applicable fields use omitempty.
type eventPayload struct {
	Kind string `json:"kind"`

	// Common fields used by multiple kinds.
	PlayerID         game.PlayerID         `json:"playerId,omitempty"`
	Name             string                `json:"name,omitempty"`
	Target           game.PlayerID         `json:"target,omitempty"`
	Role             game.Role             `json:"role,omitempty"`
	Keyword          string                `json:"keyword,omitempty"`
	Team             game.Team             `json:"team,omitempty"`
	MafiaIDs         []game.PlayerID       `json:"mafiaIds,omitempty"`
	RepresentativeID game.PlayerID         `json:"representativeId,omitempty"`
	Police           game.PlayerID         `json:"police,omitempty"`
	Phase            game.Phase            `json:"phase,omitempty"`
	Day              int                   `json:"day,omitempty"`
	DeadlineMs       int64                 `json:"deadlineMs,omitempty"`
	SecondsLeft      int                   `json:"secondsLeft,omitempty"`
	Victim           game.PlayerID         `json:"victim,omitempty"`
	Counts           map[game.PlayerID]int `json:"counts,omitempty"`
	Eliminated       *game.PlayerID        `json:"eliminated,omitempty"`
	Recount          bool                  `json:"recount,omitempty"`
	OldID            game.PlayerID         `json:"oldId,omitempty"`
	NewID            game.PlayerID         `json:"newId,omitempty"`
	Winner           *game.Team            `json:"winner,omitempty"`
	EndReason        game.EndReason        `json:"endReason,omitempty"`
	Reveal           []game.Player         `json:"reveal,omitempty"`
	On               *bool                 `json:"on,omitempty"`
	Step             game.NightStep        `json:"step,omitempty"`
	// Iteration 5 — NightStepChanged carries the freshly computed
	// wall-clock deadline so the public timer bar can render a
	// synchronized countdown. GamePaused/GameResumed reuse Phase.
	StepDeadlineMs int64 `json:"stepDeadlineMs,omitempty"`
}

type announceMsg struct {
	Type     string `json:"type"`
	Subtitle string `json:"subtitle"`
	AudioID  string `json:"audioId,omitempty"`
	Severity string `json:"severity"`
}

type errorMsg struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Iteration 2 messages.

type hostOpenRoomPayload struct {
	Type    string       `json:"type"`
	Options game.Options `json:"options"`
}

type hostTokenMsg struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type roomOpenedMsg struct {
	Type    string       `json:"type"`
	Options game.Options `json:"options"`
}

type roomHostOccupiedMsg struct {
	Type string `json:"type"`
}

type roomClosedMsg struct {
	Type string `json:"type"`
}

// mustMarshal is the JSON encoder used for all outgoing wire messages.
// Any non-marshalable value is a programmer bug — we panic rather than
// return an error so the test suite catches it immediately.
func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("ws: failed to marshal wire message: " + err.Error())
	}
	return b
}

// visibilityToString maps game.Visibility to the wire protocol's string
// representation. Unknown values are encoded as "UNKNOWN" so a client
// can detect protocol drift instead of silently mis-routing.
func visibilityToString(v game.Visibility) string {
	switch v {
	case game.VisPublic:
		return "PUBLIC"
	case game.VisPlayer:
		return "PLAYER"
	case game.VisRoleMafia:
		return "ROLE_MAFIA"
	default:
		return "UNKNOWN"
	}
}
