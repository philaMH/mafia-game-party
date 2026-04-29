package session

import (
	"context"
	"log/slog"
	"time"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

// SubmitAction forwards an Action to the underlying engine, persists when
// trigger events fire (BR-U2-PERSIST-1/2), and dispatches rendered
// announcements to subscribers. On engine error the state is unchanged
// (NFR-U1-R2) and a single error announcement is returned to the sender.
func (s *session) SubmitAction(ctx context.Context, action game.Action) ([]EventOut, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.sess.Started {
		return nil, &game.EngineError{Code: game.CodeWrongPhase, Message: "game not started"}
	}

	state, envs, err := s.engine.Apply(action)
	if err != nil {
		errAnn := s.catalog.RenderError(err, senderOf(action), s.catalogContext())
		out := EventOut{}
		if !errAnn.IsEmpty() {
			ea := errAnn
			out.Announcement = &ea
		}
		return []EventOut{out}, err
	}
	return s.persistAndDispatch(ctx, state, envs), nil
}

// persistAndDispatch is the heart of U2: it renders catalog announcements,
// persists when triggers fire (BR-U2-PERSIST-1/2/3), and notifies
// Subscribe handlers. Caller must hold s.mu.
func (s *session) persistAndDispatch(ctx context.Context, state game.State, envs []game.EventEnvelope) []EventOut {
	outs := make([]EventOut, 0, len(envs)+len(s.systemMsg))

	// Drain queued system announcements (e.g., post-restore notice) first.
	for _, sysAnn := range s.systemMsg {
		ann := sysAnn
		outs = append(outs, EventOut{Announcement: &ann, State: state})
	}
	s.systemMsg = nil

	cctx := s.catalogContext()
	for _, env := range envs {
		out := EventOut{Envelope: env, State: state}
		ann := s.catalog.Render(env, cctx)
		if !ann.IsEmpty() {
			a := ann
			out.Announcement = &a
		}
		outs = append(outs, out)
	}

	// Optional event log (off by default).
	if s.opts.EventLog {
		for _, env := range envs {
			if err := s.persistence.AppendEvent(ctx, s.sess.GameID, env); err != nil {
				slog.Warn("append event failed", "err", err)
			}
		}
	}

	if shouldPersist(envs) {
		snap := persistence.Snapshot{
			GameID:  s.sess.GameID,
			State:   state,
			Members: persistedMembers(s.sess.Members),
			HostID:  s.sess.HostID,
		}
		if err := s.persistence.SaveSnapshot(ctx, snap); err != nil {
			slog.Error("save snapshot failed", "err", err, "game_id", s.sess.GameID)
			pf := announce.SystemPersistFailure()
			outs = append(outs, EventOut{Announcement: &pf})
		}
	}

	if g, end := findGameEnded(envs); end {
		s.handleGameEnd(ctx, g, state)
	}

	s.dispatchHandlers(outs)
	return outs
}

// shouldPersist returns true if any event in envs is a persist trigger
// (PhaseChanged, DeathAnnounced, Eliminated, GameEnded, MafiaRepresentativeReassigned).
func shouldPersist(envs []game.EventEnvelope) bool {
	for _, env := range envs {
		switch env.Event.(type) {
		case game.PhaseChanged, game.DeathAnnounced, game.Eliminated,
			game.GameEnded, game.MafiaRepresentativeReassigned:
			return true
		}
	}
	return false
}

// findGameEnded returns the GameEnded event (if any) in envs.
func findGameEnded(envs []game.EventEnvelope) (game.GameEnded, bool) {
	for _, env := range envs {
		if g, ok := env.Event.(game.GameEnded); ok {
			return g, true
		}
	}
	return game.GameEnded{}, false
}

// handleGameEnd writes the final result and clears the active snapshot in a
// single transaction (BR-U2-PERSIST-3). On error we log and continue;
// next boot's restore path will detect the lingering active snapshot and
// auto-finalize via BR-U2-RESTORE-6.
func (s *session) handleGameEnd(ctx context.Context, g game.GameEnded, state game.State) {
	result := persistence.GameResult{
		GameID:    s.sess.GameID,
		StartedAt: s.sess.StartedAt,
		EndedAt:   s.clock.Now(),
		Winner:    g.Winner,
		EndReason: g.EndReason,
		Options:   state.Settings,
		Members:   persistedMembers(s.sess.Members),
		Reveal:    g.Reveal,
	}
	if err := s.persistence.SaveResultAndClearActive(ctx, result); err != nil {
		slog.Error("save result failed", "err", err, "game_id", s.sess.GameID)
		pf := announce.SystemPersistFailure()
		// queue for next dispatch since we are mid-loop
		s.systemMsg = append(s.systemMsg, pf)
		return
	}
	s.sess.Started = false
}

// buildResultFromState reconstructs a GameResult from a restored end-state.
// Used by BR-U2-RESTORE-6.
func buildResultFromState(sess Session, st game.State) persistence.GameResult {
	winner := st.Winner
	endReason := game.EndHostForceEnd
	if st.EndReason != nil {
		endReason = *st.EndReason
	}
	reveal := make([]game.Player, len(st.Players))
	copy(reveal, st.Players)
	return persistence.GameResult{
		GameID:    sess.GameID,
		StartedAt: sess.StartedAt,
		EndedAt:   time.Now(),
		Winner:    winner,
		EndReason: endReason,
		Options:   st.Settings,
		Members:   persistedMembers(sess.Members),
		Reveal:    reveal,
	}
}

// catalogContext snapshots session-derived lookups for catalog rendering.
// Caller must hold s.mu.
func (s *session) catalogContext() announce.CatalogContext {
	memberNames := make(map[game.PlayerID]string, len(s.sess.Members))
	for id, m := range s.sess.Members {
		memberNames[id] = m.Name
	}
	state := s.engine.Snapshot()
	return announce.CatalogContext{
		GetName: func(id game.PlayerID) string {
			if n, ok := memberNames[id]; ok {
				return n
			}
			// Fallback: lookup in engine Players (post-Start the engine
			// is the authoritative source of names).
			if p, ok := state.FindPlayer(id); ok {
				return p.Name
			}
			return string(id)
		},
		IntroSecondsPerPlayer: state.Settings.IntroSecondsPerPlayer,
	}
}

// dispatchHandlers invokes each subscriber once per EventOut. Each call is
// wrapped in defer-recover to satisfy P-U2-4 (one panicking handler must
// not bring down the manager).
func (s *session) dispatchHandlers(outs []EventOut) {
	for _, h := range s.handlers {
		for _, out := range outs {
			s.callHandler(h.fn, out)
		}
	}
}

func (s *session) callHandler(fn EventHandler, out EventOut) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("event handler panicked", "panic", r)
		}
	}()
	fn(out)
}

// senderOf attempts to recover the originating PlayerID from a typed
// Action. The result is informational (used when rendering errors); when
// the action is host-issued or has no obvious sender, the empty PlayerID is
// returned and the catalog falls back gracefully.
func senderOf(a game.Action) game.PlayerID {
	switch v := a.(type) {
	case game.SubmitMafiaKill:
		return v.Mafia
	case game.SubmitDoctorHeal:
		return v.Doctor
	case game.SubmitPoliceCheck:
		return v.Police
	case game.SubmitVote:
		return v.Voter
	case game.StartGame:
		return v.HostID
	case game.AdvanceIntro:
		return v.HostID
	case game.EndSelfIntro:
		return v.PlayerID
	case game.EndNightEarly:
		return v.HostID
	case game.EndDiscussionEarly:
		return v.HostID
	case game.ToggleVoice:
		return v.HostID
	case game.ForceEndGame:
		return v.HostID
	case game.PauseGame:
		return v.HostID
	case game.ResumeGame:
		return v.HostID
	default:
		return ""
	}
}
