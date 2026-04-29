package ws

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/session"
)

// readDeadline bounds a quiet client; pongs reset it (BR-U3-HEARTBEAT-1).
const readDeadline = 30 * time.Second

// readLimit caps a single inbound message at 64 KiB (NFR-U3-S3).
const readLimit = 64 << 10

// readLoop pumps inbound messages from a single client until the
// connection drops, then ensures the client is unregistered. All
// session-affecting work is funneled through handleIncoming.
func (h *hub) readLoop(c *Client) {
	defer func() {
		// Iteration 2: surrender the GM seat on disconnect so the next
		// /public connection can become host.
		if c.HostToken != "" {
			h.mgr.ReleaseHost(c.HostToken)
			c.HostToken = ""
		}
		h.Unregister(c.ID)
	}()

	c.Conn.SetReadLimit(readLimit)
	_ = c.Conn.SetReadDeadline(time.Now().Add(readDeadline))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			h.log.Debug("ws read end", "client", c.ID, "err", err)
			return
		}

		var env incomingEnvelope
		if jerr := json.Unmarshal(raw, &env); jerr != nil || env.Type == "" {
			h.sendError(c, "VALIDATION_ERROR", "invalid message")
			continue
		}
		h.log.Debug("ws incoming", "client", c.ID, "type", env.Type)
		h.handleIncoming(c, env.Type, raw)
	}
}

