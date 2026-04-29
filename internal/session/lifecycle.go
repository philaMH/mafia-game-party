package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

// newID returns a 16-char (8-byte) random hex identifier suitable for
// PlayerID / GameID. crypto/rand is used directly so the public API has
// no third-party dependency beyond modernc.org/sqlite (NFR-U2-M6).
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read failures are extremely rare
	return hex.EncodeToString(b[:])
}

// CreateSession initializes a new lobby with the host as its first member.
// It refuses to overwrite an in-progress game.
func (s *session) CreateSession(ctx context.Context, hostName string) (JoinResult, error) {
	if hostName == "" {
		return JoinResult{}, &game.EngineError{Code: game.CodeValidation, Message: "host name required", Field: "hostName"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sess.Started {
		return JoinResult{}, &game.EngineError{Code: game.CodeWrongPhase, Message: "game already in progress"}
	}
	if len(s.sess.Members) > 0 {
		return JoinResult{}, &game.EngineError{Code: game.CodeWrongPhase, Message: "active lobby exists; resume or end first"}
	}

	hostID := game.PlayerID(newID())
	tok, err := s.issueUniqueToken()
	if err != nil {
		return JoinResult{}, fmt.Errorf("issue token: %w", err)
	}

	s.sess.GameID = newID()
	s.sess.HostID = hostID
	s.sess.Members[hostID] = &Member{
		ID:        hostID,
		Name:      hostName,
		Token:     tok,
		Connected: true,
		JoinedAt:  s.clock.Now(),
	}

	lobby := lobbyStateFromMembers(s.sess.GameID, hostID, s.sess.Members)
	envs := []game.EventEnvelope{{
		Event:      game.PlayerJoined{PlayerID: hostID, Name: hostName},
		Visibility: game.VisPublic,
	}}
	s.persistAndDispatch(ctx, lobby, envs)

	return JoinResult{
		PlayerID:     hostID,
		Token:        tok,
		IsHost:       true,
		CurrentState: lobby,
	}, nil
}

// JoinPlayer admits a new player to the LOBBY. Game-already-started, lobby
// full, and duplicate-name rejections all surface as EngineError.
func (s *session) JoinPlayer(ctx context.Context, name string) (JoinResult, error) {
	if name == "" {
		return JoinResult{}, &game.EngineError{Code: game.CodeValidation, Message: "name required", Field: "name"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sess.Started {
		return JoinResult{}, &game.EngineError{Code: game.CodeWrongPhase, Message: "joining requires LOBBY"}
	}
	if len(s.sess.Members) == 0 && !s.sess.RoomOpened {
		return JoinResult{}, &game.EngineError{Code: game.CodeWrongPhase, Message: "no active lobby; host must open the room first"}
	}
	if len(s.sess.Members) >= s.opts.MaxLobbySize {
		return JoinResult{}, &game.EngineError{Code: game.CodeValidation, Message: "lobby is full", Field: "lobby"}
	}
	for _, m := range s.sess.Members {
		if m.Name == name {
			return JoinResult{}, &game.EngineError{Code: game.CodeValidation, Message: "name already taken", Field: "name"}
		}
	}

	pid := game.PlayerID(newID())
	tok, err := s.issueUniqueToken()
	if err != nil {
		return JoinResult{}, fmt.Errorf("issue token: %w", err)
	}
	s.sess.Members[pid] = &Member{
		ID:        pid,
		Name:      name,
		Token:     tok,
		Connected: true,
		JoinedAt:  s.clock.Now(),
	}

	lobby := lobbyStateFromMembers(s.sess.GameID, s.sess.HostID, s.sess.Members)
	envs := []game.EventEnvelope{{
		Event:      game.PlayerJoined{PlayerID: pid, Name: name},
		Visibility: game.VisPublic,
	}}
	s.persistAndDispatch(ctx, lobby, envs)

	return JoinResult{
		PlayerID:     pid,
		Token:        tok,
		IsHost:       false,
		CurrentState: lobby,
	}, nil
}

// ResumePlayer reconnects a player by token. It is allowed at any phase.
func (s *session) ResumePlayer(ctx context.Context, token string) (JoinResult, error) {
	if token == "" {
		return JoinResult{}, &game.EngineError{Code: game.CodeValidation, Message: "token required", Field: "token"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var member *Member
	for _, m := range s.sess.Members {
		if m.Token == token {
			member = m
			break
		}
	}
	if member == nil {
		return JoinResult{}, &game.EngineError{Code: game.CodeUnknownPlayer, Message: "invalid token"}
	}
	member.Connected = true

	state := s.engine.Snapshot()
	view := BuildPrivateView(state, member.ID, s.sess.HostID)

	return JoinResult{
		PlayerID:     member.ID,
		Token:        token,
		IsHost:       member.ID == s.sess.HostID,
		CurrentState: view.State,
		YourRole:     view.YourRole,
		YourKeyword:  view.YourKeyword,
		YourTeam:     view.YourTeam,
		MafiaCohort:  view.MafiaCohort,
	}, nil
}

// StartGame transitions the lobby into an active game. Host-only.
func (s *session) StartGame(ctx context.Context, hostID game.PlayerID, opts game.Options) ([]EventOut, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sess.Started {
		return nil, &game.EngineError{Code: game.CodeWrongPhase, Message: "game already started"}
	}
	if hostID != s.sess.HostID {
		return nil, &game.EngineError{Code: game.CodePermissionDenied, Message: "only host may start"}
	}
	if len(s.sess.Members) < s.opts.MinPlayers {
		return nil, &game.EngineError{
			Code:    game.CodeValidation,
			Message: fmt.Sprintf("need at least %d players to start", s.opts.MinPlayers),
			Field:   "members",
		}
	}

	players := make([]game.Player, 0, len(s.sess.Members))
	// Iterate in insertion-friendly order: host first, then by JoinedAt.
	ordered := orderMembers(s.sess.Members, s.sess.HostID)
	for _, m := range ordered {
		players = append(players, game.Player{
			ID:    m.ID,
			Name:  m.Name,
			Alive: true,
		})
	}

	state, envs, err := s.engine.Start(s.sess.GameID, hostID, players, opts)
	if err != nil {
		return nil, err
	}
	s.sess.Started = true
	s.sess.StartedAt = state.StartedAt

	return s.persistAndDispatch(ctx, state, envs), nil
}

// lobbyStateFromMembers builds a LOBBY-phase snapshot from the current
// member map. Each member appears as an alive Player with empty Role —
// LOBBY membership is a domain fact (PlayerJoined events) but secret roles
// are assigned only at StartGame. The returned State is used both for
// JoinResult.CurrentState and as the EventOut.State carried alongside
// PlayerJoined envelopes through persistAndDispatch.
func lobbyStateFromMembers(gameID string, hostID game.PlayerID, members map[game.PlayerID]*Member) game.State {
	ordered := orderMembers(members, hostID)
	players := make([]game.Player, 0, len(ordered))
	for _, m := range ordered {
		players = append(players, game.Player{ID: m.ID, Name: m.Name, Alive: true})
	}
	return game.State{
		GameID:  gameID,
		Phase:   game.PhaseLobby,
		HostID:  hostID,
		Players: players,
		Votes:   map[game.PlayerID]game.PlayerID{},
	}
}

// persistedMembers serializes the in-memory map to the persistence type
// (avoids exposing session.Member through the persistence.Snapshot edge).
func persistedMembers(m map[game.PlayerID]*Member) []persistence.PersistedMember {
	out := make([]persistence.PersistedMember, 0, len(m))
	for _, mem := range m {
		out = append(out, persistence.PersistedMember{
			ID:        mem.ID,
			Name:      mem.Name,
			Token:     mem.Token,
			Connected: mem.Connected,
			JoinedAt:  mem.JoinedAt,
		})
	}
	return out
}

// ClaimHost grants the GM seat to the first /public connection.
// (Iteration 2: FR-9.2 / FR-10.2)
func (s *session) ClaimHost(ctx context.Context) (HostToken, error) {
	_ = ctx
	return s.hostAuth.Claim()
}

// ReleaseHost surrenders the GM seat (called when the host's WS closes).
func (s *session) ReleaseHost(token HostToken) { s.hostAuth.Release(token) }

// OpenRoom transitions the room from Idle to Opened. The host enters game
// settings (max players, mafia count) and then explicitly opens the room
// for player joins. The host is NOT added as a member (FR-9.1).
//
// Returns the resulting LOBBY State (Players=empty until joins arrive).
func (s *session) OpenRoom(ctx context.Context, token HostToken, opts game.Options) (game.State, error) {
	if err := s.hostAuth.Verify(token); err != nil {
		return game.State{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sess.Started {
		return game.State{}, &game.EngineError{Code: game.CodeWrongPhase, Message: "game already in progress"}
	}
	if len(s.sess.Members) > 0 {
		return game.State{}, &game.EngineError{Code: game.CodeWrongPhase, Message: "active lobby exists; end first"}
	}

	s.sess.GameID = newID()
	s.sess.HostID = "" // GM seat is decoupled from any PlayerID under the v2 flow.
	s.sess.Members = make(map[game.PlayerID]*Member)
	s.sess.PendingOptions = opts
	s.sess.RoomOpened = true

	lobby := lobbyStateFromMembers(s.sess.GameID, s.sess.HostID, s.sess.Members)
	s.persistAndDispatch(ctx, lobby, nil)
	return lobby, nil
}

// HostStartGame triggers the LOBBY -> INTRO transition under the v2 GM flow.
// The host is NOT in players (host=="" passed to Engine.Start).
func (s *session) HostStartGame(ctx context.Context, token HostToken) ([]EventOut, error) {
	if err := s.hostAuth.Verify(token); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sess.Started {
		return nil, &game.EngineError{Code: game.CodeWrongPhase, Message: "game already started"}
	}
	if !s.sess.RoomOpened {
		return nil, &game.EngineError{Code: game.CodeWrongPhase, Message: "room not opened"}
	}
	if len(s.sess.Members) < s.opts.MinPlayers {
		return nil, &game.EngineError{
			Code:    game.CodeValidation,
			Message: fmt.Sprintf("need at least %d players to start", s.opts.MinPlayers),
			Field:   "members",
		}
	}

	players := make([]game.Player, 0, len(s.sess.Members))
	ordered := orderMembers(s.sess.Members, "")
	for _, m := range ordered {
		players = append(players, game.Player{ID: m.ID, Name: m.Name, Alive: true})
	}

	state, envs, err := s.engine.Start(s.sess.GameID, "", players, s.sess.PendingOptions)
	if err != nil {
		return nil, err
	}
	s.sess.Started = true
	s.sess.StartedAt = state.StartedAt

	return s.persistAndDispatch(ctx, state, envs), nil
}

// HostCloseRoom resets the room and engine state so the host can open a
// fresh room. Allowed at any time after ClaimHost: if a game is in
// progress it is dropped (use HostForceTerminate first if a graceful
// END announcement is desired). All player tokens become invalid; the
// caller (U3) is responsible for broadcasting room:closed and unbinding
// any per-player client state.
func (s *session) HostCloseRoom(ctx context.Context, token HostToken) error {
	_ = ctx
	if err := s.hostAuth.Verify(token); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Restore the engine to a clean LOBBY-ready state. Engine.Start
	// requires an empty or LOBBY phase, so we cannot leave the engine
	// holding the END snapshot from the previous game.
	if err := s.engine.Restore(game.State{
		Phase:   game.PhaseLobby,
		Players: []game.Player{},
		Votes:   map[game.PlayerID]game.PlayerID{},
	}); err != nil {
		return err
	}

	s.sess = Session{Members: make(map[game.PlayerID]*Member)}

	return nil
}

// HostForceTerminate ends the active game with EndReason=HOST_FORCE_END.
// Under the v2 GM flow there is no host PlayerID; we synthesize a ForceEndGame
// action whose HostID matches state.HostID (empty), so the engine's host
// check is consistent.
func (s *session) HostForceTerminate(ctx context.Context, token HostToken) ([]EventOut, error) {
	if err := s.hostAuth.Verify(token); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.sess.Started {
		return nil, &game.EngineError{Code: game.CodeWrongPhase, Message: "no active game"}
	}
	state := s.engine.Snapshot()
	newState, envs, err := s.engine.Apply(game.ForceEndGame{HostID: state.HostID})
	if err != nil {
		return nil, err
	}
	return s.persistAndDispatch(ctx, newState, envs), nil
}

// orderMembers returns members with the host first, then in JoinedAt order.
func orderMembers(m map[game.PlayerID]*Member, hostID game.PlayerID) []*Member {
	out := make([]*Member, 0, len(m))
	if h, ok := m[hostID]; ok {
		out = append(out, h)
	}
	rest := make([]*Member, 0, len(m))
	for id, mem := range m {
		if id == hostID {
			continue
		}
		rest = append(rest, mem)
	}
	// Stable insertion sort by JoinedAt.
	for i := 1; i < len(rest); i++ {
		for j := i; j > 0 && rest[j-1].JoinedAt.After(rest[j].JoinedAt); j-- {
			rest[j-1], rest[j] = rest[j], rest[j-1]
		}
	}
	out = append(out, rest...)
	return out
}
