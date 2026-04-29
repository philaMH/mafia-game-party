package ws

import (
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

// broadcastRoomOpened sends a room:opened message to every connected
// client (PUBLIC and PLAYER). Used by the v2 GM flow when the host
// completes OpenRoom — gating screens then transition to the join form.
func (h *hub) broadcastRoomOpened(opts game.Options) {
	msg := mustMarshal(roomOpenedMsg{Type: TypeRoomOpened, Options: opts})
	for _, c := range h.registry.snapshotPublic() {
		h.enqueue(c, msg)
	}
	for _, c := range h.registry.snapshotPlayers() {
		h.enqueue(c, msg)
	}
}

// broadcastRoomClosed sends a room:closed message to every connected
// client. Players unbind their PlayerID and clear their token; the host
// retains its hostToken and returns to the OpenRoom configuration screen.
// PLAYER clients are demoted to PUBLIC so a stale playerId index does
// not prevent a fresh join after the new room opens.
func (h *hub) broadcastRoomClosed() {
	msg := mustMarshal(roomClosedMsg{Type: TypeRoomClosed})
	for _, c := range h.registry.snapshotPublic() {
		h.enqueue(c, msg)
	}
	for _, c := range h.registry.snapshotPlayers() {
		h.enqueue(c, msg)
		h.registry.unbindPlayer(c)
	}
}

// pushRoomState sends current room state to a single freshly registered
// client (Iteration 3 — late-joiner resync). Issued in send order:
// room:opened (so the reducer ungates the join form) → snapshot (LOBBY
// roster, in-progress game state, or END reveal) → room:host-occupied
// (so /public hides the claim form). The reducer treats each message
// as idempotent, so a race against broadcastRoomOpened that double-
// delivers room:opened is harmless.
//
// The snapshot is sent whenever the room is open: a refreshing host
// during LOBBY needs the roster + start button, and players/host who
// reconnect after GameEnded need the reveal screen.
func (h *hub) pushRoomState(c *Client, snap session.RoomSnapshot) {
	if snap.RoomOpened {
		h.enqueue(c, mustMarshal(roomOpenedMsg{Type: TypeRoomOpened, Options: snap.Options}))
		h.enqueue(c, mustMarshal(snapshotMsg{
			Type:   TypeSnapshot,
			State:  snap.State,
			IsHost: false,
			Your:   yourInfo{},
		}))
	}
	if snap.HostOccupied {
		h.enqueue(c, mustMarshal(roomHostOccupiedMsg{Type: TypeRoomHostOccupied}))
	}
}

// onEvent is the SessionManager Subscribe callback. It runs *inside* U2's
// GM lock, so it may not perform any blocking I/O. The implementation
// builds wire messages, picks targets via routeEvent, and enqueues — all
// non-blocking. Any panic is caught so a buggy upstream cannot deadlock
// the SessionManager (P-U3-5).
func (h *hub) onEvent(out session.EventOut) {
	defer func() {
		if r := recover(); r != nil {
			h.log.Error("ws onEvent panic", "panic", r)
		}
	}()

	if out.Envelope.Event != nil {
		msg := mustMarshal(eventMsg{
			Type:       TypeEvent,
			Visibility: visibilityToString(out.Envelope.Visibility),
			Event:      buildEventPayload(out.Envelope.Event),
		})
		for _, c := range h.routeEvent(out.Envelope, out.State) {
			h.enqueue(c, msg)
		}
	}

	if out.Announcement != nil && !out.Announcement.IsEmpty() && out.Announcement.ForPublicOnly {
		msg := mustMarshal(announceMsg{
			Type:     TypeAnnounce,
			Subtitle: out.Announcement.Subtitle,
			AudioID:  out.Announcement.AudioID,
			Severity: string(out.Announcement.Severity),
		})
		for _, c := range h.registry.snapshotPublic() {
			h.enqueue(c, msg)
		}
	}
}

// routeEvent maps an EventEnvelope's Visibility to the set of clients
// that should receive it. PUBLIC events fan out to every PUBLIC client
// AND every PLAYER (including the dead — Q-FD-U3-5=A). PLAYER events
// reach exactly one client; ROLE_MAFIA events reach every living mafia
// PlayerID's currently registered client. We rely on the State carried
// by EventOut (taken under the GM lock by U2) to avoid re-entering the
// session lock from inside a Subscribe handler.
func (h *hub) routeEvent(env game.EventEnvelope, state game.State) []*Client {
	switch env.Visibility {
	case game.VisPublic:
		out := h.registry.snapshotPublic()
		out = append(out, h.registry.snapshotPlayers()...)
		return out
	case game.VisPlayer:
		if c := h.registry.byPlayerSafe(env.PlayerID); c != nil {
			return []*Client{c}
		}
		return nil
	case game.VisRoleMafia:
		out := make([]*Client, 0, 3)
		for _, p := range state.Players {
			if p.Alive && p.Role == game.RoleMafia {
				if c := h.registry.byPlayerSafe(p.ID); c != nil {
					out = append(out, c)
				}
			}
		}
		return out
	}
	return nil
}

// buildEventPayload converts a game.Event into the wire eventPayload.
// Each Engine event becomes a single Kind discriminator plus its
// distinguishing fields. Unknown events serialize to {Kind: "Unknown"}
// so the client can detect protocol drift.
func buildEventPayload(ev game.Event) eventPayload {
	switch e := ev.(type) {
	case game.PlayerJoined:
		return eventPayload{
			Kind:     "PlayerJoined",
			PlayerID: e.PlayerID,
			Name:     e.Name,
		}
	case game.GameStarted:
		return eventPayload{Kind: "GameStarted"}
	case game.PhaseChanged:
		return eventPayload{
			Kind:       "PhaseChanged",
			Phase:      e.Phase,
			Day:        e.Day,
			DeadlineMs: timeToMs(e.Deadline.UnixMilli()),
		}
	case game.RoleRevealedToPlayer:
		return eventPayload{
			Kind:     "RoleRevealedToPlayer",
			PlayerID: e.PlayerID,
			Role:     e.Role,
			Keyword:  e.Keyword,
		}
	case game.MafiaCohortRevealed:
		return eventPayload{
			Kind:             "MafiaCohortRevealed",
			MafiaIDs:         e.MafiaIDs,
			RepresentativeID: e.RepresentativeID,
		}
	case game.IntroSpeakerChanged:
		return eventPayload{
			Kind:        "IntroSpeakerChanged",
			PlayerID:    e.PlayerID,
			SecondsLeft: e.SecondsLeft,
		}
	case game.MafiaTargetSelected:
		return eventPayload{
			Kind:             "MafiaTargetSelected",
			RepresentativeID: e.RepresentativeID,
			Target:           e.Target,
		}
	case game.PoliceResult:
		return eventPayload{
			Kind:   "PoliceResult",
			Police: e.Police,
			Target: e.Target,
			Team:   e.Team,
		}
	case game.DeathAnnounced:
		return eventPayload{
			Kind:   "DeathAnnounced",
			Victim: e.Victim,
		}
	case game.PeacefulNight:
		return eventPayload{Kind: "PeacefulNight"}
	case game.NightStepChanged:
		return eventPayload{
			Kind:           "NightStepChanged",
			Step:           e.Step,
			Day:            e.Day,
			StepDeadlineMs: timeToMs(e.Deadline.UnixMilli()),
		}
	case game.GamePaused:
		return eventPayload{
			Kind:  "GamePaused",
			Phase: e.Phase,
		}
	case game.GameResumed:
		return eventPayload{
			Kind:       "GameResumed",
			Phase:      e.Phase,
			DeadlineMs: timeToMs(e.Deadline.UnixMilli()),
		}
	case game.DiscussionTimerTick:
		return eventPayload{
			Kind:        "DiscussionTimerTick",
			SecondsLeft: e.SecondsLeft,
		}
	case game.VoteTallied:
		return eventPayload{
			Kind:       "VoteTallied",
			Counts:     e.Counts,
			Eliminated: e.Eliminated,
			Recount:    e.Recount,
		}
	case game.Eliminated:
		return eventPayload{
			Kind:     "Eliminated",
			PlayerID: e.PlayerID,
			Role:     e.Role,
		}
	case game.MafiaRepresentativeReassigned:
		return eventPayload{
			Kind:  "MafiaRepresentativeReassigned",
			OldID: e.OldID,
			NewID: e.NewID,
		}
	case game.GameEnded:
		return eventPayload{
			Kind:      "GameEnded",
			Winner:    e.Winner,
			EndReason: e.EndReason,
			Reveal:    e.Reveal,
		}
	case game.VoiceToggled:
		on := e.On
		return eventPayload{
			Kind: "VoiceToggled",
			On:   &on,
		}
	default:
		return eventPayload{Kind: "Unknown"}
	}
}

// timeToMs returns ms or 0 when the time is the zero value (avoid the
// epoch-1970 sentinel surfacing on the wire as a 0 value, which the
// client interprets as "no deadline").
func timeToMs(ms int64) int64 {
	if ms < 0 {
		return 0
	}
	return ms
}