// handleIncoming dispatches a single message to the appropriate
// SessionManager call. The raw bytes are kept so each branch can
// re-decode into a typed struct without an extra encoding round trip.
func (h *hub) handleIncoming(c *Client, typ string, raw []byte) {
	ctx := h.rootCtx

	switch typ {
	case TypeHostCreateSession:
		var p hostCreateSessionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		jr, err := h.mgr.CreateSession(ctx, p.Name)
		h.respondJoin(c, jr, err)

	case TypeJoin:
		var p joinPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		jr, err := h.mgr.JoinPlayer(ctx, p.Name)
		h.respondJoin(c, jr, err)

	case TypeResume:
		var p resumePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		jr, err := h.mgr.ResumePlayer(ctx, p.Token)
		h.respondJoin(c, jr, err)
		// Resume includes the immediate-snapshot push (NFR-U3-R3).
		if err == nil {
			h.enqueue(c, mustMarshal(snapshotMsg{
				Type:   TypeSnapshot,
				State:  jr.CurrentState,
				IsHost: jr.IsHost,
				Your: yourInfo{
					Role:        jr.YourRole,
					Keyword:     jr.YourKeyword,
					Team:        jr.YourTeam,
					MafiaCohort: jr.MafiaCohort,
				},
			}))
		}

	case TypeHostStart:
		var p hostStartPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		_, err := h.mgr.StartGame(ctx, c.PlayerID, p.Options)
		h.handleSubmitErr(c, err)

	case TypeSubmitAdvanceIntro:
		_, err := h.mgr.SubmitAction(ctx, game.AdvanceIntro{HostID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeSubmitMafiaKill:
		var p targetPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		_, err := h.mgr.SubmitAction(ctx, game.SubmitMafiaKill{Mafia: c.PlayerID, Target: p.Target})
		h.handleSubmitErr(c, err)

	case TypeSubmitDoctorHeal:
		var p targetPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		_, err := h.mgr.SubmitAction(ctx, game.SubmitDoctorHeal{Doctor: c.PlayerID, Target: p.Target})
		h.handleSubmitErr(c, err)

	case TypeSubmitPoliceCheck:
		var p targetPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		_, err := h.mgr.SubmitAction(ctx, game.SubmitPoliceCheck{Police: c.PlayerID, Target: p.Target})
		h.handleSubmitErr(c, err)

	case TypeSubmitEndNight:
		_, err := h.mgr.SubmitAction(ctx, game.EndNightEarly{HostID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeSubmitEndDiscuss:
		_, err := h.mgr.SubmitAction(ctx, game.EndDiscussionEarly{HostID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeSubmitVote:
		var p targetPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		_, err := h.mgr.SubmitAction(ctx, game.SubmitVote{Voter: c.PlayerID, Target: p.Target})
		h.handleSubmitErr(c, err)

	case TypeHostToggleVoice:
		var p voiceTogglePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		_, err := h.mgr.SubmitAction(ctx, game.ToggleVoice{HostID: c.PlayerID, On: p.On})
		h.handleSubmitErr(c, err)

	case TypeHostForceEnd:
		_, err := h.mgr.SubmitAction(ctx, game.ForceEndGame{HostID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeHostClaim:
		token, err := h.mgr.ClaimHost(ctx)
		if err != nil {
			h.enqueue(c, mustMarshal(roomHostOccupiedMsg{Type: TypeRoomHostOccupied}))
			return
		}
		c.HostToken = token
		h.enqueue(c, mustMarshal(hostTokenMsg{Type: TypeHostTokenOut, Token: string(token)}))

	case TypeHostOpenRoom:
		var p hostOpenRoomPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			h.sendError(c, "VALIDATION_ERROR", "bad payload")
			return
		}
		state, err := h.mgr.OpenRoom(ctx, c.HostToken, p.Options)
		if err != nil {
			h.handleSubmitErr(c, err)
			return
		}
		_ = state
		// Broadcast room:opened to all clients (PUBLIC + PLAYER) so /play
		// gating screens can advance to the join form.
		h.broadcastRoomOpened(p.Options)

	case TypeHostStartRoom:
		_, err := h.mgr.HostStartGame(ctx, c.HostToken)
		h.handleSubmitErr(c, err)

	case TypeHostTerminateRoom:
		_, err := h.mgr.HostForceTerminate(ctx, c.HostToken)
		h.handleSubmitErr(c, err)

	case TypeHostCloseRoom:
		if err := h.mgr.HostCloseRoom(ctx, c.HostToken); err != nil {
			h.handleSubmitErr(c, err)
			return
		}
		// Broadcast first so clients reset their UI immediately, then
		// drop the per-player binding so a stale playerId index does
		// not block the next round of joins on the same connection.
		h.broadcastRoomClosed()

	case TypePlayerEndSelfIntro:
		_, err := h.mgr.SubmitAction(ctx, game.EndSelfIntro{PlayerID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeHostPause:
		_, err := h.mgr.SubmitAction(ctx, game.PauseGame{HostID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeHostResume:
		_, err := h.mgr.SubmitAction(ctx, game.ResumeGame{HostID: c.PlayerID})
		h.handleSubmitErr(c, err)

	case TypeSubscribePublic:
		// PUBLIC clients are already auto-subscribed; this is a no-op
		// kept for protocol completeness.

	default:
		h.sendError(c, "VALIDATION_ERROR", "unknown message type: "+typ)
	}
}

// respondJoin handles the success/error path for join, resume, and
// host:create-session. On success it indexes the PlayerID and pushes a
// `joined` envelope. On error it forwards the engine error to the
// caller (they remain a PUBLIC client until they retry).
func (h *hub) respondJoin(c *Client, jr session.JoinResult, err error) {
	if err != nil {
		h.handleSubmitErr(c, err)
		return
	}
	h.bindPlayer(c, jr.PlayerID)
	h.enqueue(c, mustMarshal(joinedMsg{
		Type:     TypeJoined,
		PlayerID: jr.PlayerID,
		Token:    jr.Token,
		IsHost:   jr.IsHost,
	}))
}

// bindPlayer transitions c into PLAYER and evicts any prior client
// that was indexed under the same PlayerID (last-connect-wins).
func (h *hub) bindPlayer(c *Client, pid game.PlayerID) {
	oldID, hadOld := h.registry.bindPlayer(c, pid)
	if hadOld {
		h.log.Info("ws evicting prior client (last-connect-wins)",
			"old", oldID, "new", c.ID, "playerId", pid)
		h.Unregister(oldID)
	}
}

// handleSubmitErr translates a SessionManager error to wire messages.
// It always emits a `error` frame for the offending client, and may
// additionally emit an `announce` carrying the catalog's localized
// message when one was provided (BR-U3-ERR-4).
func (h *hub) handleSubmitErr(c *Client, err error) {
	if err == nil {
		return
	}
	// Localized announce — best effort. Use the announce package's
	// default catalog (the same instance the SessionManager uses for
	// VisPublic announces) so player-private messages stay private to
	// this client.
	cat := announce.NewDefaultCatalog()
	ann := cat.RenderError(err, c.PlayerID, announce.CatalogContext{})
	if !ann.IsEmpty() {
		h.enqueue(c, mustMarshal(announceMsg{
			Type:     TypeAnnounce,
			Subtitle: ann.Subtitle,
			AudioID:  ann.AudioID,
			Severity: string(ann.Severity),
		}))
	}
	h.sendError(c, errorCodeOf(err), err.Error())
}

// sendError pushes a single `error` envelope to the named client only.
func (h *hub) sendError(c *Client, code, message string) {
	h.enqueue(c, mustMarshal(errorMsg{
		Type:    TypeError,
		Code:    code,
		Message: message,
	}))
}

// errorCodeOf extracts a wire error code from any error returned by
// SessionManager. Engine errors carry a typed Code we forward verbatim
// (BR-U3-ERR-1); other errors fall back to a generic "INTERNAL".
func errorCodeOf(err error) string {
	if err == nil {
		return ""
	}
	var ee *game.EngineError
	if errors.As(err, &ee) {
		return string(ee.Code)
	}
	return "INTERNAL"
}

// Compile-time guard to keep the gorilla import necessary even when
// readLoop's dependencies shift in the future.
var _ = websocket.TextMessage
var _ = context.Background
